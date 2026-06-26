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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Memory Metrics", func() {
	var report *VMIReport

	BeforeEach(func() {
		vmi := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "vm1", Namespace: "ns1"},
			Status:     k6tv1.VirtualMachineInstanceStatus{NodeName: "node1"},
		}
		report = NewVMIReport(vmi, &VMStats{})
	})

	It("should return empty when Memory is nil", func() {
		Expect(collectMemoryMetrics(report)).To(BeEmpty())
	})

	It("should collect all memory metrics with KiB to bytes conversion", func() {
		report.Stats.DomainStats.Memory = &DomainStatsMemory{
			RSSSet: true, RSS: 1024,
			AvailableSet: true, Available: 2048,
			UnusedSet: true, Unused: 512,
			CachedSet: true, Cached: 256,
			SwapInSet: true, SwapIn: 100,
			SwapOutSet: true, SwapOut: 50,
			MajorFaultSet: true, MajorFault: 10,
			MinorFaultSet: true, MinorFault: 200,
			ActualBalloonSet: true, ActualBalloon: 4096,
			UsableSet: true, Usable: 3072,
			TotalSet: true, Total: 8192,
		}

		results := collectMemoryMetrics(report)
		Expect(results).To(HaveLen(11))

		byMetric := make(map[string]float64)
		for _, r := range results {
			byMetric[r.Metric.GetOpts().Name] = r.Value
		}

		Expect(byMetric).To(HaveKeyWithValue("kubevirt_vmi_memory_resident_bytes", 1024.0*1024))
		Expect(byMetric).To(HaveKeyWithValue("kubevirt_vmi_memory_available_bytes", 2048.0*1024))
		Expect(byMetric).To(HaveKeyWithValue("kubevirt_vmi_memory_pgmajfault_total", 10.0))
		Expect(byMetric).To(HaveKeyWithValue("kubevirt_vmi_memory_pgminfault_total", 200.0))
	})

	It("should skip unset fields", func() {
		report.Stats.DomainStats.Memory = &DomainStatsMemory{
			RSSSet: true, RSS: 1024,
		}
		results := collectMemoryMetrics(report)
		Expect(results).To(HaveLen(1))
	})
})
