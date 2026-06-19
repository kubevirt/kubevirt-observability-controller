package vmstats

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("StatsCache", func() {
	var cache *StatsCache

	BeforeEach(func() {
		cache = NewStatsCache()
	})

	It("should store and list entries", func() {
		vmi := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "vm1", Namespace: "ns1"},
		}
		stats := &VMStats{DomainStats: DomainStats{Name: "ns1_vm1"}}

		cache.Store("ns1/vm1", vmi, stats)

		items := cache.List()
		Expect(items).To(HaveLen(1))
		Expect(items[0].VMI.Name).To(Equal("vm1"))
		Expect(items[0].Stats.DomainStats.Name).To(Equal("ns1_vm1"))
	})

	It("should overwrite existing entries", func() {
		vmi := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "vm1", Namespace: "ns1"},
		}
		cache.Store("ns1/vm1", vmi, &VMStats{DomainStats: DomainStats{Name: "old"}})
		cache.Store("ns1/vm1", vmi, &VMStats{DomainStats: DomainStats{Name: "new"}})

		items := cache.List()
		Expect(items).To(HaveLen(1))
		Expect(items[0].Stats.DomainStats.Name).To(Equal("new"))
	})

	It("should prune stale entries", func() {
		vmi1 := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "vm1", Namespace: "ns1"},
		}
		vmi2 := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "vm2", Namespace: "ns1"},
		}
		cache.Store("ns1/vm1", vmi1, &VMStats{})
		cache.Store("ns1/vm2", vmi2, &VMStats{})

		cache.Prune(map[string]bool{"ns1/vm1": true})

		items := cache.List()
		Expect(items).To(HaveLen(1))
		Expect(items[0].VMI.Name).To(Equal("vm1"))
	})

	It("should return empty list when no entries", func() {
		Expect(cache.List()).To(BeEmpty())
	})

	It("should remove single entry", func() {
		vmi := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "vm1", Namespace: "ns1"},
		}
		cache.Store("ns1/vm1", vmi, &VMStats{})
		cache.Remove("ns1/vm1")

		Expect(cache.List()).To(BeEmpty())
	})
})
