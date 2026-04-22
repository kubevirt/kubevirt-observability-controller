package controller

import (
	"context"
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/kubevirt/observability-operator/pkg/monitoring/rules"
)

const (
	prometheusRuleName = "kubevirt-observability-rules"
)

type PrometheusRuleReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Version   string
	Namespace string
}

func (r *PrometheusRuleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	desired, err := r.buildDesiredPrometheusRule()
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("building PrometheusRule: %w", err)
	}

	existing := &monitoringv1.PrometheusRule{}
	err = r.Get(ctx, types.NamespacedName{
		Name:      prometheusRuleName,
		Namespace: r.Namespace,
	}, existing)

	if errors.IsNotFound(err) {
		logger.Info("Creating PrometheusRule", "name", prometheusRuleName)
		return ctrl.Result{}, r.Create(ctx, desired)
	}
	if err != nil {
		return ctrl.Result{}, err
	}

	if !equality.Semantic.DeepEqual(existing.Spec, desired.Spec) ||
		existing.Annotations["observability-operator.kubevirt.io/version"] != r.Version {
		existing.Spec = desired.Spec
		existing.Labels = desired.Labels
		existing.Annotations = desired.Annotations
		logger.Info("Updating PrometheusRule", "name", prometheusRuleName)
		return ctrl.Result{}, r.Update(ctx, existing)
	}

	return ctrl.Result{}, nil
}

func (r *PrometheusRuleReconciler) buildDesiredPrometheusRule() (*monitoringv1.PrometheusRule, error) {
	pr, err := rules.BuildPrometheusRule(prometheusRuleName, r.Namespace)
	if err != nil {
		return nil, err
	}

	if pr.Annotations == nil {
		pr.Annotations = make(map[string]string)
	}
	pr.Annotations["observability-operator.kubevirt.io/version"] = r.Version

	return pr, nil
}

func (r *PrometheusRuleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&monitoringv1.PrometheusRule{}).
		Complete(r)
}
