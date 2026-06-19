package vmstats

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestVMStats(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "VMStats Suite")
}
