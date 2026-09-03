/*
This file is part of the KubeVirt project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Copyright The KubeVirt Authors.
*/

package tlsutil

import (
	"crypto/tls"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("TLSSecurityProfileToTLSConfig", func() {
	DescribeTable("named profiles set MinVersion and CipherSuites",
		func(profileType string, expectedMinVersion uint16, expectCiphers bool) {
			fn, err := TLSSecurityProfileToTLSConfig(profileType, "", "", "", GinkgoLogr)
			Expect(err).ToNot(HaveOccurred())
			Expect(fn).ToNot(BeNil())

			cfg := &tls.Config{}
			fn(cfg)

			Expect(cfg.MinVersion).To(Equal(expectedMinVersion))
			if expectCiphers {
				Expect(cfg.CipherSuites).ToNot(BeEmpty())
			} else {
				Expect(cfg.CipherSuites).To(BeEmpty())
			}
		},
		Entry("Old profile", "Old", uint16(tls.VersionTLS10), true),
		Entry("Intermediate profile", "Intermediate", uint16(tls.VersionTLS12), true),
		Entry("Modern profile (TLS 1.3 ciphers are not configurable in Go)", "Modern", uint16(tls.VersionTLS13), true),
	)

	It("should include insecure cipher suites from the Old profile", func() {
		fn, err := TLSSecurityProfileToTLSConfig("Old", "", "", "", GinkgoLogr)
		Expect(err).ToNot(HaveOccurred())

		cfg := &tls.Config{}
		fn(cfg)

		Expect(cfg.CipherSuites).To(ContainElement(tls.TLS_RSA_WITH_AES_128_GCM_SHA256))
		Expect(cfg.CipherSuites).To(ContainElement(tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA))
	})

	It("should configure Custom profile with valid min version, ciphers and groups", func() {
		fn, err := TLSSecurityProfileToTLSConfig(
			"Custom",
			"VersionTLS12",
			"ECDHE-RSA-AES128-GCM-SHA256,ECDHE-RSA-AES256-GCM-SHA384",
			"X25519MLKEM768,X25519",
			GinkgoLogr,
		)
		Expect(err).ToNot(HaveOccurred())
		Expect(fn).ToNot(BeNil())

		cfg := &tls.Config{}
		fn(cfg)

		Expect(cfg.MinVersion).To(Equal(uint16(tls.VersionTLS12)))
		Expect(cfg.CipherSuites).To(ConsistOf(
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		))
		Expect(cfg.CurvePreferences).To(ConsistOf(
			tls.X25519,
			tls.X25519MLKEM768,
		))
	})

	It("should trim whitespace from cipher and group names", func() {
		fn, err := TLSSecurityProfileToTLSConfig(
			"Custom",
			"VersionTLS12",
			" ECDHE-RSA-AES128-GCM-SHA256 , ECDHE-RSA-AES256-GCM-SHA384 ",
			" X25519MLKEM768 , X25519 ",
			GinkgoLogr,
		)
		Expect(err).ToNot(HaveOccurred())

		cfg := &tls.Config{}
		fn(cfg)

		Expect(cfg.CipherSuites).To(ConsistOf(
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		))

		Expect(cfg.CurvePreferences).To(ConsistOf(
			tls.X25519,
			tls.X25519MLKEM768,
		))
	})

	It("should skip unknown ciphers and succeed if some are valid", func() {
		fn, err := TLSSecurityProfileToTLSConfig(
			"Custom",
			"VersionTLS12",
			"ECDHE-RSA-AES128-GCM-SHA256,UNKNOWN-CIPHER",
			"X25519MLKEM768,UNKNOWN-GROUP",
			GinkgoLogr,
		)
		Expect(err).ToNot(HaveOccurred())

		cfg := &tls.Config{}
		fn(cfg)

		Expect(cfg.CipherSuites).To(ConsistOf(
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		))
		Expect(cfg.CurvePreferences).To(ConsistOf(
			tls.X25519MLKEM768,
		))
	})

	It("should return error when all ciphers are invalid", func() {
		Expect(TLSSecurityProfileToTLSConfig(
			"Custom",
			"VersionTLS12",
			"UNKNOWN-CIPHER-1,UNKNOWN-CIPHER-2",
			"X25519MLKEM768,X25519",
			GinkgoLogr,
		)).Error().To(MatchError(ContainSubstring("no valid ciphers")))
	})

	It("should return error when all groups are invalid", func() {
		Expect(TLSSecurityProfileToTLSConfig(
			"Custom",
			"VersionTLS12",
			"ECDHE-RSA-AES128-GCM-SHA256,ECDHE-RSA-AES256-GCM-SHA384",
			"UNKNOWN-GROUP-1,UNKNOWN-GROUP-2",
			GinkgoLogr,
		)).Error().To(MatchError(ContainSubstring("no valid groups")))
	})

	It("should return error when named profile has --tls-min-version set", func() {
		Expect(
			TLSSecurityProfileToTLSConfig("Intermediate", "VersionTLS13", "", "", GinkgoLogr),
		).Error().To(MatchError(ContainSubstring("only valid with")))
	})

	It("should return error when named profile has --tls-ciphers set", func() {
		Expect(
			TLSSecurityProfileToTLSConfig("Old", "", "ECDHE-RSA-AES128-GCM-SHA256", "", GinkgoLogr),
		).Error().To(MatchError(ContainSubstring("only valid with")))
	})

	It("should return error when named profile has --tls-groups set", func() {
		Expect(
			TLSSecurityProfileToTLSConfig("Old", "", "", "X25519MLKEM768,X25519", GinkgoLogr),
		).Error().To(MatchError(ContainSubstring("only valid with")))
	})

	It("should return error for unknown profile type", func() {
		Expect(
			TLSSecurityProfileToTLSConfig("InvalidProfile", "", "", "", GinkgoLogr),
		).Error().To(MatchError(ContainSubstring("unknown TLS security profile")))
	})

	It("should return error for Custom profile with missing min version", func() {
		Expect(
			TLSSecurityProfileToTLSConfig("Custom", "", "ECDHE-RSA-AES128-GCM-SHA256", "", GinkgoLogr),
		).Error().To(MatchError(ContainSubstring("tls-min-version")))
	})

	It("should return error for Custom profile with missing ciphers", func() {
		Expect(
			TLSSecurityProfileToTLSConfig("Custom", "VersionTLS12", "", "X25519MLKEM768,X25519", GinkgoLogr),
		).Error().To(MatchError(ContainSubstring("ciphers")))
	})

	It("should not return error for Custom profile with missing ciphers, if the minVersion is 1.3", func() {
		Expect(
			TLSSecurityProfileToTLSConfig("Custom", "VersionTLS13", "", "X25519MLKEM768,X25519", GinkgoLogr),
		).Error().ToNot(HaveOccurred())
	})

	It("should return error for Custom profile with missing groups", func() {
		Expect(
			TLSSecurityProfileToTLSConfig("Custom", "VersionTLS12", "ECDHE-RSA-AES128-GCM-SHA256", "", GinkgoLogr),
		).Error().To(MatchError(ContainSubstring("groups")))
	})

	It("should return error for Custom profile with whitespace-only ciphers", func() {
		Expect(
			TLSSecurityProfileToTLSConfig("Custom", "VersionTLS12", " , , ", "", GinkgoLogr),
		).Error().To(MatchError(ContainSubstring("ciphers")))
	})

	It("should return error for Custom profile with invalid min version", func() {
		Expect(TLSSecurityProfileToTLSConfig(
			"Custom",
			"VersionTLS99",
			"ECDHE-RSA-AES128-GCM-SHA256",
			"X25519MLKEM768,X25519",
			GinkgoLogr,
		)).Error().To(MatchError(ContainSubstring("unknown tls version")))
	})
})
