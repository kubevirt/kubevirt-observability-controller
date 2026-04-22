package recordingrules

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatorrules"
)

func TestRecordingRules(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Recording Rules Suite")
}

var _ = Describe("Recording Rules", func() {
	var registry *operatorrules.Registry

	BeforeEach(func() {
		registry = operatorrules.NewRegistry()
	})

	It("should register all recording rules without error", func() {
		err := Register(registry, "kubevirt")
		Expect(err).ToNot(HaveOccurred())

		rules := registry.ListRecordingRules()
		Expect(rules).ToNot(BeEmpty())
	})

	It("should include node recording rules", func() {
		err := Register(registry, "kubevirt")
		Expect(err).ToNot(HaveOccurred())
		rules := registry.ListRecordingRules()

		ruleNames := make(map[string]bool)
		for _, r := range rules {
			ruleNames[r.GetOpts().Name] = true
		}

		Expect(ruleNames).To(HaveKey("cluster:kubevirt_non_schedulable_nodes:sum"))
		Expect(ruleNames).To(HaveKey("cluster:kubevirt_nodes_allocatable:count"))
		Expect(ruleNames).To(HaveKey("cluster:kubevirt_nodes_with_kvm:count"))
	})

	It("should include VM recording rules", func() {
		err := Register(registry, "kubevirt")
		Expect(err).ToNot(HaveOccurred())
		rules := registry.ListRecordingRules()

		ruleNames := make(map[string]bool)
		for _, r := range rules {
			ruleNames[r.GetOpts().Name] = true
		}

		Expect(ruleNames).To(HaveKey("namespace:kubevirt_vm:sum"))
	})

	It("should include VMI recording rules", func() {
		err := Register(registry, "kubevirt")
		Expect(err).ToNot(HaveOccurred())
		rules := registry.ListRecordingRules()

		ruleNames := make(map[string]bool)
		for _, r := range rules {
			ruleNames[r.GetOpts().Name] = true
		}

		Expect(ruleNames).To(HaveKey("node:kubevirt_vmi_phase:sum"))
		Expect(ruleNames).To(HaveKey("vmi:kubevirt_vmi_memory_used_bytes:sum"))
	})

	It("should include deprecated recording rules", func() {
		err := Register(registry, "kubevirt")
		Expect(err).ToNot(HaveOccurred())
		rules := registry.ListRecordingRules()

		ruleNames := make(map[string]bool)
		for _, r := range rules {
			ruleNames[r.GetOpts().Name] = true
		}

		Expect(ruleNames).To(HaveKey("kubevirt_vmi_phase_count"))
		Expect(ruleNames).To(HaveKey("kubevirt_allocatable_nodes"))
	})

	It("should include virt component recording rules with namespace", func() {
		err := Register(registry, "test-namespace")
		Expect(err).ToNot(HaveOccurred())
		rules := registry.ListRecordingRules()

		ruleNames := make(map[string]bool)
		for _, r := range rules {
			ruleNames[r.GetOpts().Name] = true
		}

		Expect(ruleNames).To(HaveKey("cluster:kubevirt_virt_api_up:sum"))
		Expect(ruleNames).To(HaveKey("cluster:kubevirt_virt_controller_up:sum"))
		Expect(ruleNames).To(HaveKey("cluster:kubevirt_virt_handler_up:sum"))
	})
})
