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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	k6tv1 "kubevirt.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/rules"
)

func TestController(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var testScheme *runtime.Scheme

func init() {
	testScheme = runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(testScheme)
	_ = monitoringv1.AddToScheme(testScheme)
	_ = k6tv1.AddToScheme(testScheme)
}

func newKubeVirt() *k6tv1.KubeVirt {
	return &k6tv1.KubeVirt{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubevirt",
			Namespace: "kubevirt",
		},
	}
}

var _ = Describe("PrometheusRule Reconciler", func() {
	BeforeEach(func() {
		rules.ResetRegistry()
	})

	It("should create PrometheusRule when it does not exist", func() {
		ctx := context.Background()

		fakeClient := fake.NewClientBuilder().
			WithScheme(testScheme).
			WithObjects(newKubeVirt()).
			Build()

		reconciler := &PrometheusRuleReconciler{
			Client:  fakeClient,
			Scheme:  testScheme,
			Version: "0.0.1",
		}

		result, err := reconciler.Reconcile(ctx, reconcile.Request{})
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))

		pr := &monitoringv1.PrometheusRule{}
		err = fakeClient.Get(ctx, types.NamespacedName{
			Name:      "kubevirt-observability-rules",
			Namespace: "kubevirt",
		}, pr)
		Expect(err).ToNot(HaveOccurred())
		Expect(pr.Spec.Groups).ToNot(BeEmpty())
		Expect(pr.Annotations).To(HaveKeyWithValue("kubevirt-observability-controller.kubevirt.io/version", "0.0.1"))
	})

	It("should update PrometheusRule when it already exists but differs", func() {
		ctx := context.Background()

		stale := &monitoringv1.PrometheusRule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt-observability-rules",
				Namespace: "kubevirt",
			},
			Spec: monitoringv1.PrometheusRuleSpec{},
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(testScheme).
			WithObjects(newKubeVirt(), stale).
			Build()

		reconciler := &PrometheusRuleReconciler{
			Client:  fakeClient,
			Scheme:  testScheme,
			Version: "0.0.1",
		}

		_, err := reconciler.Reconcile(ctx, reconcile.Request{})
		Expect(err).ToNot(HaveOccurred())

		pr := &monitoringv1.PrometheusRule{}
		err = fakeClient.Get(ctx, types.NamespacedName{
			Name:      "kubevirt-observability-rules",
			Namespace: "kubevirt",
		}, pr)
		Expect(err).ToNot(HaveOccurred())
		Expect(pr.Spec.Groups).ToNot(BeEmpty())
	})

	It("should not update PrometheusRule when it matches desired state", func() {
		ctx := context.Background()

		fakeClient := fake.NewClientBuilder().
			WithScheme(testScheme).
			WithObjects(newKubeVirt()).
			Build()

		reconciler := &PrometheusRuleReconciler{
			Client:  fakeClient,
			Scheme:  testScheme,
			Version: "0.0.1",
		}

		_, err := reconciler.Reconcile(ctx, reconcile.Request{})
		Expect(err).ToNot(HaveOccurred())

		result, err := reconciler.Reconcile(ctx, reconcile.Request{})
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
	})

	It("should skip reconciliation when no KubeVirt CR exists", func() {
		ctx := context.Background()

		fakeClient := fake.NewClientBuilder().
			WithScheme(testScheme).
			Build()

		reconciler := &PrometheusRuleReconciler{
			Client:  fakeClient,
			Scheme:  testScheme,
			Version: "0.0.1",
		}

		result, err := reconciler.Reconcile(ctx, reconcile.Request{})
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
	})

	It("should create PrometheusRule with only allowlisted alerts", func() {
		ctx := context.Background()

		fakeClient := fake.NewClientBuilder().
			WithScheme(testScheme).
			WithObjects(newKubeVirt()).
			Build()

		reconciler := &PrometheusRuleReconciler{
			Client:  fakeClient,
			Scheme:  testScheme,
			Version: "0.0.1",
			AlertsAllowlist: map[string]bool{
				"VirtAPIDown": true,
			},
			RecordingRulesAllowlist: map[string]bool{
				"cluster:kubevirt_virt_api_up:sum": true,
			},
		}

		_, err := reconciler.Reconcile(ctx, reconcile.Request{})
		Expect(err).ToNot(HaveOccurred())

		pr := &monitoringv1.PrometheusRule{}
		err = fakeClient.Get(ctx, types.NamespacedName{
			Name:      "kubevirt-observability-rules",
			Namespace: "kubevirt",
		}, pr)
		Expect(err).ToNot(HaveOccurred())

		var alertCount, recordingRuleCount int
		for _, g := range pr.Spec.Groups {
			for _, r := range g.Rules {
				if r.Alert != "" {
					alertCount++
				} else {
					recordingRuleCount++
				}
			}
		}

		Expect(alertCount).To(Equal(1))
		Expect(recordingRuleCount).To(Equal(1))
	})

	It("should not create PrometheusRule when both allowlists are empty", func() {
		ctx := context.Background()

		fakeClient := fake.NewClientBuilder().
			WithScheme(testScheme).
			WithObjects(newKubeVirt()).
			Build()

		reconciler := &PrometheusRuleReconciler{
			Client:                  fakeClient,
			Scheme:                  testScheme,
			Version:                 "0.0.1",
			AlertsAllowlist:         map[string]bool{},
			RecordingRulesAllowlist: map[string]bool{},
		}

		_, err := reconciler.Reconcile(ctx, reconcile.Request{})
		Expect(err).ToNot(HaveOccurred())

		pr := &monitoringv1.PrometheusRule{}
		err = fakeClient.Get(ctx, types.NamespacedName{
			Name:      "kubevirt-observability-rules",
			Namespace: "kubevirt",
		}, pr)
		Expect(errors.IsNotFound(err)).To(BeTrue())
	})

	It("should delete existing PrometheusRule when both allowlists become empty", func() {
		ctx := context.Background()

		existing := &monitoringv1.PrometheusRule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubevirt-observability-rules",
				Namespace: "kubevirt",
			},
			Spec: monitoringv1.PrometheusRuleSpec{},
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(testScheme).
			WithObjects(newKubeVirt(), existing).
			Build()

		reconciler := &PrometheusRuleReconciler{
			Client:                  fakeClient,
			Scheme:                  testScheme,
			Version:                 "0.0.1",
			AlertsAllowlist:         map[string]bool{},
			RecordingRulesAllowlist: map[string]bool{},
		}

		_, err := reconciler.Reconcile(ctx, reconcile.Request{})
		Expect(err).ToNot(HaveOccurred())

		pr := &monitoringv1.PrometheusRule{}
		err = fakeClient.Get(ctx, types.NamespacedName{
			Name:      "kubevirt-observability-rules",
			Namespace: "kubevirt",
		}, pr)
		Expect(errors.IsNotFound(err)).To(BeTrue())
	})
})
