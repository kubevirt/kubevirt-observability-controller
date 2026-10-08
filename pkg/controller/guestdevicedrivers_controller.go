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

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/metrics/devicedrivers"
)

const (
	GuestDeviceDriversConfigMapName = "kubevirt-guest-device-drivers-config"
	guestDeviceDriversConfigMapKey  = "drivers.yaml"
)

type GuestDeviceDriversReconciler struct {
	client.Client
	Namespace string
	Cache     *devicedrivers.DriversCache
}

func (r *GuestDeviceDriversReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if req.Namespace != r.Namespace || req.Name != GuestDeviceDriversConfigMapName {
		return ctrl.Result{}, nil
	}

	cm := &k8sv1.ConfigMap{}
	if err := r.Get(ctx, types.NamespacedName{
		Namespace: r.Namespace,
		Name:      GuestDeviceDriversConfigMapName,
	}, cm); err != nil {
		if !errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		logger.Info("Guest device drivers ConfigMap not found, clearing cached versions",
			"name", GuestDeviceDriversConfigMapName, "namespace", r.Namespace)
		r.Cache.Clear()

		return ctrl.Result{}, nil
	}

	data, ok := cm.Data[guestDeviceDriversConfigMapKey]
	if !ok {
		logger.Info("Guest device drivers ConfigMap has no drivers key, keeping cached versions",
			"name", GuestDeviceDriversConfigMapName, "key", guestDeviceDriversConfigMapKey)
		return ctrl.Result{}, nil
	}

	entries, err := devicedrivers.Parse([]byte(data))
	if err != nil {
		logger.Error(err, "Parsing guest device drivers ConfigMap, keeping cached versions",
			"name", GuestDeviceDriversConfigMapName)
		return ctrl.Result{}, nil
	}

	r.Cache.Store(entries)

	return ctrl.Result{}, nil
}

func (r *GuestDeviceDriversReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		Named("guestdevicedrivers").
		Watches(&k8sv1.ConfigMap{}, handler.EnqueueRequestsFromMapFunc(
			func(ctx context.Context, obj client.Object) []reconcile.Request {
				if obj.GetNamespace() != r.Namespace ||
					obj.GetName() != GuestDeviceDriversConfigMapName {
					return nil
				}

				return []reconcile.Request{{
					NamespacedName: types.NamespacedName{
						Namespace: r.Namespace,
						Name:      GuestDeviceDriversConfigMapName,
					},
				}}
			},
		)).
		Complete(r)
}
