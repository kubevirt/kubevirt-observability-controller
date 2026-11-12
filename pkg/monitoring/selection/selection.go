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

// Package selection resolves the allowlist flags against the controller's
// definitions.
//
// It sits above groups, which holds the registry and the grammar but cannot
// reach the definitions: every package that defines a metric, recording rule,
// or alert imports groups to tag it, so groups importing them back would be a
// cycle.
package selection

import (
	"fmt"
	"maps"
	"strings"

	"github.com/go-logr/logr"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"

	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/groups"
	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/metrics"
	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/metrics/devicedrivers"
	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/metrics/vmstats"
	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/rules/alerts"
	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/rules/recordingrules"
)

// Flag names, used to attribute resolution errors to the flag that caused them.
const (
	MetricsFlag        = "--metrics-allowlist"
	RecordingRulesFlag = "--recording-rules-allowlist"
	AlertsFlag         = "--alerts-allowlist"
)

// Indexes maps every item the controller can expose to its group, one map per
// artifact type.
type Indexes struct {
	Metrics        map[string]groups.Name
	RecordingRules map[string]groups.Name
	Alerts         map[string]groups.Name
}

// BuildIndexes reads group membership off every definition. Indexing fails if
// an item carries no registered group, which no flag value could then select.
func BuildIndexes(namespace string) (Indexes, error) {
	metricIndex, err := groups.IndexCollectors(metrics.AllCollectors())
	if err != nil {
		return Indexes{}, fmt.Errorf("indexing collector metrics: %w", err)
	}

	// The vmstats and guest device driver collectors register themselves, but
	// their metrics still have to be selectable, and the vmstats request
	// policy is derived from the result.
	for _, extra := range [][]operatormetrics.Metric{vmstats.AllMetrics(), devicedrivers.AllMetrics()} {
		got, indexErr := groups.Index(extra)
		if indexErr != nil {
			return Indexes{}, fmt.Errorf("indexing metrics: %w", indexErr)
		}
		maps.Copy(metricIndex, got)
	}

	recordingRuleIndex, err := groups.Index(recordingrules.All(namespace))
	if err != nil {
		return Indexes{}, fmt.Errorf("indexing recording rules: %w", err)
	}

	alertIndex, err := groups.IndexAlerts(alerts.All(namespace))
	if err != nil {
		return Indexes{}, fmt.Errorf("indexing alerts: %w", err)
	}

	return Indexes{
		Metrics:        metricIndex,
		RecordingRules: recordingRuleIndex,
		Alerts:         alertIndex,
	}, nil
}

// Selection is what the allowlist flags resolved to: the item names to expose,
// plus the context needed to report what was chosen.
type Selection struct {
	Metrics        map[string]bool
	RecordingRules map[string]bool
	Alerts         map[string]bool

	EnabledGroups map[groups.Name]bool

	// Empty records flag entries that resolved to nothing, keyed by flag.
	Empty map[string][]string
}

// ForFlags resolves the three allowlist values against the definitions.
func ForFlags(namespace, metricsRaw, recordingRulesRaw, alertsRaw string) (Selection, error) {
	index, err := BuildIndexes(namespace)
	if err != nil {
		return Selection{}, err
	}

	sel := Selection{Empty: map[string][]string{}}

	for _, f := range []struct {
		name   string
		raw    string
		index  map[string]groups.Name
		target *map[string]bool
	}{
		{MetricsFlag, metricsRaw, index.Metrics, &sel.Metrics},
		{RecordingRulesFlag, recordingRulesRaw, index.RecordingRules, &sel.RecordingRules},
		{AlertsFlag, alertsRaw, index.Alerts, &sel.Alerts},
	} {
		resolved, empty, resolveErr := groups.Resolve(f.index, f.raw)
		if resolveErr != nil {
			return Selection{}, fmt.Errorf("%s: %w", f.name, resolveErr)
		}
		*f.target = resolved
		if len(empty) > 0 {
			sel.Empty[f.name] = empty
		}
	}

	sel.EnabledGroups = map[groups.Name]bool{}
	maps.Copy(sel.EnabledGroups, groups.Enabled(sel.Metrics, index.Metrics))
	maps.Copy(sel.EnabledGroups, groups.Enabled(sel.RecordingRules, index.RecordingRules))
	maps.Copy(sel.EnabledGroups, groups.Enabled(sel.Alerts, index.Alerts))

	return sel, nil
}

// Log reports what the flags resolved to. Worth saying out loud: with no flags
// set the controller exposes a deliberately narrow set, and the counts are the
// quickest way to see that is what happened.
func (s Selection) Log(log logr.Logger) {
	enabled := make([]string, 0, len(s.EnabledGroups))
	for _, g := range groups.All {
		if s.EnabledGroups[g.Name] {
			enabled = append(enabled, string(g.Name))
		}
	}

	log.Info("groups enabled", "groups", strings.Join(enabled, ", "))
	log.Info("selection resolved",
		"metrics", len(s.Metrics),
		"recordingRules", len(s.RecordingRules),
		"alerts", len(s.Alerts))

	for flagName, entries := range s.Empty {
		log.Info("allowlist entry selected nothing of this kind",
			"flag", flagName, "entries", strings.Join(entries, ", "))
	}
}
