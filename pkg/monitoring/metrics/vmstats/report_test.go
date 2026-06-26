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

var _ = Describe("VMIReport", func() {
	var (
		vmi    *k6tv1.VirtualMachineInstance
		stats  *VMStats
		report *VMIReport
		gauge  operatormetrics.Metric
	)

	BeforeEach(func() {
		vmi = &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-vmi",
				Namespace: "default",
				Labels: map[string]string{
					"app":              "web",
					"my.label/version": "v1",
				},
			},
			Status: k6tv1.VirtualMachineInstanceStatus{
				NodeName: "node1",
			},
		}
		stats = &VMStats{}
		report = NewVMIReport(vmi, stats)
		gauge = operatormetrics.NewGauge(operatormetrics.MetricOpts{
			Name: "test_metric",
			Help: "test",
		})
	})

	It("should include base labels", func() {
		cr := report.newCollectorResult(gauge, 42.0)
		Expect(cr.ConstLabels).To(HaveKeyWithValue("node", "node1"))
		Expect(cr.ConstLabels).To(HaveKeyWithValue("namespace", "default"))
		Expect(cr.ConstLabels).To(HaveKeyWithValue("name", "test-vmi"))
		Expect(cr.Value).To(Equal(42.0))
	})

	It("should include runtime labels from VMI labels", func() {
		cr := report.newCollectorResult(gauge, 1.0)
		Expect(cr.ConstLabels).To(HaveKeyWithValue("kubernetes_vmi_label_app", "web"))
		Expect(cr.ConstLabels).To(HaveKeyWithValue("kubernetes_vmi_label_my_label_version", "v1"))
	})

	It("should merge additional labels", func() {
		cr := report.newCollectorResultWithLabels(gauge, 1.0, map[string]string{
			"interface": "eth0",
		})
		Expect(cr.ConstLabels).To(HaveKeyWithValue("interface", "eth0"))
		Expect(cr.ConstLabels).To(HaveKeyWithValue("node", "node1"))
	})
})
