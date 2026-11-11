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

package rules_test

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kubevirt/kubevirt-observability-controller/test/lib"
	"github.com/kubevirt/kubevirt-observability-controller/test/monitoring/rules"
)

const prometheusRuleName = "virt-observability-rules"

var _ = Describe("PrometheusRule Reconciliation", func() {
	It("should create the PrometheusRule", func() {
		Eventually(func() error {
			_, err := rules.GetPrometheusRule(kvNamespace, prometheusRuleName)
			return err
		}, 2*time.Minute, 5*time.Second).Should(Succeed())
	})

	It("should have the managed-by label", func() {
		pr, err := rules.GetPrometheusRule(kvNamespace, prometheusRuleName)
		Expect(err).ToNot(HaveOccurred())
		Expect(pr.Labels).To(HaveKeyWithValue(
			"app.kubernetes.io/managed-by", "virt-observability-controller"))
	})

	It("should contain alerts", func() {
		pr, err := rules.GetPrometheusRule(kvNamespace, prometheusRuleName)
		Expect(err).ToNot(HaveOccurred())
		Expect(rules.AlertCount(pr)).To(BeNumerically(">", 0))
	})

	It("should contain recording rules", func() {
		pr, err := rules.GetPrometheusRule(kvNamespace, prometheusRuleName)
		Expect(err).ToNot(HaveOccurred())
		Expect(rules.RecordingRuleCount(pr)).To(BeNumerically(">", 0))
	})

	// The suite deploys without allowlist flags, which selects only the groups
	// KubeVirt does not publish. "Group Selection" covers the virt-* groups
	// by deploying with every group selected.
	It("should contain expected alerts", func() {
		pr, err := rules.GetPrometheusRule(kvNamespace, prometheusRuleName)
		Expect(err).ToNot(HaveOccurred())
		Expect(rules.AlertNames(pr)).To(ConsistOf(
			"OutdatedGuestDeviceDrivers",
			"KubeVirtVMIOrdinalPodInterfaceName",
			"VirtualMachineInstanceHasEphemeralHotplugVolume",
		))
	})

	It("should contain expected recording rules", func() {
		pr, err := rules.GetPrometheusRule(kvNamespace, prometheusRuleName)
		Expect(err).ToNot(HaveOccurred())
		Expect(rules.RecordingRuleNames(pr)).To(ConsistOf(
			"vmi:kubevirt_vmi_guest_device_driver_timestamp_seconds:max",
		))
	})

	It("should have required labels on all alerts", func() {
		pr, err := rules.GetPrometheusRule(kvNamespace, prometheusRuleName)
		Expect(err).ToNot(HaveOccurred())
		for _, g := range pr.Spec.Groups {
			for _, r := range g.Rules {
				if r.Alert == "" {
					continue
				}
				Expect(r.Labels).To(HaveKeyWithValue(
					"kubernetes_operator_part_of", "kubevirt"),
					"alert %s missing part_of label", r.Alert)
				Expect(r.Labels).To(HaveKeyWithValue(
					"kubernetes_operator_component", "kubevirt"),
					"alert %s missing component label", r.Alert)
				Expect(r.Annotations).To(HaveKey("runbook_url"),
					"alert %s missing runbook_url annotation", r.Alert)
			}
		}
	})

	It("should re-create the PrometheusRule after deletion", func() {
		_, err := lib.Kubectl("delete", "prometheusrule",
			prometheusRuleName, "-n", kvNamespace)
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() error {
			_, err := rules.GetPrometheusRule(kvNamespace, prometheusRuleName)
			return err
		}, 2*time.Minute, 5*time.Second).Should(Succeed())
	})
})
