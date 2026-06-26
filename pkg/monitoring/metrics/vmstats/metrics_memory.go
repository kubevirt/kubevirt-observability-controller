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
	memoryMetricsList = []operatormetrics.Metric{
		memoryResident, memoryAvailable, memoryUnused, memoryCached,
		memorySwapInTrafficBytes, memorySwapOutTrafficBytes,
		memoryPgmajfaultTotal, memoryPgminfaultTotal,
		memoryActualBalloon, memoryUsableBytes, memoryDomainBytes,
	}

	memoryResident = operatormetrics.NewGauge(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_memory_resident_bytes",
		Help: "Resident set size of the process running the domain.",
	})
	memoryAvailable = operatormetrics.NewGauge(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_memory_available_bytes",
		Help: "Amount of usable memory as seen by the domain.",
	})
	memoryUnused = operatormetrics.NewGauge(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_memory_unused_bytes",
		Help: "The amount of memory left completely unused by the system.",
	})
	memoryCached = operatormetrics.NewGauge(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_memory_cached_bytes",
		Help: "The amount of memory that is being used to cache I/O and is available to be reclaimed.",
	})
	memorySwapInTrafficBytes = operatormetrics.NewGauge(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_memory_swap_in_traffic_bytes",
		Help: "The total amount of data read from swap space of the guest in bytes.",
	})
	memorySwapOutTrafficBytes = operatormetrics.NewGauge(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_memory_swap_out_traffic_bytes",
		Help: "The total amount of memory written out to swap space of the guest in bytes.",
	})
	memoryPgmajfaultTotal = operatormetrics.NewCounter(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_memory_pgmajfault_total",
		Help: "The number of page faults when disk IO was required.",
	})
	memoryPgminfaultTotal = operatormetrics.NewCounter(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_memory_pgminfault_total",
		Help: "The number of other page faults, when disk IO was not required.",
	})
	memoryActualBalloon = operatormetrics.NewGauge(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_memory_actual_balloon_bytes",
		Help: "Current balloon size in bytes.",
	})
	memoryUsableBytes = operatormetrics.NewGauge(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_memory_usable_bytes",
		Help: "The amount of memory which can be reclaimed by balloon without pushing the guest system to swap.",
	})
	memoryDomainBytes = operatormetrics.NewGauge(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_memory_domain_bytes",
		Help: "The amount of memory in bytes allocated to the domain.",
	})
)

func collectMemoryMetrics(report *VMIReport) []operatormetrics.CollectorResult {
	var crs []operatormetrics.CollectorResult

	if report.Stats.DomainStats.Memory == nil {
		return crs
	}

	mem := report.Stats.DomainStats.Memory

	if mem.RSSSet {
		crs = append(crs, report.newCollectorResult(memoryResident, kibibytesToBytes(mem.RSS)))
	}
	if mem.AvailableSet {
		crs = append(crs, report.newCollectorResult(memoryAvailable, kibibytesToBytes(mem.Available)))
	}
	if mem.UnusedSet {
		crs = append(crs, report.newCollectorResult(memoryUnused, kibibytesToBytes(mem.Unused)))
	}
	if mem.CachedSet {
		crs = append(crs, report.newCollectorResult(memoryCached, kibibytesToBytes(mem.Cached)))
	}
	if mem.SwapInSet {
		crs = append(crs, report.newCollectorResult(memorySwapInTrafficBytes, kibibytesToBytes(mem.SwapIn)))
	}
	if mem.SwapOutSet {
		crs = append(crs, report.newCollectorResult(memorySwapOutTrafficBytes, kibibytesToBytes(mem.SwapOut)))
	}
	if mem.MajorFaultSet {
		crs = append(crs, report.newCollectorResult(memoryPgmajfaultTotal, float64(mem.MajorFault)))
	}
	if mem.MinorFaultSet {
		crs = append(crs, report.newCollectorResult(memoryPgminfaultTotal, float64(mem.MinorFault)))
	}
	if mem.ActualBalloonSet {
		crs = append(crs, report.newCollectorResult(memoryActualBalloon, kibibytesToBytes(mem.ActualBalloon)))
	}
	if mem.UsableSet {
		crs = append(crs, report.newCollectorResult(memoryUsableBytes, kibibytesToBytes(mem.Usable)))
	}
	if mem.TotalSet {
		crs = append(crs, report.newCollectorResult(memoryDomainBytes, kibibytesToBytes(mem.Total)))
	}

	return crs
}
