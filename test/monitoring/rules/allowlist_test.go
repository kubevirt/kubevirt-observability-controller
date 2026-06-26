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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kubevirt/kubevirt-observability-controller/test/lib"
	"github.com/kubevirt/kubevirt-observability-controller/test/monitoring/rules"
)

var _ = Describe("Allowlist Filtering", Ordered, func() {
	AfterAll(func() {
		err := lib.RedeployController(kvNamespace, image)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should filter to a single alert when alerts-allowlist is set", func() {
		err := lib.RedeployController(kvNamespace, image,
			"--alerts-allowlist=VirtAPIDown")
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() error {
			_, err := rules.GetPrometheusRule(kvNamespace, prometheusRuleName)
			return err
		}, 2*time.Minute, 5*time.Second).Should(Succeed())

		pr, err := rules.GetPrometheusRule(kvNamespace, prometheusRuleName)
		Expect(err).ToNot(HaveOccurred())
		Expect(rules.AlertNames(pr)).To(ConsistOf("VirtAPIDown"))
		Expect(rules.RecordingRuleCount(pr)).To(BeNumerically(">", 0))
	})

	It("should filter to a single recording rule when recording-rules-allowlist is set",
		func() {
			err := lib.RedeployController(kvNamespace, image,
				"--recording-rules-allowlist=cluster:kubevirt_virt_api_up:sum")
			Expect(err).ToNot(HaveOccurred())

			Eventually(func() error {
				_, err := rules.GetPrometheusRule(kvNamespace, prometheusRuleName)
				return err
			}, 2*time.Minute, 5*time.Second).Should(Succeed())

			pr, err := rules.GetPrometheusRule(kvNamespace, prometheusRuleName)
			Expect(err).ToNot(HaveOccurred())
			Expect(rules.AlertCount(pr)).To(BeNumerically(">", 0))
			Expect(rules.RecordingRuleNames(pr)).To(
				ConsistOf("cluster:kubevirt_virt_api_up:sum"))
		})

	It("should filter both when both allowlists are set", func() {
		err := lib.RedeployController(kvNamespace, image,
			"--alerts-allowlist=VirtAPIDown",
			"--recording-rules-allowlist=cluster:kubevirt_virt_api_up:sum")
		Expect(err).ToNot(HaveOccurred())

		Eventually(func() error {
			_, err := rules.GetPrometheusRule(kvNamespace, prometheusRuleName)
			return err
		}, 2*time.Minute, 5*time.Second).Should(Succeed())

		pr, err := rules.GetPrometheusRule(kvNamespace, prometheusRuleName)
		Expect(err).ToNot(HaveOccurred())
		Expect(rules.AlertNames(pr)).To(ConsistOf("VirtAPIDown"))
		Expect(rules.RecordingRuleNames(pr)).To(
			ConsistOf("cluster:kubevirt_virt_api_up:sum"))
	})

	It("should not create a PrometheusRule when both allowlists are none", func() {
		err := lib.RedeployController(kvNamespace, image,
			"--alerts-allowlist=none",
			"--recording-rules-allowlist=none")
		Expect(err).ToNot(HaveOccurred())

		time.Sleep(30 * time.Second)

		_, err = lib.Kubectl("get", "prometheusrule",
			prometheusRuleName, "-n", kvNamespace)
		Expect(err).To(HaveOccurred())
	})
})
