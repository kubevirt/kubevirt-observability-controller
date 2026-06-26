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

var _ = Describe("Dirty Rate Metrics", func() {
	var report *VMIReport

	BeforeEach(func() {
		vmi := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "vm1", Namespace: "ns1"},
			Status:     k6tv1.VirtualMachineInstanceStatus{NodeName: "node1"},
		}
		report = NewVMIReport(vmi, &VMStats{})
	})

	It("should return empty when DirtyRate is nil", func() {
		Expect(collectDirtyRateMetrics(report)).To(BeEmpty())
	})

	It("should return empty when MegabytesPerSecondSet is false", func() {
		report.Stats.DomainStats.DirtyRate = &DomainStatsDirtyRate{
			MegabytesPerSecondSet: false,
		}
		Expect(collectDirtyRateMetrics(report)).To(BeEmpty())
	})

	It("should convert MBps to bytes per second", func() {
		report.Stats.DomainStats.DirtyRate = &DomainStatsDirtyRate{
			MegabytesPerSecondSet: true,
			MegabytesPerSecond:    10,
		}

		results := collectDirtyRateMetrics(report)
		Expect(results).To(HaveLen(1))
		Expect(results[0].Value).To(Equal(float64(10 * 1024 * 1024)))
		Expect(results[0].Metric.GetOpts().Name).To(Equal("kubevirt_vmi_dirty_rate_bytes_per_second"))
	})
})
