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

package alerts

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatorrules"
)

func TestAlerts(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Alerts Suite")
}

var _ = Describe("Alerts", func() {
	var registry *operatorrules.Registry

	BeforeEach(func() {
		registry = operatorrules.NewRegistry()
	})

	It("should register all alerts without error", func() {
		err := Register(registry, "kubevirt", nil)
		Expect(err).ToNot(HaveOccurred())

		alerts := registry.ListAlerts()
		Expect(alerts).ToNot(BeEmpty())
	})

	It("should include system alerts", func() {
		err := Register(registry, "kubevirt", nil)
		Expect(err).ToNot(HaveOccurred())
		alerts := registry.ListAlerts()

		alertNames := make(map[string]bool)
		for _, a := range alerts {
			alertNames[a.Alert] = true
		}

		Expect(alertNames).To(HaveKey("LowKVMNodesCount"))
		Expect(alertNames).To(HaveKey("KubeVirtNoAvailableNodesToRunVMs"))
	})

	It("should include VM alerts", func() {
		err := Register(registry, "kubevirt", nil)
		Expect(err).ToNot(HaveOccurred())
		alerts := registry.ListAlerts()

		alertNames := make(map[string]bool)
		for _, a := range alerts {
			alertNames[a.Alert] = true
		}

		Expect(alertNames).To(HaveKey("VirtLauncherPodsStuckFailed"))
		Expect(alertNames).To(HaveKey("VMCannotBeEvicted"))
		Expect(alertNames).To(HaveKey("KubeVirtVMIExcessiveMigrations"))
	})

	It("should include component alerts", func() {
		err := Register(registry, "kubevirt", nil)
		Expect(err).ToNot(HaveOccurred())
		alerts := registry.ListAlerts()

		alertNames := make(map[string]bool)
		for _, a := range alerts {
			alertNames[a.Alert] = true
		}

		Expect(alertNames).To(HaveKey("VirtAPIDown"))
		Expect(alertNames).To(HaveKey("VirtControllerDown"))
		Expect(alertNames).To(HaveKey("VirtOperatorDown"))
		Expect(alertNames).To(HaveKey("VirtHandlerDaemonSetRolloutFailing"))
	})

	It("should have runbook URLs on all alerts", func() {
		err := Register(registry, "kubevirt", nil)
		Expect(err).ToNot(HaveOccurred())
		alerts := registry.ListAlerts()

		for _, a := range alerts {
			Expect(a.Annotations).To(HaveKey("runbook_url"),
				"alert %s should have runbook_url", a.Alert)
		}
	})

	It("should have part_of and component labels on all alerts", func() {
		err := Register(registry, "kubevirt", nil)
		Expect(err).ToNot(HaveOccurred())
		alerts := registry.ListAlerts()

		for _, a := range alerts {
			Expect(a.Labels).To(HaveKeyWithValue("kubernetes_operator_part_of", "kubevirt"),
				"alert %s should have part_of label", a.Alert)
			Expect(a.Labels).To(HaveKeyWithValue("kubernetes_operator_component", "kubevirt"),
				"alert %s should have component label", a.Alert)
		}
	})

	Context("allowlist filtering", func() {
		It("should register only allowlisted alerts", func() {
			allowlist := map[string]bool{
				"VirtAPIDown":      true,
				"VirtOperatorDown": true,
			}
			err := Register(registry, "kubevirt", allowlist)
			Expect(err).ToNot(HaveOccurred())

			alerts := registry.ListAlerts()
			Expect(alerts).To(HaveLen(2))

			alertNames := make(map[string]bool)
			for _, a := range alerts {
				alertNames[a.Alert] = true
			}

			Expect(alertNames).To(HaveKey("VirtAPIDown"))
			Expect(alertNames).To(HaveKey("VirtOperatorDown"))
		})

		It("should register no alerts when allowlist is empty map", func() {
			err := Register(registry, "kubevirt", map[string]bool{})
			Expect(err).ToNot(HaveOccurred())

			alerts := registry.ListAlerts()
			Expect(alerts).To(BeEmpty())
		})

		It("should have labels and annotations on allowlisted alerts", func() {
			allowlist := map[string]bool{
				"VirtAPIDown": true,
			}
			err := Register(registry, "kubevirt", allowlist)
			Expect(err).ToNot(HaveOccurred())

			alerts := registry.ListAlerts()
			Expect(alerts).To(HaveLen(1))
			Expect(alerts[0].Labels).To(HaveKeyWithValue("kubernetes_operator_part_of", "kubevirt"))
			Expect(alerts[0].Labels).To(HaveKeyWithValue("kubernetes_operator_component", "kubevirt"))
			Expect(alerts[0].Annotations).To(HaveKey("runbook_url"))
		})
	})
})
