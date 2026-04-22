package metrics

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k6tv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("VM Stats Collector", func() {
	Describe("CollectVMsInfo", func() {
		It("should collect VM info with annotations", func() {
			vm := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{Name: "test-vm", Namespace: "default"},
				Spec: k6tv1.VirtualMachineSpec{
					Template: &k6tv1.VirtualMachineInstanceTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Annotations: map[string]string{
								"vm.kubevirt.io/os":       "linux",
								"vm.kubevirt.io/workload": "server",
								"vm.kubevirt.io/flavor":   "large",
							},
						},
					},
				},
				Status: k6tv1.VirtualMachineStatus{
					PrintableStatus: k6tv1.VirtualMachineStatusRunning,
				},
			}

			results := CollectVMsInfo([]*k6tv1.VirtualMachine{vm})
			Expect(results).To(HaveLen(1))
			Expect(results[0].Labels[2]).To(Equal("linux"))
			Expect(results[0].Labels[3]).To(Equal("server"))
			Expect(results[0].Labels[4]).To(Equal("large"))
			Expect(results[0].Labels[8]).To(Equal("running"))
			Expect(results[0].Labels[9]).To(Equal("running"))
		})
	})

	Describe("CollectResourceRequestsAndLimits", func() {
		It("should collect memory requests from domain resources", func() {
			vm := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{Name: "vm1", Namespace: "ns1"},
				Spec: k6tv1.VirtualMachineSpec{
					Template: &k6tv1.VirtualMachineInstanceTemplateSpec{
						Spec: k6tv1.VirtualMachineInstanceSpec{
							Domain: k6tv1.DomainSpec{
								Resources: k6tv1.ResourceRequirements{
									Requests: k8sv1.ResourceList{
										k8sv1.ResourceMemory: resource.MustParse("1Gi"),
									},
								},
							},
						},
					},
				},
			}

			results := CollectResourceRequestsAndLimits([]*k6tv1.VirtualMachine{vm})

			var memFound bool
			for _, r := range results {
				if r.Metric == vmResourceRequests &&
					r.Labels[2] == "memory" && r.Labels[4] == "domain" {
					memFound = true
					q := resource.MustParse("1Gi")
					Expect(r.Value).To(Equal(float64(q.Value())))
				}
			}
			Expect(memFound).To(BeTrue())
		})

		It("should default to 1 CPU when no CPU spec is set", func() {
			vm := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{Name: "vm1", Namespace: "ns1"},
				Spec: k6tv1.VirtualMachineSpec{
					Template: &k6tv1.VirtualMachineInstanceTemplateSpec{
						Spec: k6tv1.VirtualMachineInstanceSpec{
							Domain: k6tv1.DomainSpec{},
						},
					},
				},
			}

			results := CollectResourceRequestsAndLimits([]*k6tv1.VirtualMachine{vm})

			var defaultCPUFound bool
			for _, r := range results {
				if r.Metric == vmResourceRequests &&
					r.Labels[2] == "cpu" && r.Labels[4] == "default" {
					defaultCPUFound = true
					Expect(r.Value).To(Equal(1.0))
				}
			}
			Expect(defaultCPUFound).To(BeTrue())
		})
	})

	Describe("CollectVmsVnicInfo", func() {
		It("should collect vNIC info with pod network", func() {
			vm := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{Name: "vm1", Namespace: "ns1"},
				Spec: k6tv1.VirtualMachineSpec{
					Template: &k6tv1.VirtualMachineInstanceTemplateSpec{
						Spec: k6tv1.VirtualMachineInstanceSpec{
							Domain: k6tv1.DomainSpec{
								Devices: k6tv1.Devices{
									Interfaces: []k6tv1.Interface{
										{
											Name: "default",
											InterfaceBindingMethod: k6tv1.InterfaceBindingMethod{
												Masquerade: &k6tv1.InterfaceMasquerade{},
											},
										},
									},
								},
							},
							Networks: []k6tv1.Network{
								{
									Name: "default",
									NetworkSource: k6tv1.NetworkSource{
										Pod: &k6tv1.PodNetwork{},
									},
								},
							},
						},
					},
				},
			}

			results := CollectVmsVnicInfo([]*k6tv1.VirtualMachine{vm})
			Expect(results).To(HaveLen(1))
			Expect(results[0].Labels[3]).To(Equal("core"))
			Expect(results[0].Labels[4]).To(Equal("pod networking"))
			Expect(results[0].Labels[5]).To(Equal("masquerade"))
		})
	})

	Describe("ReportVMLabels", func() {
		It("should report VM labels as const labels", func() {
			vm := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "vm1",
					Namespace: "ns1",
					Labels: map[string]string{
						"app": "test",
					},
				},
			}

			results := ReportVMLabels(vm)
			Expect(results).To(HaveLen(1))
			Expect(results[0].ConstLabels).To(HaveKeyWithValue("label_app", "test"))
		})

		It("should return nil for VM without labels", func() {
			vm := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{Name: "vm1", Namespace: "ns1"},
			}
			Expect(ReportVMLabels(vm)).To(BeNil())
		})
	})

	Describe("ReportVMStats", func() {
		It("should report timestamp metrics for VMs", func() {
			vm := &k6tv1.VirtualMachine{
				ObjectMeta: metav1.ObjectMeta{Name: "vm1", Namespace: "ns1"},
				Status: k6tv1.VirtualMachineStatus{
					PrintableStatus: k6tv1.VirtualMachineStatusRunning,
				},
			}

			results := ReportVMStats([]*k6tv1.VirtualMachine{vm})

			var timestamps []operatormetrics.CollectorResult
			for _, r := range results {
				for _, tm := range timestampMetrics {
					if r.Metric == tm {
						timestamps = append(timestamps, r)
					}
				}
			}
			Expect(timestamps).To(HaveLen(5))
		})
	})
})
