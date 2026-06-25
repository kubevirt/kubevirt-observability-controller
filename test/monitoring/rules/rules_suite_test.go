package rules_test

import (
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kubevirt/kubevirt-observability-controller/test/lib"
)

var (
	kvNamespace string
	image       string
)

func TestRules(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "PrometheusRule E2E Suite")
}

var _ = BeforeSuite(func() {
	image = os.Getenv("IMG")
	Expect(image).ToNot(BeEmpty(), "IMG env var must be set")

	Expect(lib.HasCRD("kubevirts.kubevirt.io")).To(BeTrue(), "KubeVirt CRDs must be installed")
	Expect(lib.HasCRD("prometheusrules.monitoring.coreos.com")).To(BeTrue(),
		"Prometheus Operator CRDs must be installed")

	var err error
	kvNamespace, err = lib.FindKubeVirtNamespace()
	Expect(err).ToNot(HaveOccurred())

	err = lib.DeployController(kvNamespace, image)
	Expect(err).ToNot(HaveOccurred())
	DeferCleanup(lib.DeleteController, kvNamespace)

	err = lib.WaitForControllerReady(kvNamespace, 2*time.Minute)
	Expect(err).ToNot(HaveOccurred())
})
