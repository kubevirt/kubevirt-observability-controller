package controller

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	k6tv1 "kubevirt.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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

func newKubeVirt(namespace string) *k6tv1.KubeVirt {
	return &k6tv1.KubeVirt{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubevirt",
			Namespace: namespace,
		},
	}
}

var _ = Describe("PrometheusRule Reconciler", func() {
	It("should create PrometheusRule when it does not exist", func() {
		ctx := context.Background()

		fakeClient := fake.NewClientBuilder().
			WithScheme(testScheme).
			WithObjects(newKubeVirt("kubevirt")).
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
			WithObjects(newKubeVirt("kubevirt"), stale).
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
			WithObjects(newKubeVirt("kubevirt")).
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
})
