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
		err := lib.RedeployController(kvNamespace, image, baselineArgs...)
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

var _ = Describe("Metrics Group Selection", Ordered, func() {
	// vmstats metrics arrive on a 30s poll, so presence has to be waited for
	// rather than scraped once. Absence is only meaningful once something
	// from the same deployment has shown up, so every spec waits on a
	// positive first.
	expectEventually := func(present string, absent ...string) {
		var output string
		Eventually(func(g Gomega) bool {
			out, err := metrics.Scrape(kvNamespace, metricsServiceName)
			g.Expect(err).ToNot(HaveOccurred())
			output = out
			return metrics.HasMetric(out, present)
		}, 5*time.Minute, 10*time.Second).Should(BeTrue(),
			"%s is in a selected group and should be exposed", present)

		for _, name := range absent {
			Expect(metrics.HasMetric(output, name)).To(BeFalse(),
				"%s is not in a selected group and should be filtered out", name)
		}
	}

	redeploy := func(extra ...string) {
		args := append([]string{
			"--enable-vmstats",
			"--vmstats-cert-path=/etc/monitoring/clientcertificates",
		}, extra...)
		Expect(lib.RedeployController(kvNamespace, image, args...)).To(Succeed())
	}

	// Restore the suite baseline rather than the default selection: specs in
	// later files assert on metrics the default set leaves out.
	AfterAll(func() {
		Expect(lib.RedeployController(kvNamespace, image, baselineArgs...)).To(Succeed())
	})

	// The reason the group layer exists: with no allowlist, nothing the
	// controller exposes collides with a series KubeVirt already publishes.
	It("should expose only the non-duplicating groups by default", func() {
		redeploy()

		expectEventually("kubevirt_vmi_guest_hostname",
			"kubevirt_vmi_info",
			"kubevirt_vm_info",
			"kubevirt_vmi_memory_resident_bytes",
			"kubevirt_vmi_filesystem_used_bytes",
		)
		Expect(metrics.HasMetric(mustScrape(), "kubevirt_vmi_guest_os_info")).To(BeTrue(),
			"guest-os is on by default")
	})

	It("should select a whole group by name", func() {
		redeploy("--metrics-allowlist=guest-agent")

		expectEventually("kubevirt_vmi_guest_hostname",
			"kubevirt_vmi_guest_os_info",
			"kubevirt_vmi_info",
		)
		Expect(metrics.HasMetric(mustScrape(), "kubevirt_vmi_guest_timezone")).To(BeTrue(),
			"guest-agent owns the timezone metric too")
	})

	It("should add a group to the defaults with +", func() {
		redeploy("--metrics-allowlist=+vmi-status")

		expectEventually("kubevirt_vmi_info",
			"kubevirt_vm_info",
		)
		Expect(metrics.HasMetric(mustScrape(), "kubevirt_vmi_guest_os_info")).To(BeTrue(),
			"the default groups are still selected alongside vmi-status")
	})
})

func mustScrape() string {
	out, err := metrics.Scrape(kvNamespace, metricsServiceName)
	Expect(err).ToNot(HaveOccurred())
	return out
}
