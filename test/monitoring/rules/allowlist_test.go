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
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	"github.com/kubevirt/kubevirt-observability-controller/test/lib"
	"github.com/kubevirt/kubevirt-observability-controller/test/monitoring/rules"
)

var _ = Describe("Allowlist Filtering", Ordered, func() {
	AfterAll(func() {
		err := lib.RedeployController(kvNamespace, image)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should filter to a single alert when alerts-allowlist is set", func() {
		err := lib.RedeployController(kvNamespace, image,
			"--alerts-allowlist=VirtAPIDown")
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() error {
			_, err := rules.GetPrometheusRule(kvNamespace, prometheusRuleName)
			return err
		}, 2*time.Minute, 5*time.Second).Should(Succeed())

		pr, err := rules.GetPrometheusRule(kvNamespace, prometheusRuleName)
		Expect(err).ToNot(HaveOccurred())
		Expect(rules.AlertNames(pr)).To(ConsistOf("VirtAPIDown"))
		Expect(rules.RecordingRuleCount(pr)).To(BeNumerically(">", 0))
	})

	It("should filter to a single recording rule when recording-rules-allowlist is set",
		func() {
			err := lib.RedeployController(kvNamespace, image,
				"--recording-rules-allowlist=cluster:kubevirt_virt_api_up:sum")
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() error {
				_, err := rules.GetPrometheusRule(kvNamespace, prometheusRuleName)
				return err
			}, 2*time.Minute, 5*time.Second).Should(Succeed())

			pr, err := rules.GetPrometheusRule(kvNamespace, prometheusRuleName)
			Expect(err).ToNot(HaveOccurred())
			Expect(rules.AlertCount(pr)).To(BeNumerically(">", 0))
			Expect(rules.RecordingRuleNames(pr)).To(
				ConsistOf("cluster:kubevirt_virt_api_up:sum"))
		})

	It("should filter both when both allowlists are set", func() {
		err := lib.RedeployController(kvNamespace, image,
			"--alerts-allowlist=VirtAPIDown",
			"--recording-rules-allowlist=cluster:kubevirt_virt_api_up:sum")
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() error {
			_, err := rules.GetPrometheusRule(kvNamespace, prometheusRuleName)
			return err
		}, 2*time.Minute, 5*time.Second).Should(Succeed())

		pr, err := rules.GetPrometheusRule(kvNamespace, prometheusRuleName)
		Expect(err).ToNot(HaveOccurred())
		Expect(rules.AlertNames(pr)).To(ConsistOf("VirtAPIDown"))
		Expect(rules.RecordingRuleNames(pr)).To(
			ConsistOf("cluster:kubevirt_virt_api_up:sum"))
	})

	It("should not create a PrometheusRule when both allowlists are none", func() {
		err := lib.RedeployController(kvNamespace, image,
			"--alerts-allowlist=none",
			"--recording-rules-allowlist=none")
		Expect(err).ToNot(HaveOccurred())

		time.Sleep(30 * time.Second)

		_, err = lib.Kubectl("get", "prometheusrule",
			prometheusRuleName, "-n", kvNamespace)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("Group Selection", Ordered, func() {
	AfterAll(func() {
		err := lib.RedeployController(kvNamespace, image)
		Expect(err).ToNot(HaveOccurred())
	})

	waitForRule := func() *promv1.PrometheusRule {
		Eventually(func() error {
			_, err := rules.GetPrometheusRule(kvNamespace, prometheusRuleName)
			return err
		}, 2*time.Minute, 5*time.Second).Should(Succeed())

		pr, err := rules.GetPrometheusRule(kvNamespace, prometheusRuleName)
		Expect(err).ToNot(HaveOccurred())
		return pr
	}

	// The contract behind running alongside KubeVirt: with no flags, the
	// PrometheusRule carries only what KubeVirt does not publish itself.
	It("should expose only the non-duplicating groups by default", func() {
		err := lib.RedeployController(kvNamespace, image)
		Expect(err).ToNot(HaveOccurred())

		pr := waitForRule()
		Expect(rules.AlertNames(pr)).To(ConsistOf(
			"OutdatedGuestDeviceDrivers",
			"KubeVirtVMIOrdinalPodInterfaceName",
			"VirtualMachineInstanceHasEphemeralHotplugVolume",
		))
		Expect(rules.RecordingRuleNames(pr)).To(ConsistOf(
			"vmi:kubevirt_vmi_guest_device_driver_timestamp_seconds:max",
		))
	})

	It("should select a whole group by name", func() {
		err := lib.RedeployController(kvNamespace, image,
			"--alerts-allowlist=virt-api",
			"--recording-rules-allowlist=none")
		Expect(err).ToNot(HaveOccurred())

		pr := waitForRule()
		Expect(rules.AlertNames(pr)).To(ConsistOf(
			"VirtAPIDown", "LowVirtAPICount",
			"VirtApiRESTErrorsBurst", "KubeVirtDeprecatedAPIRequested",
		))
		Expect(rules.RecordingRuleCount(pr)).To(Equal(0))
	})

	It("should add a group to the defaults with +", func() {
		err := lib.RedeployController(kvNamespace, image,
			"--alerts-allowlist=+virt-handler")
		Expect(err).ToNot(HaveOccurred())

		pr := waitForRule()
		Expect(rules.AlertNames(pr)).To(ConsistOf(
			"OutdatedGuestDeviceDrivers",
			"KubeVirtVMIOrdinalPodInterfaceName",
			"VirtualMachineInstanceHasEphemeralHotplugVolume",
			"VirtHandlerDaemonSetRolloutFailing",
			"VirtHandlerRESTErrorsBurst",
		))
	})

	It("should remove a default group with -", func() {
		err := lib.RedeployController(kvNamespace, image,
			"--alerts-allowlist=-guest-drivers")
		Expect(err).ToNot(HaveOccurred())

		pr := waitForRule()
		Expect(rules.AlertNames(pr)).ToNot(ContainElement("OutdatedGuestDeviceDrivers"))
		Expect(rules.AlertNames(pr)).To(ContainElement("KubeVirtVMIOrdinalPodInterfaceName"))
	})

	// A typo must stop the controller rather than silently narrow the
	// selection, which is the failure the name-only allowlist used to allow.
	It("should refuse to start on an unknown name", func() {
		Expect(lib.RedeployController(kvNamespace, image,
			"--alerts-allowlist=no-such-group")).ToNot(Succeed())

		// A crash-looping pod has no running container to read from between
		// restarts, so fall back to the previous attempt's logs.
		Eventually(func() string {
			logs, err := lib.Kubectl("logs", "-n", kvNamespace,
				"-l", "app="+lib.ControllerName, "--tail=50")
			if err != nil || logs == "" {
				logs, err = lib.Kubectl("logs", "-n", kvNamespace,
					"-l", "app="+lib.ControllerName, "--tail=50", "--previous")
				if err != nil {
					return ""
				}
			}
			return logs
		}, 2*time.Minute, 5*time.Second).Should(ContainSubstring("no-such-group"))
	})

	// The virt-* groups still have to render correctly when selected; the
	// default set no longer covers them.
	It("should render the full rule set when every group is selected", func() {
		err := lib.RedeployController(kvNamespace, image,
			"--alerts-allowlist=all",
			"--recording-rules-allowlist=all")
		Expect(err).ToNot(HaveOccurred())

		pr := waitForRule()
		Expect(rules.AlertNames(pr)).To(ContainElements(
			"VirtAPIDown", "VirtControllerDown", "VirtOperatorDown"))
		Expect(rules.RecordingRuleNames(pr)).To(ContainElements(
			"cluster:kubevirt_virt_api_up:sum",
			"namespace:kubevirt_vm:sum",
			"node:kubevirt_vmi_phase:sum",
		))
	})
})
