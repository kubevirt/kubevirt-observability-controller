/*
This file is part of the KubeVirt project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Copyright The KubeVirt Authors.
*/

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
			storesRef.Store(&Stores{})
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
			storesRef.Store(&Stores{})
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
			storesRef.Store(&Stores{})
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

	Describe("collectVMIGuestOSInfo", func() {
		It("should collect the guest OS reported in the VMI status", func() {
			vmi := &k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "vmi1", Namespace: "ns1"},
				Status: k6tv1.VirtualMachineInstanceStatus{
					NodeName: "node1",
					GuestOSInfo: k6tv1.VirtualMachineInstanceGuestOSInfo{
						ID:            "fedora",
						Name:          "Fedora Linux",
						Version:       "38",
						KernelRelease: "6.2.0",
						Machine:       "x86_64",
					},
				},
			}

			results := collectVMIGuestOSInfo(vmi)

			Expect(results).To(HaveLen(1))
			Expect(results[0].Value).To(Equal(1.0))
			Expect(results[0].Labels).To(Equal([]string{
				"node1", "ns1", "vmi1",
				"Fedora Linux", "38", "fedora", "6.2.0", "x86_64",
			}))
		})

		It("should report nothing when the VMI status has no guest OS", func() {
			vmi := &k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "vmi1", Namespace: "ns1"},
				Status: k6tv1.VirtualMachineInstanceStatus{
					GuestOSInfo: k6tv1.VirtualMachineInstanceGuestOSInfo{VersionID: "2022"},
				},
			}

			Expect(collectVMIGuestOSInfo(vmi)).To(BeEmpty())
		})
	})

	Describe("collectVMIGuestInterfaces", func() {
		It("should collect one series per guest agent reported interface", func() {
			vmi := &k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "vmi1", Namespace: "ns1"},
				Status: k6tv1.VirtualMachineInstanceStatus{
					NodeName: "node1",
					Interfaces: []k6tv1.VirtualMachineInstanceNetworkInterface{
						{
							Name:          "default",
							InterfaceName: "eth0",
							MAC:           "f6:98:90:b9:bd:70",
							IP:            "10.0.2.2",
							IPs:           []string{"10.0.2.2", "fd10::2"},
							InfoSource:    "domain, guest-agent, multus-status",
						},
						{
							Name:          "secondary",
							InterfaceName: "eth1",
							MAC:           "02:11:22:33:44:55",
							InfoSource:    "guest-agent",
						},
					},
				},
			}

			results := collectVMIGuestInterfaces(vmi)

			Expect(results).To(HaveLen(2))
			Expect(results[0].Value).To(Equal(1.0))
			Expect(results[0].Labels).To(Equal([]string{
				"node1", "ns1", "vmi1",
				"eth0", "default", "f6:98:90:b9:bd:70", "10.0.2.2",
			}))
			Expect(results[1].Labels).To(Equal([]string{
				"node1", "ns1", "vmi1",
				"eth1", "secondary", "02:11:22:33:44:55", "",
			}))
		})

		It("should skip interfaces the guest agent did not report", func() {
			vmi := &k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "vmi1", Namespace: "ns1"},
				Status: k6tv1.VirtualMachineInstanceStatus{
					NodeName: "node1",
					Interfaces: []k6tv1.VirtualMachineInstanceNetworkInterface{
						{
							Name:       "default",
							MAC:        "f6:98:90:b9:bd:70",
							IP:         "10.0.2.2",
							InfoSource: "domain, multus-status",
						},
						{
							Name:          "reported",
							InterfaceName: "eth1",
							MAC:           "02:11:22:33:44:55",
							InfoSource:    "domain,guest-agent",
						},
					},
				},
			}

			results := collectVMIGuestInterfaces(vmi)

			Expect(results).To(HaveLen(1))
			Expect(results[0].Labels).To(Equal([]string{
				"node1", "ns1", "vmi1",
				"eth1", "reported", "02:11:22:33:44:55", "",
			}))
		})

		DescribeTable("should match the guest agent info source exactly",
			func(infoSource string, expectedSeries int) {
				vmi := &k6tv1.VirtualMachineInstance{
					ObjectMeta: metav1.ObjectMeta{
						Name: "vmi1", Namespace: "ns1",
					},
					Status: k6tv1.VirtualMachineInstanceStatus{
						Interfaces: []k6tv1.VirtualMachineInstanceNetworkInterface{
							{Name: "default", InfoSource: infoSource},
						},
					},
				}

				Expect(collectVMIGuestInterfaces(vmi)).
					To(HaveLen(expectedSeries))
			},
			Entry("single source", "guest-agent", 1),
			Entry("comma separated", "domain,guest-agent", 1),
			Entry("space separated", "domain, guest-agent", 1),
			Entry("trailing source", "domain, guest-agent, multus-status", 1),
			Entry("domain only", "domain", 0),
			Entry("multus only", "multus-status", 0),
			Entry("unset", "", 0),
			Entry("substring match", "not-guest-agent", 0),
		)

		It("should report nothing when the VMI status has no interfaces", func() {
			vmi := &k6tv1.VirtualMachineInstance{
				ObjectMeta: metav1.ObjectMeta{Name: "vmi1", Namespace: "ns1"},
			}

			Expect(collectVMIGuestInterfaces(vmi)).To(BeEmpty())
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
})
