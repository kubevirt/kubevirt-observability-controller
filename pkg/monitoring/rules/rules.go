package rules

import (
	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatorrules"

	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/rules/alerts"
	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/rules/recordingrules"
)

var registry = operatorrules.NewRegistry()

func ResetRegistry() {
	registry = operatorrules.NewRegistry()
}

func SetupRules(namespace string, alertsAllowlist, recordingRulesAllowlist map[string]bool) error {
	if err := recordingrules.Register(registry, namespace, recordingRulesAllowlist); err != nil {
		return err
	}

	return alerts.Register(registry, namespace, alertsAllowlist)
}

func HasRegisteredRules() bool {
	return len(registry.ListAlerts()) > 0 || len(registry.ListRecordingRules()) > 0
}

func BuildPrometheusRule(name, namespace string) (*promv1.PrometheusRule, error) {
	return registry.BuildPrometheusRule(
		name,
		namespace,
		map[string]string{
			"app.kubernetes.io/managed-by": "kubevirt-observability-controller",
		},
	)
}
