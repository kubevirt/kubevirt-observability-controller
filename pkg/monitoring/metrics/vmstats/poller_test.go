package vmstats

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	k6tv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Poller", func() {
	Describe("groupVMIsByNode", func() {
		It("should group VMIs by node name", func() {
			vmis := []any{
				&k6tv1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "vm1", Namespace: "ns1"},
					Status:     k6tv1.VirtualMachineInstanceStatus{NodeName: "node1"},
				},
				&k6tv1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "vm2", Namespace: "ns1"},
					Status:     k6tv1.VirtualMachineInstanceStatus{NodeName: "node1"},
				},
				&k6tv1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "vm3", Namespace: "ns2"},
					Status:     k6tv1.VirtualMachineInstanceStatus{NodeName: "node2"},
				},
			}

			result := groupVMIsByNode(vmis)
			Expect(result).To(HaveLen(2))
			Expect(result["node1"]).To(HaveLen(2))
			Expect(result["node2"]).To(HaveLen(1))
		})

		It("should skip VMIs without node name", func() {
			vmis := []any{
				&k6tv1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{Name: "vm1", Namespace: "ns1"},
					Status:     k6tv1.VirtualMachineInstanceStatus{},
				},
			}

			result := groupVMIsByNode(vmis)
			Expect(result).To(BeEmpty())
		})
	})

	Describe("findVirtHandlerPodIP", func() {
		It("should find pod IP for node", func() {
			store := cache.NewStore(cache.MetaNamespaceKeyFunc)
			Expect(store.Add(&k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "virt-handler-abc",
					Namespace: "kubevirt",
					Labels:    map[string]string{"kubevirt.io": "virt-handler"},
				},
				Spec: k8sv1.PodSpec{NodeName: "node1"},
				Status: k8sv1.PodStatus{
					PodIP: "10.0.0.5",
					Phase: k8sv1.PodRunning,
				},
			})).To(Succeed())

			ip, err := findVirtHandlerPodIP(store, "node1")
			Expect(err).ToNot(HaveOccurred())
			Expect(ip).To(Equal("10.0.0.5"))
		})

		It("should return error when pod not found for node", func() {
			store := cache.NewStore(cache.MetaNamespaceKeyFunc)
			_, err := findVirtHandlerPodIP(store, "node1")
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("pollOnce", func() {
		It("should poll node VMStats and store successful results in cache", func() {
			bulkResponse := map[string]*VMStatsResult{
				"ns1/vm1": {
					Stats: &VMStats{
						DomainStats: DomainStats{
							Name: "ns1_vm1",
							Cpu:  &DomainStatsCPU{TimeSet: true, Time: 1_000_000_000},
						},
					},
				},
				"ns1/vm2": {
					Stats: &VMStats{
						DomainStats: DomainStats{
							Name: "ns1_vm2",
							Cpu:  &DomainStatsCPU{TimeSet: true, Time: 2_000_000_000},
						},
					},
				},
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.URL.Path).To(Equal("/v1/vmstats"))
				w.Header().Set("Content-Type", "application/json")
				Expect(json.NewEncoder(w).Encode(bulkResponse)).To(Succeed())
			}))
			defer server.Close()

			vmiStore := cache.NewStore(cache.MetaNamespaceKeyFunc)
			vmi1 := &k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "vm1", Namespace: "ns1"},
				Status:     k6tv1.VirtualMachineInstanceStatus{NodeName: "node1"},
			}
			vmi2 := &k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "vm2", Namespace: "ns1"},
				Status:     k6tv1.VirtualMachineInstanceStatus{NodeName: "node1"},
			}
			Expect(vmiStore.Add(vmi1)).To(Succeed())
			Expect(vmiStore.Add(vmi2)).To(Succeed())

			podStore := cache.NewStore(cache.MetaNamespaceKeyFunc)
			Expect(podStore.Add(&k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "virt-handler-abc", Namespace: "kubevirt",
					Labels: map[string]string{"kubevirt.io": "virt-handler"},
				},
				Spec:   k8sv1.PodSpec{NodeName: "node1"},
				Status: k8sv1.PodStatus{PodIP: "10.0.0.5", Phase: k8sv1.PodRunning},
			})).To(Succeed())

			statsCache := NewStatsCache()
			vmClient := NewVMStatsClient(server.Client(), 0)
			vmClient.baseURLOverride = server.URL

			p := NewPoller(PollerConfig{MaxConcurrent: 10}, statsCache, vmClient, vmiStore, podStore)
			p.pollOnce()

			items := statsCache.List()
			Expect(items).To(HaveLen(2))

			names := map[string]bool{}
			for _, item := range items {
				names[item.Stats.DomainStats.Name] = true
			}
			Expect(names).To(HaveKey("ns1_vm1"))
			Expect(names).To(HaveKey("ns1_vm2"))
		})

		It("should skip VMIs with errors in bulk response", func() {
			bulkResponse := map[string]*VMStatsResult{
				"ns1/vm1": {
					Stats: &VMStats{
						DomainStats: DomainStats{Name: "ns1_vm1"},
					},
				},
				"ns1/vm2": {
					Error: "failed to connect to cmd client socket",
				},
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				Expect(json.NewEncoder(w).Encode(bulkResponse)).To(Succeed())
			}))
			defer server.Close()

			vmiStore := cache.NewStore(cache.MetaNamespaceKeyFunc)
			Expect(vmiStore.Add(&k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "vm1", Namespace: "ns1"},
				Status:     k6tv1.VirtualMachineInstanceStatus{NodeName: "node1"},
			})).To(Succeed())
			Expect(vmiStore.Add(&k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "vm2", Namespace: "ns1"},
				Status:     k6tv1.VirtualMachineInstanceStatus{NodeName: "node1"},
			})).To(Succeed())

			podStore := cache.NewStore(cache.MetaNamespaceKeyFunc)
			Expect(podStore.Add(&k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "virt-handler-abc", Namespace: "kubevirt",
					Labels: map[string]string{"kubevirt.io": "virt-handler"},
				},
				Spec:   k8sv1.PodSpec{NodeName: "node1"},
				Status: k8sv1.PodStatus{PodIP: "10.0.0.5", Phase: k8sv1.PodRunning},
			})).To(Succeed())

			statsCache := NewStatsCache()
			vmClient := NewVMStatsClient(server.Client(), 0)
			vmClient.baseURLOverride = server.URL

			p := NewPoller(PollerConfig{MaxConcurrent: 10}, statsCache, vmClient, vmiStore, podStore)
			p.pollOnce()

			items := statsCache.List()
			Expect(items).To(HaveLen(1))
			Expect(items[0].Stats.DomainStats.Name).To(Equal("ns1_vm1"))
		})

		It("should prune stale entries from cache", func() {
			bulkResponse := map[string]*VMStatsResult{
				"ns1/vm1": {
					Stats: &VMStats{
						DomainStats: DomainStats{Name: "ns1_vm1"},
					},
				},
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				Expect(json.NewEncoder(w).Encode(bulkResponse)).To(Succeed())
			}))
			defer server.Close()

			vmiStore := cache.NewStore(cache.MetaNamespaceKeyFunc)
			Expect(vmiStore.Add(&k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "vm1", Namespace: "ns1"},
				Status:     k6tv1.VirtualMachineInstanceStatus{NodeName: "node1"},
			})).To(Succeed())

			podStore := cache.NewStore(cache.MetaNamespaceKeyFunc)
			Expect(podStore.Add(&k8sv1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "virt-handler-abc", Namespace: "kubevirt",
					Labels: map[string]string{"kubevirt.io": "virt-handler"},
				},
				Spec:   k8sv1.PodSpec{NodeName: "node1"},
				Status: k8sv1.PodStatus{PodIP: "10.0.0.5", Phase: k8sv1.PodRunning},
			})).To(Succeed())

			statsCache := NewStatsCache()
			statsCache.Store("ns1/stale-vm", &k6tv1.VirtualMachineInstance{}, &VMStats{})

			vmClient := NewVMStatsClient(server.Client(), 0)
			vmClient.baseURLOverride = server.URL

			p := NewPoller(PollerConfig{MaxConcurrent: 10}, statsCache, vmClient, vmiStore, podStore)
			p.pollOnce()

			items := statsCache.List()
			Expect(items).To(HaveLen(1))
			Expect(items[0].Stats.DomainStats.Name).To(Equal("ns1_vm1"))
		})
	})
})
