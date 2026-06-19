package vmstats

import "github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"

var (
	dirtyRateMetricsList = []operatormetrics.Metric{
		dirtyRateBytesPerSecond,
	}

	dirtyRateBytesPerSecond = operatormetrics.NewGauge(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_dirty_rate_bytes_per_second",
		Help: "Guest dirty-rate in bytes per second.",
	})
)

func collectDirtyRateMetrics(report *VMIReport) []operatormetrics.CollectorResult {
	if report.Stats.DomainStats.DirtyRate == nil || !report.Stats.DomainStats.DirtyRate.MegabytesPerSecondSet {
		return nil
	}

	dirtyRateInBytesPerSecond := report.Stats.DomainStats.DirtyRate.MegabytesPerSecond * 1024 * 1024
	return []operatormetrics.CollectorResult{
		report.newCollectorResult(dirtyRateBytesPerSecond, float64(dirtyRateInBytesPerSecond)),
	}
}
