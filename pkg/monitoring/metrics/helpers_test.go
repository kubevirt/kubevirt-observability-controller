package metrics

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8sv1 "k8s.io/api/core/v1"

	k6tv1 "kubevirt.io/api/core/v1"
)

func TestHelpers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Helpers Suite")
}

var _ = Describe("Helper functions", func() {
	Describe("GetNumberOfVCPUs", func() {
		It("should return cores * threads * sockets", func() {
			cpu := &k6tv1.CPU{Cores: 2, Threads: 2, Sockets: 1}
			Expect(GetNumberOfVCPUs(cpu)).To(Equal(int64(4)))
		})

		It("should default unset values to 1", func() {
			cpu := &k6tv1.CPU{Cores: 4}
			Expect(GetNumberOfVCPUs(cpu)).To(Equal(int64(4)))
		})

		It("should return 1 for nil CPU", func() {
			Expect(GetNumberOfVCPUs(nil)).To(Equal(int64(1)))
		})

		It("should return 1 for empty CPU", func() {
			Expect(GetNumberOfVCPUs(&k6tv1.CPU{})).To(Equal(int64(1)))
		})
	})

	Describe("GetSystemInfoFromAnnotations", func() {
		It("should extract os, workload, flavor from annotations", func() {
			annotations := map[string]string{
				"vm.kubevirt.io/os":       "linux",
				"vm.kubevirt.io/workload": "server",
				"vm.kubevirt.io/flavor":   "large",
			}
			os, workload, flavor := GetSystemInfoFromAnnotations(annotations)
			Expect(os).To(Equal("linux"))
			Expect(workload).To(Equal("server"))
			Expect(flavor).To(Equal("large"))
		})

		It("should return empty strings for missing annotations", func() {
			os, workload, flavor := GetSystemInfoFromAnnotations(nil)
			Expect(os).To(Equal(""))
			Expect(workload).To(Equal(""))
			Expect(flavor).To(Equal(""))
		})
	})

	Describe("GetVMStatusGroup", func() {
		It("should return running for Running status", func() {
			Expect(GetVMStatusGroup(k6tv1.VirtualMachineStatusRunning)).To(Equal("running"))
		})

		It("should return starting for Starting status", func() {
			Expect(GetVMStatusGroup(k6tv1.VirtualMachineStatusStarting)).To(Equal("starting"))
		})

		It("should return error for CrashLoopBackOff status", func() {
			Expect(GetVMStatusGroup(k6tv1.VirtualMachineStatusCrashLoopBackOff)).To(Equal("error"))
		})

		It("should return non_running for Stopped status", func() {
			Expect(GetVMStatusGroup(k6tv1.VirtualMachineStatusStopped)).To(Equal("non_running"))
		})

		It("should return migrating for Migrating status", func() {
			Expect(GetVMStatusGroup(k6tv1.VirtualMachineStatusMigrating)).To(Equal("migrating"))
		})
	})

	Describe("SanitizeLabelName", func() {
		It("should replace invalid characters with underscore", func() {
			Expect(SanitizeLabelName("foo.bar/baz")).To(Equal("foo_bar_baz"))
		})

		It("should prefix with underscore if starts with number", func() {
			Expect(SanitizeLabelName("123abc")).To(Equal("_123abc"))
		})
	})

	Describe("NamespacedKey", func() {
		It("should join namespace and name", func() {
			Expect(NamespacedKey("ns", "name")).To(Equal("ns/name"))
		})
	})

	Describe("IsVMIEvictable", func() {
		It("should return true when no eviction strategy is set", func() {
			vmi := &k6tv1.VirtualMachineInstance{}
			Expect(IsVMIEvictable(vmi)).To(BeTrue())
		})

		It("should return false when LiveMigrate but not migratable", func() {
			strategy := k6tv1.EvictionStrategyLiveMigrate
			vmi := &k6tv1.VirtualMachineInstance{
				Spec: k6tv1.VirtualMachineInstanceSpec{
					EvictionStrategy: &strategy,
				},
			}
			Expect(IsVMIEvictable(vmi)).To(BeFalse())
		})

		It("should return true when LiveMigrate and migratable condition is true", func() {
			strategy := k6tv1.EvictionStrategyLiveMigrate
			vmi := &k6tv1.VirtualMachineInstance{
				Spec: k6tv1.VirtualMachineInstanceSpec{
					EvictionStrategy: &strategy,
				},
				Status: k6tv1.VirtualMachineInstanceStatus{
					Conditions: []k6tv1.VirtualMachineInstanceCondition{
						{
							Type:   k6tv1.VirtualMachineInstanceIsMigratable,
							Status: k8sv1.ConditionTrue,
						},
					},
				},
			}
			Expect(IsVMIEvictable(vmi)).To(BeTrue())
		})
	})

	Describe("IsVMIOutdated", func() {
		It("should return false when no outdated label", func() {
			vmi := &k6tv1.VirtualMachineInstance{}
			Expect(IsVMIOutdated(vmi)).To(BeFalse())
		})

		It("should return true when outdated label present", func() {
			vmi := &k6tv1.VirtualMachineInstance{}
			vmi.Labels = map[string]string{
				k6tv1.OutdatedLauncherImageLabel: "",
			}
			Expect(IsVMIOutdated(vmi)).To(BeTrue())
		})
	})

	Describe("GetBinding", func() {
		It("should return core/masquerade for masquerade binding", func() {
			iface := k6tv1.Interface{
				InterfaceBindingMethod: k6tv1.InterfaceBindingMethod{
					Masquerade: &k6tv1.InterfaceMasquerade{},
				},
			}
			bt, bn := GetBinding(iface)
			Expect(bt).To(Equal(BindingTypeCore))
			Expect(bn).To(Equal("masquerade"))
		})

		It("should return plugin binding for custom binding", func() {
			iface := k6tv1.Interface{Binding: &k6tv1.PluginBinding{Name: "my-plugin"}}
			bt, bn := GetBinding(iface)
			Expect(bt).To(Equal(BindingTypePlugin))
			Expect(bn).To(Equal("my-plugin"))
		})
	})

	Describe("GetNetworkName", func() {
		It("should return pod networking for pod network", func() {
			networks := []k6tv1.Network{
				{Name: "default", NetworkSource: k6tv1.NetworkSource{Pod: &k6tv1.PodNetwork{}}},
			}
			name, found := GetNetworkName("default", networks)
			Expect(found).To(BeTrue())
			Expect(name).To(Equal("pod networking"))
		})

		It("should return multus network name", func() {
			networks := []k6tv1.Network{
				{Name: "net1", NetworkSource: k6tv1.NetworkSource{Multus: &k6tv1.MultusNetwork{NetworkName: "my-net"}}},
			}
			name, found := GetNetworkName("net1", networks)
			Expect(found).To(BeTrue())
			Expect(name).To(Equal("my-net"))
		})

		It("should return false when no match", func() {
			_, found := GetNetworkName("missing", nil)
			Expect(found).To(BeFalse())
		})
	})

	Describe("FetchResourceName", func() {
		It("should return other when store is nil", func() {
			Expect(FetchResourceName("key", nil)).To(Equal(Other))
		})
	})
})
