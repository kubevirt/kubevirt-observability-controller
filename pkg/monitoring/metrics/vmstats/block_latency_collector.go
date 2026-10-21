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
	"github.com/prometheus/client_golang/prometheus"
	ctrl "sigs.k8s.io/controller-runtime"
)

type latencyOperation struct {
	histogram    *Histogram
	totalTimeSet bool
	totalTime    uint64
}

const (
	blockIOLatencyMetricName = "kubevirt_vmi_storage_io_latency_seconds"
	blockIOLatencyMetricHelp = "I/O latency distribution for block devices."
)

var (
	blockLatencyLog = ctrl.Log.WithName("vmstats-block-latency")

	blockIOLatencyDesc = prometheus.NewDesc(
		blockIOLatencyMetricName,
		blockIOLatencyMetricHelp,
		[]string{"node", "namespace", "name", "drive", "operation"},
		nil,
	)
)

type blockLatencyCollector struct {
	cache *StatsCache
}

var _ prometheus.Collector = &blockLatencyCollector{}

func newBlockLatencyCollector(cache *StatsCache) *blockLatencyCollector {
	return &blockLatencyCollector{
		cache: cache,
	}
}

func (c *blockLatencyCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- blockIOLatencyDesc
}

func (c *blockLatencyCollector) Collect(ch chan<- prometheus.Metric) {
	if c.cache == nil {
		return
	}

	for _, entry := range c.cache.List() {
		if entry == nil || entry.VMI == nil || entry.Stats == nil {
			continue
		}

		report := NewVMIReport(entry.VMI, entry.Stats)
		collectBlockLatencyHistograms(report, ch)
	}
}

func collectBlockLatencyHistograms(
	report *VMIReport,
	ch chan<- prometheus.Metric,
) {
	if report == nil || report.VMI == nil || report.Stats == nil {
		return
	}

	for _, block := range report.Stats.DomainStats.Block {
		if !block.NameSet {
			continue
		}

		drive := block.Name
		if block.Alias != "" {
			drive = block.Alias
		}

		operations := getLatencyOperations(&block)

		for _, operation := range operations {
			if operation.histogram == nil ||
				operation.histogram.Name == "" ||
				!operation.totalTimeSet {
				continue
			}

			buckets := convertLatencyHistogram(operation.histogram)

			metric, err := prometheus.NewConstHistogram(
				blockIOLatencyDesc,
				operation.histogram.Count,
				nanosecondsToSeconds(operation.totalTime),
				buckets,
				report.VMI.Status.NodeName,
				report.VMI.Namespace,
				report.VMI.Name,
				drive,
				operation.histogram.Name,
			)

			if err != nil {
				blockLatencyLog.Error(
					err,
					"failed to create block I/O latency histogram",
					"namespace", report.VMI.Namespace,
					"name", report.VMI.Name,
					"drive", drive,
					"operation", operation.histogram.Name,
				)
				continue
			}

			ch <- metric
		}
	}
}

func getLatencyOperations(block *DomainStatsBlock) []latencyOperation {
	return []latencyOperation{
		{
			histogram:    block.LatencyHistograms.Read,
			totalTimeSet: block.RdTimesSet,
			totalTime:    block.RdTimes,
		},
		{
			histogram:    block.LatencyHistograms.Write,
			totalTimeSet: block.WrTimesSet,
			totalTime:    block.WrTimes,
		},
		{
			histogram:    block.LatencyHistograms.Flush,
			totalTimeSet: block.FlTimesSet,
			totalTime:    block.FlTimes,
		},
	}
}

func convertLatencyHistogram(
	histogram *Histogram,
) map[float64]uint64 {
	buckets := make(map[float64]uint64, len(histogram.Buckets))

	for _, bucket := range histogram.Buckets {
		buckets[nanosecondsToSeconds(bucket.UpperBound)] = bucket.CumulativeCount
	}

	return buckets
}
