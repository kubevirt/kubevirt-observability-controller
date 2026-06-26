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
