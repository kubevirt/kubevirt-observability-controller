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

package vmstats

import (
	"slices"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

func RegisterCollector(cache *StatsCache, allowlist map[string]bool) error {
	operatormetrics.Register = ctrlmetrics.Registry.Register
	operatormetrics.Unregister = ctrlmetrics.Registry.Unregister

	collector := vmStatsCollector(cache)

	if allowlist == nil {
		return operatormetrics.RegisterCollector(collector)
	}

	filtered := filterVMStatsCollector(collector, allowlist)
	if filtered == nil {
		return nil
	}
	return operatormetrics.RegisterCollector(*filtered)
}

func vmStatsCollector(cache *StatsCache) operatormetrics.Collector {
	allMetrics := slices.Concat(
		cpuMetricsList,
		memoryMetricsList,
		vcpuMetricsList,
		networkMetricsList,
		blockMetricsList,
		dirtyRateMetricsList,
		guestMetricsList,
	)

	return operatormetrics.Collector{
		Metrics: allMetrics,
		CollectCallback: func() []operatormetrics.CollectorResult {
			entries := cache.List()
			if len(entries) == 0 {
				return nil
			}

			var crs []operatormetrics.CollectorResult
			for _, entry := range entries {
				report := NewVMIReport(entry.VMI, entry.Stats)
				crs = append(crs, collectCPUMetrics(report)...)
				crs = append(crs, collectMemoryMetrics(report)...)
				crs = append(crs, collectVCPUMetrics(report)...)
				crs = append(crs, collectNetworkMetrics(report)...)
				crs = append(crs, collectBlockMetrics(report)...)
				crs = append(crs, collectDirtyRateMetrics(report)...)
				crs = append(crs, collectGuestMetrics(report)...)
			}
			return crs
		},
	}
}

func filterVMStatsCollector(c operatormetrics.Collector, allowlist map[string]bool) *operatormetrics.Collector {
	var kept []operatormetrics.Metric
	for _, m := range c.Metrics {
		if allowlist[m.GetOpts().Name] {
			kept = append(kept, m)
		}
	}
	if len(kept) == 0 {
		return nil
	}

	allowedSet := make(map[string]bool, len(kept))
	for _, m := range kept {
		allowedSet[m.GetOpts().Name] = true
	}

	originalCallback := c.CollectCallback
	return &operatormetrics.Collector{
		Metrics: kept,
		CollectCallback: func() []operatormetrics.CollectorResult {
			results := originalCallback()
			var filtered []operatormetrics.CollectorResult
			for _, r := range results {
				if allowedSet[r.Metric.GetOpts().Name] {
					filtered = append(filtered, r)
				}
			}
			return filtered
		},
	}
}
