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
