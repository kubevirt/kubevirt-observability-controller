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

package vmstats

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("vCPU Metrics", func() {
	var report *VMIReport

	BeforeEach(func() {
		vmi := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "vm1", Namespace: "ns1"},
			Status:     k6tv1.VirtualMachineInstanceStatus{NodeName: "node1"},
		}
		report = NewVMIReport(vmi, &VMStats{})
	})

	It("should return empty when Vcpu is nil", func() {
		Expect(collectVCPUMetrics(report)).To(BeEmpty())
	})

	It("should collect metrics for multiple vcpus", func() {
		report.Stats.DomainStats.Vcpu = []DomainStatsVcpu{
			{StateSet: true, State: VCPURunning, TimeSet: true, Time: 1_000_000_000, WaitSet: true, Wait: 500_000_000},
			{StateSet: true, State: VCPUBlocked, TimeSet: true, Time: 2_000_000_000, DelaySet: true, Delay: 100_000_000},
		}

		results := collectVCPUMetrics(report)
		Expect(results).To(HaveLen(4))

		var timeResults []operatormetrics.CollectorResult
		for _, r := range results {
			if r.Metric.GetOpts().Name == "kubevirt_vmi_vcpu_seconds_total" {
				timeResults = append(timeResults, r)
			}
		}
		Expect(timeResults).To(HaveLen(2))
		Expect(timeResults[0].ConstLabels).To(HaveKeyWithValue("id", "0"))
		Expect(timeResults[0].ConstLabels).To(HaveKeyWithValue("state", "running"))
		Expect(timeResults[1].ConstLabels).To(HaveKeyWithValue("id", "1"))
		Expect(timeResults[1].ConstLabels).To(HaveKeyWithValue("state", "blocked"))
	})
})
