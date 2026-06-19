package vmstats

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Dirty Rate Metrics", func() {
	var report *VMIReport

	BeforeEach(func() {
		vmi := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "vm1", Namespace: "ns1"},
			Status:     k6tv1.VirtualMachineInstanceStatus{NodeName: "node1"},
		}
		report = NewVMIReport(vmi, &VMStats{})
	})

	It("should return empty when DirtyRate is nil", func() {
		Expect(collectDirtyRateMetrics(report)).To(BeEmpty())
	})

	It("should return empty when MegabytesPerSecondSet is false", func() {
		report.Stats.DomainStats.DirtyRate = &DomainStatsDirtyRate{
			MegabytesPerSecondSet: false,
		}
		Expect(collectDirtyRateMetrics(report)).To(BeEmpty())
	})

	It("should convert MBps to bytes per second", func() {
		report.Stats.DomainStats.DirtyRate = &DomainStatsDirtyRate{
			MegabytesPerSecondSet: true,
			MegabytesPerSecond:    10,
		}

		results := collectDirtyRateMetrics(report)
		Expect(results).To(HaveLen(1))
		Expect(results[0].Value).To(Equal(float64(10 * 1024 * 1024)))
		Expect(results[0].Metric.GetOpts().Name).To(Equal("kubevirt_vmi_dirty_rate_bytes_per_second"))
	})
})
