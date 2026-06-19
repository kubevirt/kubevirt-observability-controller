package vmstats

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Collector", func() {
	BeforeEach(func() {
		Expect(operatormetrics.CleanRegistry()).To(Succeed())
	})

	It("should register and collect metrics from cache", func() {
		cache := NewStatsCache()
		err := RegisterCollector(cache, nil)
		Expect(err).ToNot(HaveOccurred())

		vmi := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "vm1", Namespace: "ns1"},
			Status:     k6tv1.VirtualMachineInstanceStatus{NodeName: "node1"},
		}
		stats := &VMStats{
			DomainStats: DomainStats{
				Cpu: &DomainStatsCPU{TimeSet: true, Time: 1_000_000_000},
			},
		}
		cache.Store("ns1/vm1", vmi, stats)

		collector := vmStatsCollector(cache)
		results := collector.CollectCallback()
		Expect(results).ToNot(BeEmpty())

		var cpuFound bool
		for _, r := range results {
			if r.Metric.GetOpts().Name == "kubevirt_vmi_cpu_usage_seconds_total" {
				cpuFound = true
				Expect(r.Value).To(Equal(1.0))
				Expect(r.ConstLabels).To(HaveKeyWithValue("name", "vm1"))
			}
		}
		Expect(cpuFound).To(BeTrue())
	})

	It("should return empty results when cache is empty", func() {
		cache := NewStatsCache()
		collector := vmStatsCollector(cache)
		results := collector.CollectCallback()
		Expect(results).To(BeEmpty())
	})
})
