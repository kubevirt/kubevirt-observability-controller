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
