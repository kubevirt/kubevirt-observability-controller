package vmstats

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Block Metrics", func() {
	var report *VMIReport

	BeforeEach(func() {
		vmi := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "vm1", Namespace: "ns1"},
			Status:     k6tv1.VirtualMachineInstanceStatus{NodeName: "node1"},
		}
		report = NewVMIReport(vmi, &VMStats{})
	})

	It("should return empty when Block is nil", func() {
		Expect(collectBlockMetrics(report)).To(BeEmpty())
	})

	It("should collect block metrics with drive name", func() {
		report.Stats.DomainStats.Block = []DomainStatsBlock{
			{NameSet: true, Name: "vda", RdReqsSet: true, RdReqs: 100, WrBytesSet: true, WrBytes: 5000},
		}

		results := collectBlockMetrics(report)
		Expect(results).To(HaveLen(2))
		Expect(results[0].ConstLabels).To(HaveKeyWithValue("drive", "vda"))
	})

	It("should use alias when present", func() {
		report.Stats.DomainStats.Block = []DomainStatsBlock{
			{NameSet: true, Name: "vda", Alias: "ua-rootdisk", RdReqsSet: true, RdReqs: 10},
		}

		results := collectBlockMetrics(report)
		Expect(results).To(HaveLen(1))
		Expect(results[0].ConstLabels).To(HaveKeyWithValue("drive", "ua-rootdisk"))
	})

	It("should convert time metrics from nanoseconds to seconds", func() {
		report.Stats.DomainStats.Block = []DomainStatsBlock{
			{NameSet: true, Name: "vda", RdTimesSet: true, RdTimes: 2_000_000_000},
		}

		results := collectBlockMetrics(report)
		Expect(results).To(HaveLen(1))
		Expect(results[0].Value).To(Equal(2.0))
	})

	It("should skip when NameSet is false", func() {
		report.Stats.DomainStats.Block = []DomainStatsBlock{
			{NameSet: false, RdReqsSet: true, RdReqs: 100},
		}
		Expect(collectBlockMetrics(report)).To(BeEmpty())
	})
})
