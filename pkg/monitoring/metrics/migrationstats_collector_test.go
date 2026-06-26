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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Migration Stats Collector", func() {
	It("should count migrations by phase", func() {
		vmims := []*k6tv1.VirtualMachineInstanceMigration{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "m1", Namespace: "ns1"},
				Spec:       k6tv1.VirtualMachineInstanceMigrationSpec{VMIName: "vmi1"},
				Status: k6tv1.VirtualMachineInstanceMigrationStatus{
					Phase: k6tv1.MigrationPending,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "m2", Namespace: "ns1"},
				Spec:       k6tv1.VirtualMachineInstanceMigrationSpec{VMIName: "vmi2"},
				Status: k6tv1.VirtualMachineInstanceMigrationStatus{
					Phase: k6tv1.MigrationPending,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "m3", Namespace: "ns1"},
				Spec:       k6tv1.VirtualMachineInstanceMigrationSpec{VMIName: "vmi3"},
				Status: k6tv1.VirtualMachineInstanceMigrationStatus{
					Phase: k6tv1.MigrationRunning,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "m4", Namespace: "ns1"},
				Spec:       k6tv1.VirtualMachineInstanceMigrationSpec{VMIName: "vmi4"},
				Status: k6tv1.VirtualMachineInstanceMigrationStatus{
					Phase: k6tv1.MigrationSucceeded,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "m5", Namespace: "ns1"},
				Spec:       k6tv1.VirtualMachineInstanceMigrationSpec{VMIName: "vmi5"},
				Status: k6tv1.VirtualMachineInstanceMigrationStatus{
					Phase: k6tv1.MigrationFailed,
				},
			},
		}

		results := ReportMigrationStats(vmims)

		findGauge := func(metric operatormetrics.Metric) float64 {
			for _, r := range results {
				if r.Metric == metric {
					return r.Value
				}
			}
			return -1
		}

		Expect(findGauge(PendingMigrations)).To(Equal(float64(2)))
		Expect(findGauge(RunningMigrations)).To(Equal(float64(1)))
		Expect(findGauge(SchedulingMigrations)).To(Equal(float64(0)))
		Expect(findGauge(UnsetMigration)).To(Equal(float64(0)))
	})

	It("should report succeeded and failed migrations with labels", func() {
		vmims := []*k6tv1.VirtualMachineInstanceMigration{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "m1", Namespace: "ns1"},
				Spec:       k6tv1.VirtualMachineInstanceMigrationSpec{VMIName: "vmi1"},
				Status: k6tv1.VirtualMachineInstanceMigrationStatus{
					Phase: k6tv1.MigrationSucceeded,
				},
			},
		}

		results := ReportMigrationStats(vmims)

		var found bool
		for _, r := range results {
			if r.Metric == SucceededMigration {
				Expect(r.Labels).To(Equal([]string{"vmi1", "m1", "ns1"}))
				Expect(r.Value).To(Equal(float64(1)))
				found = true
			}
		}
		Expect(found).To(BeTrue())
	})

	It("should return zero counts for empty list", func() {
		results := ReportMigrationStats(nil)
		Expect(results).To(HaveLen(4))
	})
})
