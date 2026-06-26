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
