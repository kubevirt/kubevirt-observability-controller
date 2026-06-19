package vmstats

import (
	"fmt"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
)

var (
	vcpuMetricsList = []operatormetrics.Metric{
		vcpuSeconds, vcpuWaitSeconds, vcpuDelaySeconds,
	}

	vcpuSeconds = operatormetrics.NewCounter(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_vcpu_seconds_total",
		Help: "Total amount of time spent in each state by each vcpu.",
	})
	vcpuWaitSeconds = operatormetrics.NewCounter(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_vcpu_wait_seconds_total",
		Help: "Amount of time spent by each vcpu while waiting on I/O.",
	})
	vcpuDelaySeconds = operatormetrics.NewCounter(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_vcpu_delay_seconds_total",
		Help: "Amount of time spent by each vcpu waiting in the queue instead of running.",
	})
)

func collectVCPUMetrics(report *VMIReport) []operatormetrics.CollectorResult {
	var crs []operatormetrics.CollectorResult

	if report.Stats.DomainStats.Vcpu == nil {
		return crs
	}

	for vcpuIdx, vcpu := range report.Stats.DomainStats.Vcpu {
		id := fmt.Sprintf("%d", vcpuIdx)

		if vcpu.TimeSet {
			labels := map[string]string{"id": id, "state": humanReadableVCPUState(vcpu.State)}
			crs = append(crs, report.newCollectorResultWithLabels(
				vcpuSeconds, nanosecondsToSeconds(vcpu.Time), labels,
			))
		}
		if vcpu.WaitSet {
			crs = append(crs, report.newCollectorResultWithLabels(
				vcpuWaitSeconds, nanosecondsToSeconds(vcpu.Wait), map[string]string{"id": id},
			))
		}
		if vcpu.DelaySet {
			crs = append(crs, report.newCollectorResultWithLabels(
				vcpuDelaySeconds, nanosecondsToSeconds(vcpu.Delay), map[string]string{"id": id},
			))
		}
	}

	return crs
}
