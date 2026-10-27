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

package vmstats

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sync"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// CAConfigMapName is the ConfigMap virt-operator keeps the KubeVirt CA bundle in.
	CAConfigMapName = "kubevirt-ca"
	caBundleKey     = "ca-bundle"

	virtHandlerServerCommonName = "kubevirt.io:system:node:virt-handler"

	caRefreshInterval = time.Minute
	caFetchTimeout    = 10 * time.Second
)

var tlsLog = ctrl.Log.WithName("vmstats-tls")

// ClientCertificateFunc returns the client certificate to present to virt-handler.
type ClientCertificateFunc func() (*tls.Certificate, error)

// CAManager serves the KubeVirt CA bundle, re-reading it periodically so that
// CA rotations are picked up without a restart.
type CAManager struct {
	reader    client.Reader
	namespace string

	mu        sync.Mutex
	pool      *x509.CertPool
	fetchedAt time.Time
}

func NewCAManager(reader client.Reader, namespace string) *CAManager {
	return &CAManager{reader: reader, namespace: namespace}
}

func (m *CAManager) CertPool(ctx context.Context) (*x509.CertPool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.pool != nil && time.Since(m.fetchedAt) < caRefreshInterval {
		return m.pool, nil
	}

	pool, err := m.fetch(ctx)
	if err != nil {
		if m.pool != nil {
			tlsLog.V(2).Info("keeping previously loaded KubeVirt CA bundle", "error", err)
			return m.pool, nil
		}
		return nil, err
	}

	m.pool = pool
	m.fetchedAt = time.Now()

	return m.pool, nil
}

func (m *CAManager) fetch(ctx context.Context) (*x509.CertPool, error) {
	ctx, cancel := context.WithTimeout(ctx, caFetchTimeout)
	defer cancel()

	configMap := &k8sv1.ConfigMap{}
	key := types.NamespacedName{Namespace: m.namespace, Name: CAConfigMapName}
	if err := m.reader.Get(ctx, key, configMap); err != nil {
		return nil, fmt.Errorf("reading ConfigMap %s: %w", key, err)
	}

	bundle := configMap.Data[caBundleKey]
	if bundle == "" {
		return nil, fmt.Errorf("ConfigMap %s has no %q entry", key, caBundleKey)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM([]byte(bundle)) {
		return nil, fmt.Errorf("ConfigMap %s holds no parseable certificate", key)
	}

	return pool, nil
}

// NewVMStatsTLSConfig builds the client TLS configuration for the virt-handler
// vmstats endpoint.
//
// virt-handler is dialed by pod IP and its server certificate carries DNS SANs
// only, so the stdlib hostname check cannot be used. The certificate chain is
// verified against the KubeVirt CA and the virt-handler common name is pinned
// instead, which is how KubeVirt's own clients reach virt-handler.
func NewVMStatsTLSConfig(caManager *CAManager, clientCert ClientCertificateFunc) *tls.Config {
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		GetClientCertificate: func(_ *tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return clientCert()
		},
		// #nosec G402 -- verification happens in VerifyPeerCertificate below.
		InsecureSkipVerify: true,
		VerifyPeerCertificate: func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
			pool, err := caManager.CertPool(context.Background())
			if err != nil {
				return fmt.Errorf("loading KubeVirt CA bundle: %w", err)
			}
			return verifyVirtHandlerCert(rawCerts, pool)
		},
	}
}

func verifyVirtHandlerCert(rawCerts [][]byte, pool *x509.CertPool) error {
	if len(rawCerts) == 0 {
		return fmt.Errorf("virt-handler presented no server certificate")
	}

	peer, err := x509.ParseCertificate(rawCerts[0])
	if err != nil {
		return fmt.Errorf("parsing virt-handler server certificate: %w", err)
	}

	intermediates := x509.NewCertPool()
	for _, raw := range rawCerts[1:] {
		if c, err := x509.ParseCertificate(raw); err == nil {
			intermediates.AddCert(c)
		}
	}

	if _, err := peer.Verify(x509.VerifyOptions{
		Roots:         pool,
		Intermediates: intermediates,
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}); err != nil {
		return fmt.Errorf("verifying virt-handler server certificate: %w", err)
	}

	if peer.Subject.CommonName != virtHandlerServerCommonName {
		return fmt.Errorf(
			"virt-handler server certificate has common name %q, expected %q",
			peer.Subject.CommonName, virtHandlerServerCommonName,
		)
	}

	return nil
}
