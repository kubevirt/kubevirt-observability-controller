package controller

import (
	"context"
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	k6tv1 "kubevirt.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/rules"
)

const (
	prometheusRuleName = "kubevirt-observability-rules"
)

type PrometheusRuleReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Version string
}

func (r *PrometheusRuleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	kvList := &k6tv1.KubeVirtList{}
	if err := r.List(ctx, kvList); err != nil {
		return ctrl.Result{}, fmt.Errorf("listing KubeVirt CRs: %w", err)
	}

	if len(kvList.Items) == 0 {
		logger.Info("No KubeVirt CR found, skipping reconciliation")
		return ctrl.Result{}, nil
	}

	namespace := kvList.Items[0].Namespace

	if err := rules.SetupRules(namespace); err != nil {
		return ctrl.Result{}, fmt.Errorf("setting up rules: %w", err)
	}

	desired, err := r.buildDesiredPrometheusRule(namespace)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("building PrometheusRule: %w", err)
	}

	existing := &monitoringv1.PrometheusRule{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      prometheusRuleName,
		Namespace: namespace,
	}, existing)

	if errors.IsNotFound(err) {
		logger.Info("Creating PrometheusRule", "name", prometheusRuleName, "namespace", namespace)
		return ctrl.Result{}, r.Create(ctx, desired)
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	if !equality.Semantic.DeepEqual(existing.Spec, desired.Spec) ||
		existing.Annotations["kubevirt-observability-controller.kubevirt.io/version"] != r.Version {
		existing.Spec = desired.Spec
		existing.Labels = desired.Labels
		existing.Annotations = desired.Annotations
		logger.Info("Updating PrometheusRule", "name", prometheusRuleName, "namespace", namespace)
		return ctrl.Result{}, r.Update(ctx, existing)
	}

	return ctrl.Result{}, nil
}

func (r *PrometheusRuleReconciler) buildDesiredPrometheusRule(namespace string) (*monitoringv1.PrometheusRule, error) {
	pr, err := rules.BuildPrometheusRule(prometheusRuleName, namespace)
	if err != nil {
		return nil, err
	}

	if pr.Annotations == nil {
		pr.Annotations = make(map[string]string)
	}
	pr.Annotations["kubevirt-observability-controller.kubevirt.io/version"] = r.Version

	return pr, nil
}

func (r *PrometheusRuleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&monitoringv1.PrometheusRule{}).
		Watches(&k6tv1.KubeVirt{}, handler.EnqueueRequestsFromMapFunc(
			func(ctx context.Context, obj client.Object) []reconcile.Request {
				return []reconcile.Request{{}}
			},
		)).
		Complete(r)
}
