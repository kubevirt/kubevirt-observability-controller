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

package rules

import (
	"encoding/json"
	"fmt"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	"github.com/kubevirt/kubevirt-observability-controller/test/lib"
)

func GetPrometheusRule(namespace, name string) (*promv1.PrometheusRule, error) {
	out, err := lib.Kubectl("get", "prometheusrule", name, "-n", namespace, "-o", "json")
	if err != nil {
		return nil, fmt.Errorf("getting PrometheusRule %s/%s: %w", namespace, name, err)
	}
	pr := &promv1.PrometheusRule{}
	if err := json.Unmarshal([]byte(out), pr); err != nil {
		return nil, fmt.Errorf("unmarshaling PrometheusRule: %w", err)
	}
	return pr, nil
}

func AlertNames(pr *promv1.PrometheusRule) []string {
	var names []string
	for _, g := range pr.Spec.Groups {
		for _, r := range g.Rules {
			if r.Alert != "" {
				names = append(names, r.Alert)
			}
		}
	}
	return names
}

func RecordingRuleNames(pr *promv1.PrometheusRule) []string {
	var names []string
	for _, g := range pr.Spec.Groups {
		for _, r := range g.Rules {
			if r.Record != "" {
				names = append(names, r.Record)
			}
		}
	}
	return names
}

func AlertCount(pr *promv1.PrometheusRule) int {
	return len(AlertNames(pr))
}

func RecordingRuleCount(pr *promv1.PrometheusRule) int {
	return len(RecordingRuleNames(pr))
}
