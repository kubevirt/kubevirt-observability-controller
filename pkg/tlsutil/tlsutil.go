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

	"github.com/go-logr/logr"
	configv1 "github.com/openshift/api/config/v1"
	ocpcrypto "github.com/openshift/library-go/pkg/crypto"
)

func TLSSecurityProfileToTLSConfig(
	profileType, minVersion, ciphers, groups string, logger logr.Logger,
) (func(*tls.Config), error) {
	var profileMinVersion string
	var profileCiphers []string
	var profileGroups []configv1.TLSGroup

	tlsProfileType := configv1.TLSProfileType(profileType)

	switch tlsProfileType {
	case configv1.TLSProfileOldType, configv1.TLSProfileIntermediateType, configv1.TLSProfileModernType:
		if minVersion != "" || ciphers != "" || groups != "" {
			return nil, fmt.Errorf(
				"--tls-min-version, --tls-ciphers and --tls-groups are only valid with --tls-security-profile=Custom",
			)
		}
		spec := configv1.TLSProfiles[tlsProfileType]
		profileMinVersion = string(spec.MinTLSVersion)
		profileCiphers = spec.Ciphers
		profileGroups = spec.Groups

	case configv1.TLSProfileCustomType:
		if minVersion == "" {
			return nil, fmt.Errorf("--tls-min-version is required when --tls-security-profile=Custom")
		}

		if ciphers == "" {
			if minVersion != string(configv1.VersionTLS13) {
				return nil, fmt.Errorf("--tls-ciphers is required when --tls-security-profile=Custom")
			}
		} else {
			for c := range strings.SplitSeq(ciphers, ",") {
				if s := strings.TrimSpace(c); s != "" {
					profileCiphers = append(profileCiphers, s)
				}
			}
		}

		profileMinVersion = minVersion
		if len(profileCiphers) == 0 && profileMinVersion != string(configv1.VersionTLS13) {
			return nil, fmt.Errorf("--tls-ciphers is required when --tls-security-profile=Custom")
		}

		for c := range strings.SplitSeq(groups, ",") {
			if s := strings.TrimSpace(c); s != "" {
				profileGroups = append(profileGroups, configv1.TLSGroup(s))
			}
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

	curveIDs, unsupported := ocpcrypto.TLSGroupsToCurveIDs(profileGroups)
	if len(unsupported) > 0 {
		logger.WithName("tls-security-profile-logger").Info("unsupported TLS groups ignored", "groups", unsupported)
	}

	if len(curveIDs) == 0 {
		return nil, fmt.Errorf("no valid groups resolved from the provided list")
	}

	return func(cfg *tls.Config) {
		cfg.MinVersion = goMinVersion
		cfg.CipherSuites = cipherSuiteIDs
		cfg.CurvePreferences = curveIDs
	}, nil
}

var tls13Ciphers = map[string]struct{}{
	"TLS_AES_128_GCM_SHA256":       {},
	"TLS_AES_256_GCM_SHA384":       {},
	"TLS_CHACHA20_POLY1305_SHA256": {},
}

var ianaToID map[string]uint16

func init() {
	ianaToID = make(map[string]uint16)

	for _, suite := range tls.CipherSuites() {
		ianaToID[suite.Name] = suite.ID
	}
	for _, suite := range tls.InsecureCipherSuites() {
		ianaToID[suite.Name] = suite.ID
	}
}

func ianaCiphersToTLSIDs(ianaNames []string) []uint16 {
	var ids []uint16
	for _, name := range ianaNames {
		if id, ok := ianaToID[name]; ok {
			ids = append(ids, id)
		}
	}

	return ids
}

func openSSLCiphersToIDs(opensslCiphers []string) ([]uint16, error) {
	ianaNames := ocpcrypto.OpenSSLToIANACipherSuites(opensslCiphers)
	hasTLS13Only := isTLS13Only(opensslCiphers)
	ids := ianaCiphersToTLSIDs(ianaNames)

	if len(ids) == 0 && !hasTLS13Only {
		return nil, fmt.Errorf("no valid ciphers resolved from the provided list")
	}

	return ids, nil
}

func isTLS13Only(opensslCiphers []string) bool {
	for _, cipher := range opensslCiphers {
		if _, isV13cipher := tls13Ciphers[cipher]; !isV13cipher {
			return false
		}
	}

	return true
}
