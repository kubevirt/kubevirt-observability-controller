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

var _ = Describe("VMStats Metrics", func() {
	var metricsOutput string

	BeforeEach(func() {
		Eventually(func() (string, error) {
			return metrics.Scrape(kvNamespace, metricsServiceName)
		}, 3*time.Minute, 10*time.Second).Should(
			SatisfyAll(
				Not(BeEmpty()),
				ContainSubstring("kubevirt_vmi_cpu_usage_seconds_total"),
			),
		)

		var err error
		metricsOutput, err = metrics.Scrape(kvNamespace, metricsServiceName)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should expose CPU metrics", func() {
		Expect(metrics.HasMetric(metricsOutput,
			"kubevirt_vmi_cpu_usage_seconds_total")).To(BeTrue())
	})

	It("should expose memory metrics", func() {
		Expect(metrics.HasMetric(metricsOutput,
			"kubevirt_vmi_memory_resident_bytes")).To(BeTrue())
	})

	It("should expose network metrics", func() {
		Expect(metrics.HasMetric(metricsOutput,
			"kubevirt_vmi_network_receive_bytes_total")).To(BeTrue())
	})

	It("should expose vcpu metrics", func() {
		Expect(metrics.HasMetric(metricsOutput,
			"kubevirt_vmi_vcpu_seconds_total")).To(BeTrue())
	})
})
