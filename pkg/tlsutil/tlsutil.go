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
	"fmt"
	"strings"

	configv1 "github.com/openshift/api/config/v1"
	ocpcrypto "github.com/openshift/library-go/pkg/crypto"
)

func TLSSecurityProfileToTLSConfig(profileType string, minVersion string, ciphers string) (func(*tls.Config), error) {
	var profileMinVersion string
	var profileCiphers []string

	tlsProfileType := configv1.TLSProfileType(profileType)

	switch tlsProfileType {
	case configv1.TLSProfileOldType, configv1.TLSProfileIntermediateType, configv1.TLSProfileModernType:
		if minVersion != "" || ciphers != "" {
			return nil, fmt.Errorf(
				"--tls-min-version and --tls-ciphers are only valid with --tls-security-profile=Custom",
			)
		}
		spec := configv1.TLSProfiles[tlsProfileType]
		profileMinVersion = string(spec.MinTLSVersion)
		profileCiphers = spec.Ciphers
	case configv1.TLSProfileCustomType:
		if minVersion == "" {
			return nil, fmt.Errorf("--tls-min-version is required when --tls-security-profile=Custom")
		}
		if ciphers == "" {
			return nil, fmt.Errorf("--tls-ciphers is required when --tls-security-profile=Custom")
		}
		profileMinVersion = minVersion
		for _, c := range strings.Split(ciphers, ",") {
			if s := strings.TrimSpace(c); s != "" {
				profileCiphers = append(profileCiphers, s)
			}
		}
		if len(profileCiphers) == 0 {
			return nil, fmt.Errorf("--tls-ciphers is required when --tls-security-profile=Custom")
		}
	default:
		return nil, fmt.Errorf(
			"unknown TLS security profile %q, valid values are: Old, Intermediate, Modern, Custom",
			profileType,
		)
	}

	goMinVersion, err := ocpcrypto.TLSVersion(profileMinVersion)
	if err != nil {
		return nil, fmt.Errorf("converting TLS min version %q: %w", profileMinVersion, err)
	}

	cipherSuiteIDs, err := openSSLCiphersToIDs(profileCiphers)
	if err != nil {
		return nil, err
	}

	return func(cfg *tls.Config) {
		cfg.MinVersion = goMinVersion
		cfg.CipherSuites = cipherSuiteIDs
	}, nil
}

var tls13Ciphers = map[string]bool{
	"TLS_AES_128_GCM_SHA256":       true,
	"TLS_AES_256_GCM_SHA384":       true,
	"TLS_CHACHA20_POLY1305_SHA256": true,
}

func openSSLCiphersToIDs(opensslCiphers []string) ([]uint16, error) {
	ianaNames := ocpcrypto.OpenSSLToIANACipherSuites(opensslCiphers)

	ianaToID := make(map[string]uint16)
	for _, suite := range tls.CipherSuites() {
		ianaToID[suite.Name] = suite.ID
	}
	for _, suite := range tls.InsecureCipherSuites() {
		ianaToID[suite.Name] = suite.ID
	}

	var ids []uint16
	hasTLS13Only := true
	for _, cipher := range opensslCiphers {
		if !tls13Ciphers[cipher] {
			hasTLS13Only = false
			break
		}
	}

	for _, name := range ianaNames {
		if id, ok := ianaToID[name]; ok {
			ids = append(ids, id)
		}
	}

	if len(ids) == 0 && !hasTLS13Only {
		return nil, fmt.Errorf("no valid ciphers resolved from the provided list")
	}

	return ids, nil
}
