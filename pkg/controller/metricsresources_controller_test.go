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

package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("MetricsResources Reconciler", func() {
	const (
		namespace          = "test-namespace"
		serviceAccountName = "test-sa"
	)

	newReconciler := func(port int32, secure bool, objs ...client.Object) *MetricsResourcesReconciler {
		fakeClient := fake.NewClientBuilder().
			WithScheme(testScheme).
			WithObjects(objs...).
			Build()
		return &MetricsResourcesReconciler{
			Client:             fakeClient,
			Scheme:             testScheme,
			Namespace:          namespace,
			ServiceAccountName: serviceAccountName,
			MetricsPort:        port,
			SecureMetrics:      secure,
		}
	}

	It("should create Service, Secret, and ServiceMonitor when none exist", func() {
		ctx := context.Background()
		r := newReconciler(8443, true)

		result, err := r.Reconcile(ctx, reconcile.Request{})
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))

		// Verify Service
		svc := &corev1.Service{}
		err = r.Get(ctx, types.NamespacedName{Name: metricsServiceName, Namespace: namespace}, svc)
		Expect(err).ToNot(HaveOccurred())
		Expect(svc.Spec.Ports).To(HaveLen(1))
		Expect(svc.Spec.Ports[0].Port).To(Equal(int32(8443)))
		Expect(svc.Spec.Ports[0].Name).To(Equal(metricsPortName))
		Expect(svc.Spec.Selector).To(HaveKeyWithValue(labelKeyName, controllerName))

		// Verify Secret
		secret := &corev1.Secret{}
		err = r.Get(ctx, types.NamespacedName{Name: metricsTokenSecretName, Namespace: namespace}, secret)
		Expect(err).ToNot(HaveOccurred())
		Expect(secret.Type).To(Equal(corev1.SecretTypeServiceAccountToken))
		Expect(secret.Annotations).To(HaveKeyWithValue(corev1.ServiceAccountNameKey, serviceAccountName))

		// Verify ServiceMonitor
		sm := &monitoringv1.ServiceMonitor{}
		err = r.Get(ctx, types.NamespacedName{Name: metricsServiceMonitorName, Namespace: namespace}, sm)
		Expect(err).ToNot(HaveOccurred())
		Expect(sm.Spec.Endpoints).To(HaveLen(1))
		Expect(sm.Spec.Endpoints[0].Scheme).ToNot(BeNil())
		Expect(*sm.Spec.Endpoints[0].Scheme).To(Equal(monitoringv1.Scheme("https")))
		Expect(sm.Spec.Endpoints[0].TLSConfig).ToNot(BeNil())
		Expect(sm.Spec.Endpoints[0].TLSConfig.InsecureSkipVerify).ToNot(BeNil())
		Expect(*sm.Spec.Endpoints[0].TLSConfig.InsecureSkipVerify).To(BeTrue())
		Expect(sm.Spec.Endpoints[0].Authorization).ToNot(BeNil())
		Expect(sm.Spec.Endpoints[0].Authorization.Credentials).ToNot(BeNil())
		Expect(sm.Spec.Endpoints[0].Authorization.Credentials.Name).To(Equal(metricsTokenSecretName))
		Expect(sm.Spec.Endpoints[0].Authorization.Credentials.Key).To(Equal(metricsTokenKey))
	})

	It("should create ServiceMonitor with HTTP scheme when secureMetrics is false", func() {
		ctx := context.Background()
		r := newReconciler(8080, false)

		_, err := r.Reconcile(ctx, reconcile.Request{})
		Expect(err).ToNot(HaveOccurred())

		sm := &monitoringv1.ServiceMonitor{}
		err = r.Get(ctx, types.NamespacedName{Name: metricsServiceMonitorName, Namespace: namespace}, sm)
		Expect(err).ToNot(HaveOccurred())
		Expect(sm.Spec.Endpoints[0].Scheme).ToNot(BeNil())
		Expect(*sm.Spec.Endpoints[0].Scheme).To(Equal(monitoringv1.Scheme("http")))
		Expect(sm.Spec.Endpoints[0].TLSConfig).To(BeNil())
	})

	It("should update Service when port differs", func() {
		ctx := context.Background()

		existingService := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      metricsServiceName,
				Namespace: namespace,
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{Name: metricsPortName, Port: 8080},
				},
			},
		}

		r := newReconciler(8443, true, existingService)

		_, err := r.Reconcile(ctx, reconcile.Request{})
		Expect(err).ToNot(HaveOccurred())

		svc := &corev1.Service{}
		err = r.Get(ctx, types.NamespacedName{Name: metricsServiceName, Namespace: namespace}, svc)
		Expect(err).ToNot(HaveOccurred())
		Expect(svc.Spec.Ports[0].Port).To(Equal(int32(8443)))
	})

	It("should not update resources when they match desired state", func() {
		ctx := context.Background()
		r := newReconciler(8443, true)

		_, err := r.Reconcile(ctx, reconcile.Request{})
		Expect(err).ToNot(HaveOccurred())

		// Reconcile again — should be a no-op
		result, err := r.Reconcile(ctx, reconcile.Request{})
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
	})

	It("should recreate deleted resources", func() {
		ctx := context.Background()
		r := newReconciler(8443, true)

		// First reconcile creates everything
		_, err := r.Reconcile(ctx, reconcile.Request{})
		Expect(err).ToNot(HaveOccurred())

		// Delete the Service
		svc := &corev1.Service{}
		err = r.Get(ctx, types.NamespacedName{Name: metricsServiceName, Namespace: namespace}, svc)
		Expect(err).ToNot(HaveOccurred())
		err = r.Delete(ctx, svc)
		Expect(err).ToNot(HaveOccurred())

		// Reconcile again — should recreate
		_, err = r.Reconcile(ctx, reconcile.Request{})
		Expect(err).ToNot(HaveOccurred())

		err = r.Get(ctx, types.NamespacedName{Name: metricsServiceName, Namespace: namespace}, svc)
		Expect(err).ToNot(HaveOccurred())
	})
})
