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
