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

	newHistogram := func(operation string) *Histogram {
		return &Histogram{
			Name:  operation,
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
		buckets := convertLatencyHistogram(newHistogram(readOperation))

		Expect(buckets).To(Equal(map[float64]uint64{
			0.001: 2,
			0.01:  5,
		}))
	})

	It("should collect read, write and flush histograms", func() {
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
						Alias:   alias,

						RdTimesSet: true,
						RdTimes:    1_000_000_000,

						WrTimesSet: true,
						WrTimes:    2_000_000_000,

						FlTimesSet: true,
						FlTimes:    3_000_000_000,

						LatencyHistograms: DomainStatsBlockLatencyHistograms{
							Read:  newHistogram(readOperation),
							Write: newHistogram(writeOperation),
							Flush: newHistogram(flushOperation),
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
		Expect(families).To(HaveLen(1))

		family := families[0]
		Expect(family.GetName()).To(Equal(blockIOLatencyMetricName))
		Expect(family.GetMetric()).To(HaveLen(3))

		foundOperations := map[string]bool{}

		for _, metric := range family.GetMetric() {
			labels := map[string]string{}

			for _, label := range metric.GetLabel() {
				labels[label.GetName()] = label.GetValue()
			}

			Expect(labels).To(HaveKeyWithValue("node", nodeName))
			Expect(labels).To(HaveKeyWithValue("namespace", namespace))
			Expect(labels).To(HaveKeyWithValue("name", name))
			Expect(labels).To(HaveKeyWithValue("drive", alias))

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
		Expect(foundOperations).To(HaveKey(writeOperation))
		Expect(foundOperations).To(HaveKey(flushOperation))
	})

	It("should use the block name when alias is not present", func() {
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
						NameSet:    true,
						Name:       drive,
						RdTimesSet: true,
						RdTimes:    1_000_000_000,
						LatencyHistograms: DomainStatsBlockLatencyHistograms{
							Read: newHistogram(readOperation),
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
		Expect(families).To(HaveLen(1))
		Expect(families[0].GetMetric()).To(HaveLen(1))

		labels := map[string]string{}
		for _, label := range families[0].GetMetric()[0].GetLabel() {
			labels[label.GetName()] = label.GetValue()
		}

		Expect(labels).To(HaveKeyWithValue("drive", drive))
	})

	It("should skip histograms without a matching total time", func() {
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
						NameSet: true,
						Name:    drive,
						LatencyHistograms: DomainStatsBlockLatencyHistograms{
							Read: newHistogram(readOperation),
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
	})

	It("should skip histograms without an operation name", func() {
		cache := NewStatsCache()

		vmi := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
		}

		histogram := newHistogram("")
		stats := &VMStats{
			DomainStats: DomainStats{
				Block: []DomainStatsBlock{
					{
						NameSet:    true,
						Name:       drive,
						RdTimesSet: true,
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
		Expect(registry.Register(newBlockLatencyCollector(cache))).To(Succeed())

		families, err := registry.Gather()
		Expect(err).ToNot(HaveOccurred())
		Expect(families).To(BeEmpty())
	})
})
