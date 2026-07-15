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
	prometheusRuleName = "virt-observability-rules"
)

type PrometheusRuleReconciler struct {
	client.Client
	Scheme                  *runtime.Scheme
	Namespace               string
	AlertsAllowlist         map[string]bool
	RecordingRulesAllowlist map[string]bool
}

func (r *PrometheusRuleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if err := rules.SetupRules(r.Namespace, r.AlertsAllowlist, r.RecordingRulesAllowlist); err != nil {
		return ctrl.Result{}, fmt.Errorf("setting up rules: %w", err)
	}

	existing := &monitoringv1.PrometheusRule{}
	getErr := r.Get(ctx, types.NamespacedName{
		Name:      prometheusRuleName,
		Namespace: r.Namespace,
	}, existing)

	if !rules.HasRegisteredRules() {
		if getErr == nil {
			logger.Info("No rules registered, deleting PrometheusRule", "name", prometheusRuleName, "namespace", r.Namespace)
			return ctrl.Result{}, r.Delete(ctx, existing)
		}
		return ctrl.Result{}, nil
	}

	desired, err := r.buildDesiredPrometheusRule()
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("building PrometheusRule: %w", err)
	}

	if errors.IsNotFound(getErr) {
		logger.Info("Creating PrometheusRule", "name", prometheusRuleName, "namespace", r.Namespace)
		return ctrl.Result{}, r.Create(ctx, desired)
	}
	if getErr != nil {
		return ctrl.Result{}, getErr
	}

	if !equality.Semantic.DeepEqual(existing.Spec, desired.Spec) ||
		!equality.Semantic.DeepEqual(existing.Labels, desired.Labels) ||
		!equality.Semantic.DeepEqual(existing.Annotations, desired.Annotations) {

		existing.Spec = desired.Spec
		existing.Labels = desired.Labels
		existing.Annotations = desired.Annotations

		logger.Info("Updating PrometheusRule", "name", prometheusRuleName, "namespace", r.Namespace)
		return ctrl.Result{}, r.Update(ctx, existing)
	}

	return ctrl.Result{}, nil
}

func (r *PrometheusRuleReconciler) buildDesiredPrometheusRule() (*monitoringv1.PrometheusRule, error) {
	pr, err := rules.BuildPrometheusRule(prometheusRuleName, r.Namespace, commonLabels(labelValueComponentRules))
	if err != nil {
		return nil, err
	}

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
