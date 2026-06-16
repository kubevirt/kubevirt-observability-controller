package metrics

import (
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Metrics Setup", func() {
	BeforeEach(func() {
		Expect(operatormetrics.CleanRegistry()).To(Succeed())
	})

	It("should register all collectors when allowlist is nil", func() {
		err := SetupMetrics(&Stores{}, &Indexers{}, nil)
		Expect(err).ToNot(HaveOccurred())

		metrics := ListMetrics()
		Expect(metrics).ToNot(BeEmpty())
	})

	It("should register no collectors when allowlist is empty (none)", func() {
		err := SetupMetrics(&Stores{}, &Indexers{}, map[string]bool{})
		Expect(err).ToNot(HaveOccurred())

		metrics := ListMetrics()
		Expect(metrics).To(BeEmpty())
	})

	It("should register only allowed metrics", func() {
		allowlist := map[string]bool{
			"kubevirt_vm_info":  true,
			"kubevirt_vmi_info": true,
		}
		err := SetupMetrics(&Stores{}, &Indexers{}, allowlist)
		Expect(err).ToNot(HaveOccurred())

		registered := ListMetrics()
		Expect(registered).To(HaveLen(2))
		for _, m := range registered {
			Expect(allowlist).To(HaveKey(m.GetOpts().Name))
		}
	})

	It("should succeed with unknown metric names in allowlist", func() {
		allowlist := map[string]bool{
			"nonexistent_metric": true,
		}
		err := SetupMetrics(&Stores{}, &Indexers{}, allowlist)
		Expect(err).ToNot(HaveOccurred())

		metrics := ListMetrics()
		Expect(metrics).To(BeEmpty())
	})
})
