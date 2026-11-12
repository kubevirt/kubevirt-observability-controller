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

package groups

import (
	"fmt"
	"maps"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
)

type opted interface {
	GetOpts() operatormetrics.MetricOpts
}

// Index maps each item name to its group. Every item must carry a registered
// group, so an untagged definition fails here rather than silently becoming
// unreachable: with no group, no flag value could ever select it.
func Index[T opted](items []T) (map[string]Name, error) {
	out := make(map[string]Name, len(items))
	for _, it := range items {
		opts := it.GetOpts()
		raw, ok := opts.ExtraFields[Key]
		if !ok || raw == "" {
			return nil, fmt.Errorf("%q has no %q in ExtraFields", opts.Name, Key)
		}
		if _, known := Lookup(Name(raw)); !known {
			return nil, fmt.Errorf("%q names unknown group %q", opts.Name, raw)
		}
		out[opts.Name] = Name(raw)
	}
	return out, nil
}

// IndexCollectors maps the metric names of every collector to their groups.
func IndexCollectors(collectors []operatormetrics.Collector) (map[string]Name, error) {
	out := make(map[string]Name)
	for _, c := range collectors {
		got, err := Index(c.Metrics)
		if err != nil {
			return nil, err
		}
		maps.Copy(out, got)
	}
	return out, nil
}

// IndexAlerts maps alert names to their groups. Alerts carry membership in the
// map key rather than in ExtraFields, which promv1.Rule does not have.
func IndexAlerts(alertGroups map[Name][]promv1.Rule) (map[string]Name, error) {
	out := make(map[string]Name)
	for g, rules := range alertGroups {
		if _, known := Lookup(g); !known {
			return nil, fmt.Errorf("alerts reference unknown group %q", g)
		}
		for _, r := range rules {
			out[r.Alert] = g
		}
	}
	return out, nil
}

// Enabled reports which groups contribute at least one selected item. Call it
// per artifact type and merge, since a group can be enabled through its alerts
// alone while its requirements are metrics.
func Enabled(selected map[string]bool, index map[string]Name) map[Name]bool {
	out := make(map[Name]bool)
	for name, g := range index {
		if selected[name] {
			out[g] = true
		}
	}
	return out
}
