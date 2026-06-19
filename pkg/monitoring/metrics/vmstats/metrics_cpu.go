package vmstats

import "github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"

var (
	cpuMetricsList = []operatormetrics.Metric{
		cpuUsageSeconds, cpuUserUsageSeconds, cpuSystemUsageSeconds,
		guestLoad1m, guestLoad5m, guestLoad15m,
	}

	cpuUsageSeconds = operatormetrics.NewCounter(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_cpu_usage_seconds_total",
		Help: "Total CPU time spent in all modes (sum of both vcpu and hypervisor usage).",
	})
	cpuUserUsageSeconds = operatormetrics.NewCounter(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_cpu_user_usage_seconds_total",
		Help: "Total CPU time spent in user mode.",
	})
	cpuSystemUsageSeconds = operatormetrics.NewCounter(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_cpu_system_usage_seconds_total",
		Help: "Total CPU time spent in system mode.",
	})
	guestLoad1m = operatormetrics.NewGauge(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_guest_load_1m",
		Help: "Guest system load average over 1 minute.",
	})
	guestLoad5m = operatormetrics.NewGauge(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_guest_load_5m",
		Help: "Guest system load average over 5 minutes.",
	})
	guestLoad15m = operatormetrics.NewGauge(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_guest_load_15m",
		Help: "Guest system load average over 15 minutes.",
	})
)

func collectCPUMetrics(report *VMIReport) []operatormetrics.CollectorResult {
	var crs []operatormetrics.CollectorResult

	if report.Stats.DomainStats.Cpu != nil {
		cpu := report.Stats.DomainStats.Cpu
		if cpu.TimeSet {
			crs = append(crs, report.newCollectorResult(cpuUsageSeconds, nanosecondsToSeconds(cpu.Time)))
		}
		if cpu.UserSet {
			crs = append(crs, report.newCollectorResult(cpuUserUsageSeconds, nanosecondsToSeconds(cpu.User)))
		}
		if cpu.SystemSet {
			crs = append(crs, report.newCollectorResult(cpuSystemUsageSeconds, nanosecondsToSeconds(cpu.System)))
		}
	}

	if report.Stats.DomainStats.Load != nil {
		load := report.Stats.DomainStats.Load
		if load.Load1mSet {
			crs = append(crs, report.newCollectorResult(guestLoad1m, load.Load1m))
		}
		if load.Load5mSet {
			crs = append(crs, report.newCollectorResult(guestLoad5m, load.Load5m))
		}
		if load.Load15mSet {
			crs = append(crs, report.newCollectorResult(guestLoad15m, load.Load15m))
		}
	}

	return crs
}
