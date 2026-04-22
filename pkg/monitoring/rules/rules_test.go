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
	It("should register all rules and alerts", func() {
		err := SetupRules("kubevirt")
		Expect(err).ToNot(HaveOccurred())
	})

	It("should build a valid PrometheusRule", func() {
		err := SetupRules("kubevirt")
		Expect(err).ToNot(HaveOccurred())

		pr, err := BuildPrometheusRule("kubevirt-observability-rules", "kubevirt")
		Expect(err).ToNot(HaveOccurred())
		Expect(pr).ToNot(BeNil())
		Expect(pr.Spec.Groups).ToNot(BeEmpty())
		Expect(pr.Name).To(Equal("kubevirt-observability-rules"))
		Expect(pr.Namespace).To(Equal("kubevirt"))
		Expect(pr.Labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "observability-operator"))
	})
})
