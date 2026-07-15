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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	metricsServiceName        = "virt-observability-controller-metrics"
	metricsTokenSecretName    = "virt-observability-controller-metrics-token"
	metricsServiceMonitorName = "virt-observability-controller"

	metricsPortName = "metrics"
	metricsTokenKey = "token"
)

type MetricsResourcesReconciler struct {
	client.Client
	Scheme             *runtime.Scheme
	Namespace          string
	ServiceAccountName string
	MetricsPort        int32
	SecureMetrics      bool
}

func (r *MetricsResourcesReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if err := r.reconcileService(ctx); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconciling Service: %w", err)
	}
	if err := r.reconcileSecret(ctx); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconciling Secret: %w", err)
	}
	if err := r.reconcileServiceMonitor(ctx); err != nil {
		return ctrl.Result{}, fmt.Errorf("reconciling ServiceMonitor: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *MetricsResourcesReconciler) reconcileService(ctx context.Context) error {
	logger := log.FromContext(ctx)
	desired := r.buildDesiredService()
	existing := &corev1.Service{}
	err := r.Get(ctx, types.NamespacedName{Name: metricsServiceName, Namespace: r.Namespace}, existing)

	if errors.IsNotFound(err) {
		logger.Info("Creating metrics Service", "name", metricsServiceName)
		return r.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	if !equality.Semantic.DeepEqual(existing.Spec.Ports, desired.Spec.Ports) ||
		!equality.Semantic.DeepEqual(existing.Spec.Selector, desired.Spec.Selector) ||
		!equality.Semantic.DeepEqual(existing.Labels, desired.Labels) {
		existing.Spec.Ports = desired.Spec.Ports
		existing.Spec.Selector = desired.Spec.Selector
		existing.Labels = desired.Labels
		logger.Info("Updating metrics Service", "name", metricsServiceName)
		return r.Update(ctx, existing)
	}

	return nil
}

func (r *MetricsResourcesReconciler) reconcileSecret(ctx context.Context) error {
	logger := log.FromContext(ctx)
	desired := r.buildDesiredSecret()
	existing := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: metricsTokenSecretName, Namespace: r.Namespace}, existing)

	if errors.IsNotFound(err) {
		logger.Info("Creating metrics token Secret", "name", metricsTokenSecretName)
		return r.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	if existing.Annotations[corev1.ServiceAccountNameKey] != r.ServiceAccountName ||
		!equality.Semantic.DeepEqual(existing.Labels, desired.Labels) {
		existing.Labels = desired.Labels
		existing.Annotations = desired.Annotations
		logger.Info("Updating metrics token Secret", "name", metricsTokenSecretName)
		return r.Update(ctx, existing)
	}

	return nil
}

func (r *MetricsResourcesReconciler) reconcileServiceMonitor(ctx context.Context) error {
	logger := log.FromContext(ctx)
	desired := r.buildDesiredServiceMonitor()
	existing := &monitoringv1.ServiceMonitor{}
	err := r.Get(ctx, types.NamespacedName{Name: metricsServiceMonitorName, Namespace: r.Namespace}, existing)

	if errors.IsNotFound(err) {
		logger.Info("Creating metrics ServiceMonitor", "name", metricsServiceMonitorName)
		return r.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	if !equality.Semantic.DeepEqual(existing.Spec, desired.Spec) ||
		!equality.Semantic.DeepEqual(existing.Labels, desired.Labels) {
		existing.Spec = desired.Spec
		existing.Labels = desired.Labels
		logger.Info("Updating metrics ServiceMonitor", "name", metricsServiceMonitorName)
		return r.Update(ctx, existing)
	}

	return nil
}

func (r *MetricsResourcesReconciler) buildDesiredService() *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      metricsServiceName,
			Namespace: r.Namespace,
			Labels:    commonLabels(labelValueComponentMetrics),
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       metricsPortName,
					Port:       r.MetricsPort,
					TargetPort: intstr.FromInt32(r.MetricsPort),
					Protocol:   corev1.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				labelKeyName: controllerName,
			},
		},
	}
}

func (r *MetricsResourcesReconciler) buildDesiredSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      metricsTokenSecretName,
			Namespace: r.Namespace,
			Labels:    commonLabels(labelValueComponentMetrics),
			Annotations: map[string]string{
				corev1.ServiceAccountNameKey: r.ServiceAccountName,
			},
		},
		Type: corev1.SecretTypeServiceAccountToken,
	}
}

func (r *MetricsResourcesReconciler) buildDesiredServiceMonitor() *monitoringv1.ServiceMonitor {
	endpoint := monitoringv1.Endpoint{
		Port: metricsPortName,
	}
	endpoint.Authorization = &monitoringv1.SafeAuthorization{
		Credentials: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: metricsTokenSecretName,
			},
			Key: metricsTokenKey,
		},
	}

	if r.SecureMetrics {
		scheme := monitoringv1.Scheme("https")
		endpoint.Scheme = &scheme
		endpoint.TLSConfig = &monitoringv1.TLSConfig{
			SafeTLSConfig: monitoringv1.SafeTLSConfig{
				InsecureSkipVerify: ptr.To(true),
			},
		}
	} else {
		scheme := monitoringv1.Scheme("http")
		endpoint.Scheme = &scheme
	}

	return &monitoringv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      metricsServiceMonitorName,
			Namespace: r.Namespace,
			Labels:    commonLabels(labelValueComponentMetrics),
		},
		Spec: monitoringv1.ServiceMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: commonLabels(labelValueComponentMetrics),
			},
			Endpoints: []monitoringv1.Endpoint{endpoint},
		},
	}
}

func (r *MetricsResourcesReconciler) SetupWithManager(mgr ctrl.Manager) error {
	mapIfOwned := func(names ...string) handler.EventHandler {
		nameSet := make(map[string]struct{}, len(names))
		for _, n := range names {
			nameSet[n] = struct{}{}
		}
		return handler.EnqueueRequestsFromMapFunc(
			func(ctx context.Context, obj client.Object) []reconcile.Request {
				if obj.GetNamespace() != r.Namespace {
					return nil
				}
				if _, ok := nameSet[obj.GetName()]; !ok {
					return nil
				}
				return []reconcile.Request{{}}
			},
		)
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named("metricsresources").
		Watches(&corev1.Service{}, mapIfOwned(metricsServiceName)).
		Watches(&corev1.Secret{}, mapIfOwned(metricsTokenSecretName)).
		Watches(&monitoringv1.ServiceMonitor{}, mapIfOwned(metricsServiceMonitorName)).
		Complete(r)
}
