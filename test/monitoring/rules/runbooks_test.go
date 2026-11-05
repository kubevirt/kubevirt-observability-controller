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
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kubevirt/kubevirt-observability-controller/test/monitoring/rules"
)

var _ = Describe("Alert Runbooks", func() {
	It("should have a published runbook for every alert", func(ctx SpecContext) {
		prometheusRule, err := rules.GetPrometheusRule(kvNamespace, prometheusRuleName)
		Expect(err).ToNot(HaveOccurred())
		Expect(rules.AlertCount(prometheusRule)).To(BeNumerically(">", 0))

		const requestTimeout = 30 * time.Second
		client := &http.Client{Timeout: requestTimeout}
		for _, group := range prometheusRule.Spec.Groups {
			for _, alert := range group.Rules {
				if alert.Alert == "" {
					continue
				}
				Expect(alert.Annotations).To(HaveKeyWithValue("runbook_url", Not(BeEmpty())),
					"alert %s is missing a runbook_url annotation", alert.Alert)
				runbookURL := alert.Annotations["runbook_url"]
				request, err := http.NewRequestWithContext(ctx, http.MethodHead, runbookURL, http.NoBody)
				Expect(err).ToNot(HaveOccurred(), "creating runbook request for alert %s: %s", alert.Alert, runbookURL)
				response, err := client.Do(request)
				Expect(err).ToNot(HaveOccurred(), "checking runbook for alert %s: %s", alert.Alert, runbookURL)
				Expect(response.Body.Close()).To(Succeed(), "closing runbook response for alert %s", alert.Alert)
				Expect(response.StatusCode).To(Equal(http.StatusOK),
					"runbook for alert %s at %s must return HTTP 200; "+
						"merge and publish the runbook in kubevirt/monitoring before adding the alert",
					alert.Alert, runbookURL)
			}
		}
	})
})
