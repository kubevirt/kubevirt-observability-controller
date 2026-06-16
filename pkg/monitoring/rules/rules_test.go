package rules

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestRules(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Rules Suite")
}

var _ = Describe("Rules Setup", func() {
	BeforeEach(func() {
		ResetRegistry()
	})

	It("should register all rules and alerts", func() {
		err := SetupRules("kubevirt", nil, nil)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should build a valid PrometheusRule", func() {
		err := SetupRules("kubevirt", nil, nil)
		Expect(err).ToNot(HaveOccurred())

		pr, err := BuildPrometheusRule("kubevirt-observability-rules", "kubevirt")
		Expect(err).ToNot(HaveOccurred())
		Expect(pr).ToNot(BeNil())
		Expect(pr.Spec.Groups).ToNot(BeEmpty())
		Expect(pr.Name).To(Equal("kubevirt-observability-rules"))
		Expect(pr.Namespace).To(Equal("kubevirt"))
		Expect(pr.Labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "kubevirt-observability-controller"))
	})

	Context("allowlist filtering", func() {
		It("should include only allowlisted alerts and recording rules", func() {
			alertsAllowlist := map[string]bool{
				"VirtAPIDown": true,
			}
			recordingRulesAllowlist := map[string]bool{
				"cluster:kubevirt_virt_api_up:sum": true,
			}

			err := SetupRules("kubevirt", alertsAllowlist, recordingRulesAllowlist)
			Expect(err).ToNot(HaveOccurred())

			pr, err := BuildPrometheusRule("kubevirt-observability-rules", "kubevirt")
			Expect(err).ToNot(HaveOccurred())
			Expect(pr).ToNot(BeNil())

			var alertCount, recordingRuleCount int
			for _, g := range pr.Spec.Groups {
				for _, r := range g.Rules {
					if r.Alert != "" {
						alertCount++
					} else {
						recordingRuleCount++
					}
				}
			}

			Expect(alertCount).To(Equal(1))
			Expect(recordingRuleCount).To(Equal(1))
		})

		It("should report no registered rules when both allowlists are empty maps", func() {
			err := SetupRules("kubevirt", map[string]bool{}, map[string]bool{})
			Expect(err).ToNot(HaveOccurred())
			Expect(HasRegisteredRules()).To(BeFalse())
		})
	})
})
