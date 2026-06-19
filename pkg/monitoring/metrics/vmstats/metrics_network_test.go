package vmstats

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Network Metrics", func() {
	var report *VMIReport

	BeforeEach(func() {
		vmi := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "vm1", Namespace: "ns1"},
			Status:     k6tv1.VirtualMachineInstanceStatus{NodeName: "node1"},
		}
		report = NewVMIReport(vmi, &VMStats{})
	})

	It("should return empty when Net is nil", func() {
		Expect(collectNetworkMetrics(report)).To(BeEmpty())
	})

	It("should collect network metrics with interface name", func() {
		report.Stats.DomainStats.Net = []DomainStatsNet{
			{NameSet: true, Name: "vnet0", RxBytesSet: true, RxBytes: 1000, TxBytesSet: true, TxBytes: 2000},
		}

		results := collectNetworkMetrics(report)
		Expect(results).To(HaveLen(2))
		Expect(results[0].ConstLabels).To(HaveKeyWithValue("interface", "vnet0"))
	})

	It("should use alias when set", func() {
		report.Stats.DomainStats.Net = []DomainStatsNet{
			{NameSet: true, Name: "vnet0", AliasSet: true, Alias: "ua-default", RxBytesSet: true, RxBytes: 100},
		}

		results := collectNetworkMetrics(report)
		Expect(results).To(HaveLen(1))
		Expect(results[0].ConstLabels).To(HaveKeyWithValue("interface", "ua-default"))
	})

	It("should skip when NameSet is false", func() {
		report.Stats.DomainStats.Net = []DomainStatsNet{
			{NameSet: false, RxBytesSet: true, RxBytes: 100},
		}
		Expect(collectNetworkMetrics(report)).To(BeEmpty())
	})
})
