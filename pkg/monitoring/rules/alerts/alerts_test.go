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
		err := Register(registry, "kubevirt")
		Expect(err).ToNot(HaveOccurred())

		alerts := registry.ListAlerts()
		Expect(alerts).ToNot(BeEmpty())
	})

	It("should include system alerts", func() {
		err := Register(registry, "kubevirt")
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
		err := Register(registry, "kubevirt")
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
		err := Register(registry, "kubevirt")
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
		err := Register(registry, "kubevirt")
		Expect(err).ToNot(HaveOccurred())
		alerts := registry.ListAlerts()

		for _, a := range alerts {
			Expect(a.Annotations).To(HaveKey("runbook_url"),
				"alert %s should have runbook_url", a.Alert)
		}
	})

	It("should have part_of and component labels on all alerts", func() {
		err := Register(registry, "kubevirt")
		Expect(err).ToNot(HaveOccurred())
		alerts := registry.ListAlerts()

		for _, a := range alerts {
			Expect(a.Labels).To(HaveKeyWithValue("kubernetes_operator_part_of", "kubevirt"),
				"alert %s should have part_of label", a.Alert)
			Expect(a.Labels).To(HaveKeyWithValue("kubernetes_operator_component", "kubevirt"),
				"alert %s should have component label", a.Alert)
		}
	})
})
