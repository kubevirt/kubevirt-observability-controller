package vmstats

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("CPU Metrics", func() {
	var report *VMIReport

	BeforeEach(func() {
		vmi := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "vm1", Namespace: "ns1"},
			Status:     k6tv1.VirtualMachineInstanceStatus{NodeName: "node1"},
		}
		report = NewVMIReport(vmi, &VMStats{})
	})

	It("should return empty when DomainStats.Cpu is nil", func() {
		Expect(collectCPUMetrics(report)).To(BeEmpty())
	})

	It("should collect CPU time metrics", func() {
		report.Stats.DomainStats.Cpu = &DomainStatsCPU{
			TimeSet: true, Time: 2_000_000_000,
			UserSet: true, User: 1_000_000_000,
			SystemSet: true, System: 500_000_000,
		}

		results := collectCPUMetrics(report)
		Expect(results).To(HaveLen(3))

		byMetric := make(map[string]float64)
		for _, r := range results {
			byMetric[r.Metric.GetOpts().Name] = r.Value
		}
		Expect(byMetric).To(HaveKeyWithValue("kubevirt_vmi_cpu_usage_seconds_total", 2.0))
		Expect(byMetric).To(HaveKeyWithValue("kubevirt_vmi_cpu_user_usage_seconds_total", 1.0))
		Expect(byMetric).To(HaveKeyWithValue("kubevirt_vmi_cpu_system_usage_seconds_total", 0.5))
	})

	It("should collect load metrics", func() {
		report.Stats.DomainStats.Load = &DomainStatsLoad{
			Load1mSet: true, Load1m: 0.5,
			Load5mSet: true, Load5m: 0.3,
			Load15mSet: true, Load15m: 0.1,
		}

		results := collectCPUMetrics(report)
		Expect(results).To(HaveLen(3))

		byMetric := make(map[string]float64)
		for _, r := range results {
			byMetric[r.Metric.GetOpts().Name] = r.Value
		}
		Expect(byMetric).To(HaveKeyWithValue("kubevirt_vmi_guest_load_1m", 0.5))
		Expect(byMetric).To(HaveKeyWithValue("kubevirt_vmi_guest_load_5m", 0.3))
		Expect(byMetric).To(HaveKeyWithValue("kubevirt_vmi_guest_load_15m", 0.1))
	})

	It("should skip unset fields", func() {
		report.Stats.DomainStats.Cpu = &DomainStatsCPU{
			TimeSet: true, Time: 1_000_000_000,
		}
		results := collectCPUMetrics(report)
		Expect(results).To(HaveLen(1))
	})
})
