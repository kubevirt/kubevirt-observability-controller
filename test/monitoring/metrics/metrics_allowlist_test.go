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
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kubevirt/kubevirt-observability-controller/test/lib"
	"github.com/kubevirt/kubevirt-observability-controller/test/monitoring/metrics"
)

var _ = Describe("Metrics Allowlist", Ordered, func() {
	AfterAll(func() {
		err := lib.RedeployController(kvNamespace, image,
			"--enable-vmstats",
			"--vmstats-cert-path=/etc/monitoring/clientcertificates")
		Expect(err).ToNot(HaveOccurred())
	})

	It("should only expose the allowlisted metric when metrics-allowlist is set", func() {
		err := lib.RedeployController(kvNamespace, image,
			"--enable-vmstats",
			"--vmstats-cert-path=/etc/monitoring/clientcertificates",
			"--metrics-allowlist=kubevirt_vmi_info")
		Expect(err).ToNot(HaveOccurred())

		var metricsOutput string
		Eventually(func() error {
			var scrapeErr error
			metricsOutput, scrapeErr = metrics.Scrape(kvNamespace, metricsServiceName)
			return scrapeErr
		}, 2*time.Minute, 10*time.Second).Should(Succeed())

		Expect(metrics.HasMetric(metricsOutput, "kubevirt_vmi_info")).To(BeTrue(),
			"allowlisted metric kubevirt_vmi_info should be present")
		Expect(metrics.HasMetric(metricsOutput, "kubevirt_vm_info")).To(BeFalse(),
			"non-allowlisted metric kubevirt_vm_info should be filtered out")
		Expect(metrics.HasMetric(metricsOutput, "controller_runtime_reconcile_total")).To(BeTrue(),
			"controller-runtime metrics should not be affected by the allowlist")
	})

	It("should expose no custom metrics when metrics-allowlist is none", func() {
		err := lib.RedeployController(kvNamespace, image,
			"--enable-vmstats",
			"--vmstats-cert-path=/etc/monitoring/clientcertificates",
			"--metrics-allowlist=none")
		Expect(err).ToNot(HaveOccurred())

		var metricsOutput string
		Eventually(func() error {
			var scrapeErr error
			metricsOutput, scrapeErr = metrics.Scrape(kvNamespace, metricsServiceName)
			return scrapeErr
		}, 2*time.Minute, 10*time.Second).Should(Succeed())

		for _, line := range strings.Split(metricsOutput, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			Expect(line).ToNot(HavePrefix("kubevirt_"),
				"no custom kubevirt_* metrics should be present when allowlist is none")
		}

		Expect(metrics.HasMetric(metricsOutput, "controller_runtime_reconcile_total")).To(BeTrue(),
			"controller-runtime metrics should not be affected by the allowlist")
	})
})
