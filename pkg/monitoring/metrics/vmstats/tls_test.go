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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const testNamespace = "kubevirt"

type testCA struct {
	cert   *x509.Certificate
	key    *ecdsa.PrivateKey
	bundle []byte
}

func newTestCA(commonName string) *testCA {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	Expect(err).ToNot(HaveOccurred())

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: commonName},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	Expect(err).ToNot(HaveOccurred())

	cert, err := x509.ParseCertificate(der)
	Expect(err).ToNot(HaveOccurred())

	return &testCA{
		cert:   cert,
		key:    key,
		bundle: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
	}
}

// serverCert issues a leaf certificate without any IP SAN, mirroring the
// virt-handler server certificate that is dialed by pod IP.
func (ca *testCA) serverCert(commonName string) tls.Certificate {
	return ca.leafCert(commonName, x509.ExtKeyUsageServerAuth)
}

func (ca *testCA) clientCert(commonName string) tls.Certificate {
	return ca.leafCert(commonName, x509.ExtKeyUsageClientAuth)
}

func (ca *testCA) leafCert(commonName string, usage x509.ExtKeyUsage) tls.Certificate {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	Expect(err).ToNot(HaveOccurred())

	template := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: commonName},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{usage},
		DNSNames:     []string{"virt-handler", "virt-handler.kubevirt.svc"},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, ca.cert, &key.PublicKey, ca.key)
	Expect(err).ToNot(HaveOccurred())

	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

func caConfigMap(bundle string) *k8sv1.ConfigMap {
	return &k8sv1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: CAConfigMapName, Namespace: testNamespace},
		Data:       map[string]string{caBundleKey: bundle},
	}
}

var _ = Describe("CAManager", func() {
	It("should build a cert pool from the kubevirt CA bundle", func() {
		ca := newTestCA("kubevirt.io")
		c := fake.NewClientBuilder().WithObjects(caConfigMap(string(ca.bundle))).Build()

		pool, err := NewCAManager(c, testNamespace).CertPool(context.Background())
		Expect(err).ToNot(HaveOccurred())

		leaf, err := x509.ParseCertificate(ca.serverCert("leaf").Certificate[0])
		Expect(err).ToNot(HaveOccurred())
		_, err = leaf.Verify(x509.VerifyOptions{Roots: pool})
		Expect(err).ToNot(HaveOccurred())
	})

	It("should fail when the CA ConfigMap is missing", func() {
		c := fake.NewClientBuilder().Build()

		_, err := NewCAManager(c, testNamespace).CertPool(context.Background())
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(CAConfigMapName))
	})

	It("should fail when the CA bundle holds no certificate", func() {
		c := fake.NewClientBuilder().WithObjects(caConfigMap("not a cert")).Build()

		_, err := NewCAManager(c, testNamespace).CertPool(context.Background())
		Expect(err).To(HaveOccurred())
	})

	It("should not re-read the ConfigMap within the TTL", func() {
		ca := newTestCA("kubevirt.io")
		reads := 0
		c := countingReader(caConfigMap(string(ca.bundle)), &reads)

		manager := NewCAManager(c, testNamespace)
		for range 3 {
			_, err := manager.CertPool(context.Background())
			Expect(err).ToNot(HaveOccurred())
		}
		Expect(reads).To(Equal(1))
	})

	It("should re-read the ConfigMap once the TTL expired", func() {
		ca := newTestCA("kubevirt.io")
		reads := 0
		c := countingReader(caConfigMap(string(ca.bundle)), &reads)

		manager := NewCAManager(c, testNamespace)
		_, err := manager.CertPool(context.Background())
		Expect(err).ToNot(HaveOccurred())

		manager.fetchedAt = time.Now().Add(-2 * caRefreshInterval)
		_, err = manager.CertPool(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(reads).To(Equal(2))
	})

	It("should keep serving the last known CA when a refresh fails", func() {
		ca := newTestCA("kubevirt.io")
		failing := false
		c := fake.NewClientBuilder().
			WithObjects(caConfigMap(string(ca.bundle))).
			WithInterceptorFuncs(interceptor.Funcs{
				Get: func(
					ctx context.Context, cl client.WithWatch, key client.ObjectKey,
					obj client.Object, opts ...client.GetOption,
				) error {
					if failing {
						return fmt.Errorf("API unavailable")
					}
					return cl.Get(ctx, key, obj, opts...)
				},
			}).Build()

		manager := NewCAManager(c, testNamespace)
		_, err := manager.CertPool(context.Background())
		Expect(err).ToNot(HaveOccurred())

		failing = true
		manager.fetchedAt = time.Now().Add(-2 * caRefreshInterval)
		pool, err := manager.CertPool(context.Background())
		Expect(err).ToNot(HaveOccurred())
		Expect(pool).ToNot(BeNil())
	})
})

var _ = Describe("vmstats TLS config", func() {
	var ca *testCA
	var caManager *CAManager

	BeforeEach(func() {
		ca = newTestCA("kubevirt.io")
		caManager = NewCAManager(
			fake.NewClientBuilder().WithObjects(caConfigMap(string(ca.bundle))).Build(),
			testNamespace,
		)
	})

	serve := func(serverCert tls.Certificate, clientTLS *tls.Config) error {
		server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		server.TLS = &tls.Config{Certificates: []tls.Certificate{serverCert}, MinVersion: tls.VersionTLS12}
		server.StartTLS()
		defer server.Close()

		httpClient := &http.Client{Transport: &http.Transport{TLSClientConfig: clientTLS}}
		resp, err := httpClient.Get(server.URL)
		if err == nil {
			_ = resp.Body.Close()
		}
		return err
	}

	It("should accept a virt-handler certificate signed by the kubevirt CA", func() {
		err := serve(
			ca.serverCert(virtHandlerServerCommonName),
			NewVMStatsTLSConfig(caManager, staticCert(tls.Certificate{})),
		)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should reject a certificate signed by an unknown CA", func() {
		other := newTestCA("rogue.io")

		err := serve(
			other.serverCert(virtHandlerServerCommonName),
			NewVMStatsTLSConfig(caManager, staticCert(tls.Certificate{})),
		)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("verifying virt-handler server certificate"))
	})

	It("should reject a kubevirt certificate with another common name", func() {
		err := serve(
			ca.serverCert("kubevirt.io:system:node:virt-launcher"),
			NewVMStatsTLSConfig(caManager, staticCert(tls.Certificate{})),
		)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("common name"))
	})

	It("should fail the handshake when the CA bundle cannot be read", func() {
		err := serve(
			ca.serverCert(virtHandlerServerCommonName),
			NewVMStatsTLSConfig(NewCAManager(fake.NewClientBuilder().Build(), testNamespace), staticCert(tls.Certificate{})),
		)
		Expect(err).To(HaveOccurred())
	})

	It("should present the client certificate to the server", func() {
		clientCert := ca.clientCert("kubevirt.io:system:client:monitoring")

		var peerCommonName string
		server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			peerCommonName = r.TLS.PeerCertificates[0].Subject.CommonName
			w.WriteHeader(http.StatusOK)
		}))
		pool := x509.NewCertPool()
		pool.AddCert(ca.cert)
		server.TLS = &tls.Config{
			Certificates: []tls.Certificate{ca.serverCert(virtHandlerServerCommonName)},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    pool,
			MinVersion:   tls.VersionTLS12,
		}
		server.StartTLS()
		defer server.Close()

		httpClient := &http.Client{
			Transport: &http.Transport{TLSClientConfig: NewVMStatsTLSConfig(caManager, staticCert(clientCert))},
		}
		resp, err := httpClient.Get(server.URL)
		Expect(err).ToNot(HaveOccurred())
		_ = resp.Body.Close()
		Expect(peerCommonName).To(Equal("kubevirt.io:system:client:monitoring"))
	})

	It("should present the rotated client certificate on the next handshake", func() {
		current := ca.clientCert("kubevirt.io:system:client:monitoring")

		var peerCommonNames []string
		server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			peerCommonNames = append(peerCommonNames, r.TLS.PeerCertificates[0].Subject.CommonName)
			w.WriteHeader(http.StatusOK)
		}))
		pool := x509.NewCertPool()
		pool.AddCert(ca.cert)
		server.TLS = &tls.Config{
			Certificates: []tls.Certificate{ca.serverCert(virtHandlerServerCommonName)},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ClientCAs:    pool,
			MinVersion:   tls.VersionTLS12,
		}
		server.StartTLS()
		defer server.Close()

		httpClient := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig:   NewVMStatsTLSConfig(caManager, func() (*tls.Certificate, error) { return &current, nil }),
				DisableKeepAlives: true,
			},
		}

		resp, err := httpClient.Get(server.URL)
		Expect(err).ToNot(HaveOccurred())
		_ = resp.Body.Close()

		current = ca.clientCert("kubevirt.io:system:client:monitoring-rotated")

		resp, err = httpClient.Get(server.URL)
		Expect(err).ToNot(HaveOccurred())
		_ = resp.Body.Close()

		Expect(peerCommonNames).To(Equal([]string{
			"kubevirt.io:system:client:monitoring",
			"kubevirt.io:system:client:monitoring-rotated",
		}))
	})
})

func staticCert(cert tls.Certificate) ClientCertificateFunc {
	return func() (*tls.Certificate, error) { return &cert, nil }
}

func countingReader(cm *k8sv1.ConfigMap, reads *int) client.Client {
	return fake.NewClientBuilder().WithObjects(cm).WithInterceptorFuncs(interceptor.Funcs{
		Get: func(
			ctx context.Context, cl client.WithWatch, key client.ObjectKey,
			obj client.Object, opts ...client.GetOption,
		) error {
			*reads++
			return cl.Get(ctx, key, obj, opts...)
		},
	}).Build()
}
