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

package alerts

import (
	"errors"
	"fmt"
	"maps"
	"os"
	"strings"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatorrules"

	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/groups"
)

const (
	prometheusRunbookAnnotationKey = "runbook_url"
	defaultRunbookURLTemplate      = "https://kubevirt.io/monitoring/runbooks/%s"
	runbookURLTemplateEnv          = "RUNBOOK_URL_TEMPLATE"

	severityAlertLabelKey        = "severity"
	operatorHealthImpactLabelKey = "operator_health_impact"
	summaryAnnotationKey         = "summary"
	descriptionAnnotationKey     = "description"
	partOfAlertLabelKey          = "kubernetes_operator_part_of"
	componentAlertLabelKey       = "kubernetes_operator_component"
	namespaceAlertLabelKey       = "namespace"
	kubevirtLabelValue           = "kubevirt"

	eightyPercent = 80
	fiveMinutes   = 5
)

// All returns every alert the controller defines, keyed by its group, with the
// common labels and annotations applied.
//
// Each call returns freshly copied rules. Decorating the package-level slices
// in place would leak one caller's namespace into the next, which matters now
// that the selection layer builds its index from a separate call.
func All(namespace string) map[groups.Name][]promv1.Rule {
	// Component and system alerts operate in the KubeVirt install namespace.
	// VM workload alerts derive namespace from their PromQL expression instead.
	component := map[groups.Name][]promv1.Rule{
		groups.ClusterNodes:   systemAlerts(namespace),
		groups.VirtAPI:        virtAPIAlerts(namespace),
		groups.VirtController: virtControllerAlerts(namespace),
		groups.VirtHandler:    virtHandlerAlerts(namespace),
		groups.VirtOperator:   virtOperatorAlerts(namespace),
	}
	workload := map[groups.Name][]promv1.Rule{
		groups.VMIStatus:       vmiStatusAlerts,
		groups.VMStatus:        vmStatusAlerts,
		groups.VMIMigration:    vmiMigrationAlerts,
		groups.VMICPU:          vmiCPUAlerts,
		groups.VMIMemory:       vmiMemoryAlerts,
		groups.GuestFilesystem: guestFilesystemAlerts,
		groups.VMIHotplug:      vmiHotplugAlerts,
		groups.GuestDrivers:    guestDriverAlerts,
		groups.VMIInterfaces:   vmiInterfaceAlerts,
	}

	runbookURLTemplate := getRunbookURLTemplate()
	out := make(map[groups.Name][]promv1.Rule, len(component)+len(workload))

	decorate := func(src map[groups.Name][]promv1.Rule, inNamespace bool) {
		for g, rules := range src {
			cp := make([]promv1.Rule, 0, len(rules))
			for _, r := range rules {
				r.Labels = maps.Clone(r.Labels)
				r.Annotations = maps.Clone(r.Annotations)
				r.Labels[partOfAlertLabelKey] = kubevirtLabelValue
				r.Labels[componentAlertLabelKey] = kubevirtLabelValue
				r.Annotations[prometheusRunbookAnnotationKey] = fmt.Sprintf(runbookURLTemplate, r.Alert)
				if inNamespace {
					r.Labels[namespaceAlertLabelKey] = namespace
				}
				cp = append(cp, r)
			}
			out[g] = cp
		}
	}

	decorate(component, true)
	decorate(workload, false)

	return out
}

func Register(registry *operatorrules.Registry, namespace string, allowlist map[string]bool) error {
	all := All(namespace)

	// Iterate groups.All rather than the map so registration order is
	// deterministic. The registry happens to sort before rendering, so this
	// does not affect the PrometheusRule, but it keeps failures reproducible.
	allAlerts := make([][]promv1.Rule, 0, len(all))
	for _, g := range groups.All {
		rules, ok := all[g.Name]
		if !ok {
			continue
		}
		if allowlist != nil {
			rules = filterAlerts(rules, allowlist)
		}
		allAlerts = append(allAlerts, rules)
	}

	return registry.RegisterAlerts(allAlerts...)
}

func filterAlerts(rules []promv1.Rule, allowlist map[string]bool) []promv1.Rule {
	var filtered []promv1.Rule
	for _, r := range rules {
		if allowlist[r.Alert] {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func getRunbookURLTemplate() string {
	runbookURLTemplate, exists := os.LookupEnv(runbookURLTemplateEnv)
	if !exists {
		runbookURLTemplate = defaultRunbookURLTemplate
	}

	if strings.Count(runbookURLTemplate, "%s") != 1 {
		panic(errors.New("runbook URL template must have exactly 1 %s substring"))
	}

	return runbookURLTemplate
}

func getErrorRatio(ns, podName string) string {
	errorCodeRegex := "(4|5)[0-9][0-9]"
	errorRatioQuery := "sum ( rate ( kubevirt_rest_client_requests_total" +
		"{namespace=\"%s\",pod=~\"%s-.*\",code=~\"%s\"} [%dm] ) )  / " +
		" sum ( rate ( kubevirt_rest_client_requests_total" +
		"{namespace=\"%s\",pod=~\"%s-.*\"} [%dm] ) )"
	return fmt.Sprintf(
		errorRatioQuery,
		ns, podName, errorCodeRegex, fiveMinutes,
		ns, podName, fiveMinutes,
	)
}

func getRestCallsFailedWarning(component string) string {
	duration := fmt.Sprintf("%d minutes", fiveMinutes)
	failingCallsPercentage := eightyPercent

	const restCallsFailWarningTemplate = "More than %d%% of the rest calls failed in %s for the last %s"
	return fmt.Sprintf(restCallsFailWarningTemplate, failingCallsPercentage, component, duration)
}
