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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Block latency collector", func() {
	const (
		readOperation  = "read"
		writeOperation = "write"
		flushOperation = "flush"
		namespace      = "ns1"
		name           = "vm1"
		drive          = "vda"
		alias          = "ua-rootdisk"
		nodeName       = "node1"
	)

	newHistogram := func() *Histogram {
		return &Histogram{
			Count: 6,
			Buckets: []HistogramBucket{
				{
					UpperBound:      1_000_000,
					CumulativeCount: 2,
				},
				{
					UpperBound:      10_000_000,
					CumulativeCount: 5,
				},
			},
		}
	}

	It("should convert histogram bucket boundaries to seconds", func() {
		buckets := convertLatencyHistogram(newHistogram())

		Expect(buckets).To(Equal(map[float64]uint64{
			0.001: 2,
			0.01:  5,
		}))
	})

	DescribeTable("should collect block latency histograms", func(blockAlias string, allOperations bool) {
		cache := NewStatsCache()

		vmi := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Status: k6tv1.VirtualMachineInstanceStatus{
				NodeName: nodeName,
			},
		}

		stats := &VMStats{
			DomainStats: DomainStats{
				Block: []DomainStatsBlock{
					{
						NameSet: true,
						Name:    drive,
						Alias:   blockAlias,

						RdTimesSet: true,
						RdTimes:    1_000_000_000,

						LatencyHistograms: DomainStatsBlockLatencyHistograms{
							Read: newHistogram(),
						},
					},
				},
			},
		}

		if allOperations {
			block := &stats.DomainStats.Block[0]
			block.WrTimesSet = true
			block.WrTimes = 2_000_000_000
			block.FlTimesSet = true
			block.FlTimes = 3_000_000_000
			block.LatencyHistograms.Write = newHistogram()
			block.LatencyHistograms.Flush = newHistogram()
		}

		cache.Store(namespace+"/"+name, vmi, stats)

		registry := prometheus.NewRegistry()
		Expect(
			registry.Register(newBlockLatencyCollector(cache)),
		).To(Succeed())

		families, err := registry.Gather()
		Expect(err).ToNot(HaveOccurred())
		Expect(families).To(HaveLen(1))

		family := families[0]
		Expect(family.GetName()).To(Equal(blockIOLatencyMetricName))
		expectedMetrics := 1
		if allOperations {
			expectedMetrics = 3
		}

		Expect(family.GetMetric()).To(HaveLen(expectedMetrics))

		expectedDrive := blockAlias
		if expectedDrive == "" {
			expectedDrive = drive
		}
		foundOperations := map[string]bool{}

		for _, metric := range family.GetMetric() {
			labels := map[string]string{}

			for _, label := range metric.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}

			Expect(labels).To(HaveKeyWithValue("node", nodeName))
			Expect(labels).To(HaveKeyWithValue("namespace", namespace))
			Expect(labels).To(HaveKeyWithValue("name", name))
			Expect(labels).To(HaveKeyWithValue("drive", expectedDrive))

			operation := labels["operation"]
			foundOperations[operation] = true

			histogram := metric.GetHistogram()
			Expect(histogram).ToNot(BeNil())
			Expect(histogram.GetSampleCount()).To(Equal(uint64(6)))
			Expect(histogram.GetBucket()).To(HaveLen(2))

			Expect(histogram.GetBucket()[0].GetUpperBound()).To(Equal(0.001))
			Expect(histogram.GetBucket()[0].GetCumulativeCount()).To(Equal(uint64(2)))

			Expect(histogram.GetBucket()[1].GetUpperBound()).To(Equal(0.01))
			Expect(histogram.GetBucket()[1].GetCumulativeCount()).To(Equal(uint64(5)))

			switch operation {
			case readOperation:
				Expect(histogram.GetSampleSum()).To(Equal(1.0))
			case writeOperation:
				Expect(histogram.GetSampleSum()).To(Equal(2.0))
			case flushOperation:
				Expect(histogram.GetSampleSum()).To(Equal(3.0))
			default:
				Fail("unexpected operation: " + operation)
			}
		}

		Expect(foundOperations).To(HaveKey(readOperation))
		if allOperations {
			Expect(foundOperations).To(HaveKey(writeOperation))
			Expect(foundOperations).To(HaveKey(flushOperation))
		}
	}, Entry("read, write and flush using the alias", alias, true),
		Entry("read using the block name when alias is absent", "", false),
	)

	DescribeTable("should skip histograms with missing data", func(histogram *Histogram, totalTimeSet bool) {
		cache := NewStatsCache()

		vmi := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}

		stats := &VMStats{
			DomainStats: DomainStats{
				Block: []DomainStatsBlock{
					{
						NameSet:    true,
						Name:       drive,
						RdTimesSet: totalTimeSet,
						RdTimes:    1_000_000_000,
						LatencyHistograms: DomainStatsBlockLatencyHistograms{
							Read: histogram,
						},
					},
				},
			},
		}

		cache.Store(namespace+"/"+name, vmi, stats)

		registry := prometheus.NewRegistry()
		Expect(
			registry.Register(newBlockLatencyCollector(cache)),
		).To(Succeed())

		families, err := registry.Gather()
		Expect(err).ToNot(HaveOccurred())
		Expect(families).To(BeEmpty())
	},
		Entry("histogram without a matching total time", newHistogram(), false),
		Entry("total time without a histogram", (*Histogram)(nil), true),
	)
})
