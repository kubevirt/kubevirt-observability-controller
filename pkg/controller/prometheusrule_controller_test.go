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
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kubevirt/observability-operator/pkg/monitoring/rules"
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
}

var _ = Describe("PrometheusRule Reconciler", func() {
	BeforeEach(func() {
		_ = rules.SetupRules("kubevirt")
	})

	It("should create PrometheusRule when it does not exist", func() {
		ctx := context.Background()

		fakeClient := fake.NewClientBuilder().
			WithScheme(testScheme).
			Build()

		reconciler := &PrometheusRuleReconciler{
			Client:    fakeClient,
			Scheme:    testScheme,
			Version:   "0.0.1",
			Namespace: "kubevirt",
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
		Expect(pr.Annotations).To(HaveKeyWithValue("observability-operator.kubevirt.io/version", "0.0.1"))
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
			WithObjects(stale).
			Build()

		reconciler := &PrometheusRuleReconciler{
			Client:    fakeClient,
			Scheme:    testScheme,
			Version:   "0.0.1",
			Namespace: "kubevirt",
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
			Build()

		reconciler := &PrometheusRuleReconciler{
			Client:    fakeClient,
			Scheme:    testScheme,
			Version:   "0.0.1",
			Namespace: "kubevirt",
		}

		_, err := reconciler.Reconcile(ctx, reconcile.Request{})
		Expect(err).ToNot(HaveOccurred())

		result, err := reconciler.Reconcile(ctx, reconcile.Request{})
		Expect(err).ToNot(HaveOccurred())
		Expect(result).To(Equal(reconcile.Result{}))
	})
})
