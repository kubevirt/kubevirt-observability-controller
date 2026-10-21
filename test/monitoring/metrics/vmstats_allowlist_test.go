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

var _ = Describe("VMStats Allowlist", Ordered, func() {
	redeployWithAllowlist := func(allowlist string) {
		err := lib.RedeployController(kvNamespace, image,
			"--enable-vmstats",
			"--vmstats-cert-path=/etc/monitoring/clientcertificates",
			"--metrics-allowlist="+allowlist)
		Expect(err).ToNot(HaveOccurred())
	}

	scopedLines := func(output, metricName string) []metrics.MetricLine {
		var out []metrics.MetricLine
		for _, l := range metrics.FindMetric(output, metricName) {
			if metrics.HasLabel(l, "name", "e2e-test-vmi") &&
				metrics.HasLabel(l, "namespace", testVMINamespace) {
				out = append(out, l)
			}
		}
		return out
	}

	// A narrowed request must still carry every command the allowlisted metric
	// needs, so the metric has to keep reporting for the test VMI.
	expectOnlyReported := func(allowlisted string, filtered ...string) {
		var output string
		Eventually(func(g Gomega) []metrics.MetricLine {
			out, err := metrics.Scrape(kvNamespace, metricsServiceName)
			g.Expect(err).ToNot(HaveOccurred())
			output = out
			return scopedLines(out, allowlisted)
		}, 5*time.Minute, 10*time.Second).ShouldNot(BeEmpty(),
			"allowlisted vmstats metric %s should still be collected", allowlisted)

		for _, name := range filtered {
			Expect(metrics.HasMetric(output, name)).To(BeFalse(),
				"non-allowlisted vmstats metric %s should be filtered out", name)
		}
	}

	AfterAll(func() {
		err := lib.RedeployController(kvNamespace, image,
			"--enable-vmstats",
			"--vmstats-cert-path=/etc/monitoring/clientcertificates")
		Expect(err).ToNot(HaveOccurred())
	})

	It("collects a domainStats-backed metric when it is the only one allowed", func() {
		redeployWithAllowlist("kubevirt_vmi_memory_resident_bytes")

		expectOnlyReported("kubevirt_vmi_memory_resident_bytes",
			"kubevirt_vmi_cpu_usage_seconds_total",
			"kubevirt_vmi_guest_hostname",
			"kubevirt_vmi_guest_os_info")
	})

	It("collects a guest-agent-backed metric when it is the only one allowed", func() {
		redeployWithAllowlist("kubevirt_vmi_guest_hostname")

		expectOnlyReported("kubevirt_vmi_guest_hostname",
			"kubevirt_vmi_memory_resident_bytes",
			"kubevirt_vmi_cpu_usage_seconds_total",
			"kubevirt_vmi_guest_os_info",
			"kubevirt_vmi_guest_interface_info")
	})

	It("keeps reporting non-vmstats metrics when no vmstats metric is allowed", func() {
		redeployWithAllowlist("kubevirt_vmi_info")

		var output string
		Eventually(func() error {
			var err error
			output, err = metrics.Scrape(kvNamespace, metricsServiceName)
			return err
		}, 2*time.Minute, 10*time.Second).Should(Succeed())

		Expect(metrics.HasMetric(output, "kubevirt_vmi_info")).To(BeTrue())
		Expect(metrics.HasMetric(output, "kubevirt_vmi_memory_resident_bytes")).To(BeFalse())
		Expect(metrics.HasMetric(output, "kubevirt_vmi_guest_hostname")).To(BeFalse())
	})
})
