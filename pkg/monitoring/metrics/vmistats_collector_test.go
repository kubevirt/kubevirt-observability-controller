package metrics

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("VMI Stats Collector", func() {
	Describe("CollectVMIInfo", func() {
		It("should collect VMI info with phase and annotations", func() {
			stores = &Stores{}
			vmi := &k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-vmi",
					Namespace: "default",
					Annotations: map[string]string{
						"vm.kubevirt.io/os":       "linux",
						"vm.kubevirt.io/workload": "server",
					},
				},
				Status: k6tv1.VirtualMachineInstanceStatus{
					Phase:    k6tv1.Running,
					NodeName: "node1",
				},
			}

			result := CollectVMIInfo(vmi)
			Expect(result.Value).To(Equal(1.0))
			Expect(result.Labels[0]).To(Equal("node1"))
			Expect(result.Labels[1]).To(Equal("default"))
			Expect(result.Labels[2]).To(Equal("test-vmi"))
			Expect(result.Labels[3]).To(Equal("running"))
			Expect(result.Labels[4]).To(Equal("linux"))
			Expect(result.Labels[5]).To(Equal("server"))
		})

		It("should include guest OS info", func() {
			stores = &Stores{}
			vmi := &k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name: "vmi1", Namespace: "ns1",
				},
				Status: k6tv1.VirtualMachineInstanceStatus{
					Phase: k6tv1.Running,
					GuestOSInfo: k6tv1.VirtualMachineInstanceGuestOSInfo{
						KernelRelease: "5.15.0",
						Machine:       "x86_64",
						Name:          "Fedora",
						VersionID:     "38",
					},
				},
			}

			result := CollectVMIInfo(vmi)
			Expect(result.Labels[9]).To(Equal("5.15.0"))
			Expect(result.Labels[11]).To(Equal("x86_64"))
			Expect(result.Labels[12]).To(Equal("Fedora"))
			Expect(result.Labels[13]).To(Equal("38"))
		})

		It("should report outdated when label is present", func() {
			stores = &Stores{}
			vmi := &k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vmi1",
					Namespace: "ns1",
					Labels: map[string]string{
						k6tv1.OutdatedLauncherImageLabel: "",
					},
				},
				Status: k6tv1.VirtualMachineInstanceStatus{Phase: k6tv1.Running},
			}

			result := CollectVMIInfo(vmi)
			Expect(result.Labels[15]).To(Equal("true"))
		})
	})

	Describe("getEvictionBlocker", func() {
		It("should return 0 when no eviction strategy", func() {
			result := getEvictionBlocker(&k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "vmi1", Namespace: "ns1"},
			})
			Expect(result.Value).To(Equal(0.0))
		})

		It("should return 1 when LiveMigrate but not migratable", func() {
			strategy := k6tv1.EvictionStrategyLiveMigrate
			result := getEvictionBlocker(&k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "vmi1", Namespace: "ns1"},
				Spec: k6tv1.VirtualMachineInstanceSpec{
					EvictionStrategy: &strategy,
				},
			})
			Expect(result.Value).To(Equal(1.0))
		})
	})

	Describe("collectVMIInterfacesInfo", func() {
		It("should collect interface info with IP", func() {
			vmi := &k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "vmi1", Namespace: "ns1"},
				Status: k6tv1.VirtualMachineInstanceStatus{
					NodeName: "node1",
					Interfaces: []k6tv1.VirtualMachineInstanceNetworkInterface{
						{
							Name:          "eth0",
							InterfaceName: "net0",
							IP:            "10.0.0.1",
						},
					},
				},
			}

			results := collectVMIInterfacesInfo(vmi)
			Expect(results).To(HaveLen(1))
			Expect(results[0].Labels[6]).To(Equal("ExternalInterface"))
		})

		It("should report SystemInterface when no IP", func() {
			vmi := &k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "vmi1", Namespace: "ns1"},
				Status: k6tv1.VirtualMachineInstanceStatus{
					NodeName: "node1",
					Interfaces: []k6tv1.VirtualMachineInstanceNetworkInterface{
						{
							Name:          "lo",
							InterfaceName: "lo0",
						},
					},
				},
			}

			results := collectVMIInterfacesInfo(vmi)
			Expect(results).To(HaveLen(1))
			Expect(results[0].Labels[6]).To(Equal("SystemInterface"))
		})
	})

	Describe("CollectVMIMigrationTime", func() {
		It("should return empty when no migration state", func() {
			vmi := &k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "vmi1", Namespace: "ns1"},
			}
			Expect(CollectVMIMigrationTime(vmi)).To(BeEmpty())
		})

		It("should collect start and end timestamps", func() {
			startTime := metav1.NewTime(
				time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			)
			endTime := metav1.NewTime(
				time.Date(2026, 1, 1, 0, 1, 0, 0, time.UTC),
			)
			vmi := &k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "vmi1", Namespace: "ns1"},
				Status: k6tv1.VirtualMachineInstanceStatus{
					NodeName: "node1",
					MigrationState: &k6tv1.VirtualMachineInstanceMigrationState{
						StartTimestamp: &startTime,
						EndTimestamp:   &endTime,
						Completed:      true,
					},
				},
			}

			results := CollectVMIMigrationTime(vmi)
			Expect(results).To(HaveLen(2))
		})
	})

	Describe("collectVMILauncherMemoryOverhead", func() {
		It("should return memory overhead from status", func() {
			overhead := resource.MustParse("256Mi")
			vmi := &k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "vmi1", Namespace: "ns1"},
				Status: k6tv1.VirtualMachineInstanceStatus{
					Memory: &k6tv1.MemoryStatus{
						MemoryOverhead: &overhead,
					},
				},
			}

			result := collectVMILauncherMemoryOverhead(vmi)
			Expect(result.Value).To(Equal(float64(overhead.Value())))
		})

		It("should return 0 when no memory status", func() {
			vmi := &k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "vmi1", Namespace: "ns1"},
			}

			result := collectVMILauncherMemoryOverhead(vmi)
			Expect(result.Value).To(Equal(float64(0)))
		})
	})

	Describe("CollectVMIEphemeralHotplug", func() {
		It("should report when annotation present", func() {
			vmi := &k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vmi1",
					Namespace: "ns1",
					Annotations: map[string]string{
						k6tv1.EphemeralHotplugAnnotation: "temp-vol",
					},
				},
			}

			results := CollectVMIEphemeralHotplug(vmi)
			Expect(results).To(HaveLen(1))
			Expect(results[0].Labels[2]).To(Equal("temp-vol"))
		})

		It("should return nil when no annotation", func() {
			vmi := &k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "vmi1", Namespace: "ns1"},
			}

			Expect(CollectVMIEphemeralHotplug(vmi)).To(BeNil())
		})
	})
})
