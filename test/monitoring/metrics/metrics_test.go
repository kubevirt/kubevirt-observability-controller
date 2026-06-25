package metrics_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kubevirt/kubevirt-observability-controller/test/monitoring/metrics"
)

const metricsServiceName = "virt-observability-controller-metrics"

var _ = Describe("Custom Metrics", func() {
	var metricsOutput string

	BeforeEach(func() {
		Eventually(func() error {
			var err error
			metricsOutput, err = metrics.Scrape(kvNamespace, metricsServiceName)
			return err
		}, 2*time.Minute, 10*time.Second).Should(Succeed())
	})

	It("should expose controller-runtime reconcile metrics", func() {
		Expect(metrics.HasMetric(metricsOutput,
			"controller_runtime_reconcile_total")).To(BeTrue())
	})

	It("should expose kubevirt_vmi_info for the test VMI", func() {
		lines := metrics.FindMetric(metricsOutput, "kubevirt_vmi_info")
		Expect(lines).ToNot(BeEmpty())

		found := false
		for _, l := range lines {
			if metrics.HasLabel(l, "name", "e2e-test-vmi") &&
				metrics.HasLabel(l, "namespace", testVMINamespace) {
				found = true
				Expect(l.Labels["phase"]).To(Equal("running"))
				break
			}
		}
		Expect(found).To(BeTrue(),
			"kubevirt_vmi_info for e2e-test-vmi not found")
	})
})
