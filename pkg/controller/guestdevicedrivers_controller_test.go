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

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/metrics/devicedrivers"
)

const testDriversYAML = `
version: v1alpha1
devices:
  - deviceID: "1041"
    deviceName: "VirtIO network device"
    current:
      - guestOSVersionID: "2022"
        driverVersion: "100.95.104.26200"
      - guestOSVersionID: "2019"
        driverVersion: "100.94.104.25200"
`

var _ = Describe("GuestDeviceDrivers controller", func() {
	const namespace = "virt-observability-controller-system"

	var (
		ctx   context.Context
		cache *devicedrivers.DriversCache
		req   reconcile.Request
	)

	BeforeEach(func() {
		ctx = context.Background()
		cache = devicedrivers.NewDriversCache()
		req = reconcile.Request{NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      GuestDeviceDriversConfigMapName,
		}}
	})

	driversConfigMap := func(data map[string]string) *k8sv1.ConfigMap {
		return &k8sv1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      GuestDeviceDriversConfigMapName,
				Namespace: namespace,
			},
			Data: data,
		}
	}

	reconcilerFor := func(objs ...*k8sv1.ConfigMap) *GuestDeviceDriversReconciler {
		scheme := clientgoscheme.Scheme
		builder := fake.NewClientBuilder().WithScheme(scheme)
		for _, obj := range objs {
			builder = builder.WithObjects(obj)
		}

		return &GuestDeviceDriversReconciler{
			Client:    builder.Build(),
			Namespace: namespace,
			Cache:     cache,
		}
	}

	It("should cache the entries from the ConfigMap", func() {
		r := reconcilerFor(driversConfigMap(map[string]string{
			guestDeviceDriversConfigMapKey: testDriversYAML,
		}))

		_, err := r.Reconcile(ctx, req)

		Expect(err).ToNot(HaveOccurred())
		Expect(cache.List()).To(ConsistOf(
			devicedrivers.DriverEntry{
				DeviceID: "1041", GuestOSVersionID: "2022", DriverVersion: "100.95.104.26200",
			},
			devicedrivers.DriverEntry{
				DeviceID: "1041", GuestOSVersionID: "2019", DriverVersion: "100.94.104.25200",
			},
		))
	})

	It("should clear the cache when the ConfigMap does not exist", func() {
		cache.Store([]devicedrivers.DriverEntry{
			{DeviceID: "1041", GuestOSVersionID: "2019", DriverVersion: "100.94.104.25200"},
		})
		r := reconcilerFor()

		_, err := r.Reconcile(ctx, req)

		Expect(err).ToNot(HaveOccurred())
		Expect(cache.List()).To(BeEmpty())
	})

	It("should keep the last valid entries when the ConfigMap becomes unparseable", func() {
		previous := []devicedrivers.DriverEntry{
			{DeviceID: "1041", GuestOSVersionID: "2019", DriverVersion: "100.94.104.25200"},
		}
		cache.Store(previous)
		r := reconcilerFor(driversConfigMap(map[string]string{
			guestDeviceDriversConfigMapKey: "devices: [oops",
		}))

		_, err := r.Reconcile(ctx, req)

		Expect(err).ToNot(HaveOccurred(), "a bad ConfigMap must not be retried forever")
		Expect(cache.List()).To(Equal(previous))
	})

	It("should keep the last valid entries when the drivers key is missing", func() {
		previous := []devicedrivers.DriverEntry{
			{DeviceID: "1041", GuestOSVersionID: "2019", DriverVersion: "100.94.104.25200"},
		}
		cache.Store(previous)
		r := reconcilerFor(driversConfigMap(map[string]string{"other.yaml": "ignored"}))

		_, err := r.Reconcile(ctx, req)

		Expect(err).ToNot(HaveOccurred())
		Expect(cache.List()).To(Equal(previous))
	})

	It("should replace the cached entries when the ConfigMap changes", func() {
		cache.Store([]devicedrivers.DriverEntry{
			{DeviceID: "9999", GuestOSVersionID: "2016", DriverVersion: "1.0.0.0"},
		})
		r := reconcilerFor(driversConfigMap(map[string]string{
			guestDeviceDriversConfigMapKey: testDriversYAML,
		}))

		_, err := r.Reconcile(ctx, req)

		Expect(err).ToNot(HaveOccurred())
		Expect(cache.List()).ToNot(ContainElement(
			devicedrivers.DriverEntry{
				DeviceID: "9999", GuestOSVersionID: "2016", DriverVersion: "1.0.0.0",
			},
		))
	})

	It("should ignore a ConfigMap with a different name", func() {
		cache.Store([]devicedrivers.DriverEntry{
			{DeviceID: "1041", GuestOSVersionID: "2019", DriverVersion: "100.94.104.25200"},
		})
		r := reconcilerFor(driversConfigMap(map[string]string{
			guestDeviceDriversConfigMapKey: testDriversYAML,
		}))

		_, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{
			Namespace: namespace,
			Name:      "some-other-configmap",
		}})

		Expect(err).ToNot(HaveOccurred())
		Expect(cache.List()).To(ConsistOf(
			devicedrivers.DriverEntry{
				DeviceID: "1041", GuestOSVersionID: "2019", DriverVersion: "100.94.104.25200",
			},
		))
	})
})
