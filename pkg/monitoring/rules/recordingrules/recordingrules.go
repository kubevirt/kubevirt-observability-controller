package recordingrules

import "github.com/rhobs/operator-observability-toolkit/pkg/operatorrules"

func Register(registry *operatorrules.Registry, namespace string, allowlist map[string]bool) error {
	allRules := [][]operatorrules.RecordingRule{
		apiRecordingRules,
		nodesRecordingRules,
		operatorRecordingRules,
		virtRecordingRules(namespace),
		vmRecordingRules,
		vmiRecordingRules,
		vmsnapshotRecordingRules,
		deprecatedRecordingRules,
	}

	if allowlist != nil {
		for i := range allRules {
			allRules[i] = filterRecordingRules(allRules[i], allowlist)
		}
	}

	return registry.RegisterRecordingRules(allRules...)
}

func filterRecordingRules(
	rules []operatorrules.RecordingRule, allowlist map[string]bool,
) []operatorrules.RecordingRule {
	var filtered []operatorrules.RecordingRule

	for _, r := range rules {
		if allowlist[r.MetricsOpts.Name] {
			filtered = append(filtered, r)
		}
	}

	return filtered
}
