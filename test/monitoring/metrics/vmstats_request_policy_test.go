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

	"github.com/kubevirt/kubevirt-observability-controller/test/monitoring/metrics"
)

var _ = Describe("VMStats OS-based request policy", func() {
	var metricsOutput string

	scopedLines := func(metricName string) []metrics.MetricLine {
		var out []metrics.MetricLine
		for _, l := range metrics.FindMetric(metricsOutput, metricName) {
			if metrics.HasLabel(l, "name", "e2e-test-vmi") &&
				metrics.HasLabel(l, "namespace", testVMINamespace) {
				out = append(out, l)
			}
		}
		return out
	}

	BeforeEach(func() {
		Eventually(func(g Gomega) {
			out, err := metrics.Scrape(kvNamespace, metricsServiceName)
			g.Expect(err).ToNot(HaveOccurred())
			metricsOutput = out

			g.Expect(scopedLines("kubevirt_vmi_guest_hostname")).ToNot(BeEmpty(),
				"no guest-agent VMStats poll for e2e-test-vmi has landed yet")
		}, 5*time.Minute, 10*time.Second).Should(Succeed())
	})

	It("classifies the guest as fedora, driving the linux family", func() {
		lines := scopedLines("kubevirt_vmi_guest_os_info")
		Expect(lines).ToNot(BeEmpty())

		found := false
		for _, l := range lines {
			if metrics.HasLabel(l, "os_id", "fedora") {
				found = true
				break
			}
		}
		Expect(found).To(BeTrue(),
			"expected guest OS info with os_id=fedora, driving the linux classification")
	})

	It("requests guest-common categories for the guest", func() {
		Expect(scopedLines("kubevirt_vmi_guest_hostname")).ToNot(BeEmpty())
		Expect(scopedLines("kubevirt_vmi_guest_timezone")).ToNot(BeEmpty())
	})

	It("does not request the windows-family categories for a linux guest", func() {
		Consistently(func(g Gomega) []metrics.MetricLine {
			out, err := metrics.Scrape(kvNamespace, metricsServiceName)
			g.Expect(err).ToNot(HaveOccurred())
			metricsOutput = out
			return append(
				scopedLines("kubevirt_vmi_guest_user_count"),
				scopedLines("kubevirt_vmi_guest_device_driver_timestamp_seconds")...,
			)
		}, 30*time.Second, 10*time.Second).Should(BeEmpty())
	})
})
