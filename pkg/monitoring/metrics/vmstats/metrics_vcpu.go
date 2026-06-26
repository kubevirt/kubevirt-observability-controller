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
