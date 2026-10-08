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

package devicedrivers

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
)

var _ = Describe("Collector", func() {
	var cache *DriversCache

	BeforeEach(func() {
		Expect(operatormetrics.CleanRegistry()).To(Succeed())
		cache = NewDriversCache()
	})

	It("should report one sample per cached entry", func() {
		cache.Store([]DriverEntry{
			{DeviceID: "1041", GuestOSVersionID: "2022", DriverVersion: "100.95.104.26200"},
			{DeviceID: "1041", GuestOSVersionID: "2019", DriverVersion: "100.94.104.25200"},
		})

		results := driversCollector(cache).CollectCallback()

		Expect(results).To(HaveLen(2))
		for _, r := range results {
			Expect(r.Metric.GetOpts().Name).To(Equal("kubevirt_guest_device_driver_latest_version_info"))
			Expect(r.Value).To(Equal(1.0))
		}
		Expect(results[0].Labels).To(Equal([]string{"1041", "2022", "100.95.104.26200"}))
		Expect(results[1].Labels).To(Equal([]string{"1041", "2019", "100.94.104.25200"}))
	})

	It("should report nothing when the cache is empty", func() {
		Expect(driversCollector(cache).CollectCallback()).To(BeEmpty())
	})

	It("should register the metric with the documented labels", func() {
		Expect(RegisterCollector(cache, nil)).To(Succeed())

		Expect(operatormetrics.ListMetrics()).To(ContainElement(
			HaveField("GetOpts().Name", "kubevirt_guest_device_driver_latest_version_info"),
		))
		Expect(latestVersionInfo.GetType()).To(Equal(operatormetrics.GaugeVecType))
	})

	It("should not register the collector when the allowlist excludes the metric", func() {
		Expect(RegisterCollector(cache, map[string]bool{"kubevirt_vmi_info": true})).To(Succeed())

		Expect(operatormetrics.ListMetrics()).To(BeEmpty())
	})

	It("should register the collector when the allowlist includes the metric", func() {
		Expect(RegisterCollector(cache, map[string]bool{
			"kubevirt_guest_device_driver_latest_version_info": true,
		})).To(Succeed())

		Expect(operatormetrics.ListMetrics()).To(HaveLen(1))
	})
})
