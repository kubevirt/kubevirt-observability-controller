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

package metrics_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kubevirt/kubevirt-observability-controller/test/lib"
	"github.com/kubevirt/kubevirt-observability-controller/test/monitoring/metrics"
)

const driverVersionMetric = "kubevirt_guest_device_driver_latest_version_info"

const initialDriversYAML = `version: v1alpha1
devices:
  - deviceID: "1041"
    deviceName: "VirtIO network device"
    current:
      - guestOSVersionID: "2022"
        driverVersion: "100.95.104.26200"
      - guestOSVersionID: "2022"
        driverVersion: "100.95.104.26100"
      - guestOSVersionID: "2019"
        driverVersion: "100.94.104.25200"`

const updatedDriversYAML = `version: v1alpha1
devices:
  - deviceID: "1041"
    deviceName: "VirtIO network device"
    current:
      - guestOSVersionID: "2022"
        driverVersion: "100.95.104.26300"`

type driverSeries struct {
	DeviceID         string
	GuestOSVersionID string
	DriverVersion    string
}

func driverVersions() ([]driverSeries, error) {
	output, err := metrics.Scrape(kvNamespace, metricsServiceName)
	if err != nil {
		return nil, err
	}

	var series []driverSeries
	for _, line := range metrics.FindMetric(output, driverVersionMetric) {
		series = append(series, driverSeries{
			DeviceID:         line.Labels["device_id"],
			GuestOSVersionID: line.Labels["guest_os_version_id"],
			DriverVersion:    line.Labels["driver_version"],
		})
	}

	return series, nil
}

var _ = Describe("Guest Device Driver Versions", Ordered, func() {
	BeforeAll(func() {
		Expect(lib.ApplyGuestDriversConfigMap(kvNamespace, initialDriversYAML)).To(Succeed())
		DeferCleanup(lib.DeleteGuestDriversConfigMap, kvNamespace)
	})

	It("should expose one series per configured driver version", func() {
		Eventually(driverVersions, 2*time.Minute, 5*time.Second).Should(ConsistOf(
			driverSeries{DeviceID: "1041", GuestOSVersionID: "2022", DriverVersion: "100.95.104.26200"},
			driverSeries{DeviceID: "1041", GuestOSVersionID: "2022", DriverVersion: "100.95.104.26100"},
			driverSeries{DeviceID: "1041", GuestOSVersionID: "2019", DriverVersion: "100.94.104.25200"},
		))
	})

	It("should pick up ConfigMap edits without a restart", func() {
		Expect(lib.ApplyGuestDriversConfigMap(kvNamespace, updatedDriversYAML)).To(Succeed())

		Eventually(driverVersions, 2*time.Minute, 5*time.Second).Should(ConsistOf(
			driverSeries{DeviceID: "1041", GuestOSVersionID: "2022", DriverVersion: "100.95.104.26300"},
		))
	})

	It("should stop reporting versions once the ConfigMap is deleted", func() {
		Expect(lib.DeleteGuestDriversConfigMap(kvNamespace)).To(Succeed())

		Eventually(driverVersions, 2*time.Minute, 5*time.Second).Should(BeEmpty())
	})
})
