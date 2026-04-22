package metrics

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Metrics Setup", func() {
	It("should register all collectors without error", func() {
		err := SetupMetrics(&Stores{}, &Indexers{})
		Expect(err).ToNot(HaveOccurred())

		metrics := ListMetrics()
		Expect(metrics).ToNot(BeEmpty())
	})
})
