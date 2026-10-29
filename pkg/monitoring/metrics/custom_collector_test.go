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

package metrics

import (
	"errors"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Custom collectors", func() {
	const testMetricName = "test_custom_metric"
	var registry *prometheus.Registry

	BeforeEach(func() {
		registry = prometheus.NewRegistry()

		originalRegister := operatormetrics.Register
		originalUnregister := operatormetrics.Unregister

		operatormetrics.Register = registry.Register
		operatormetrics.Unregister = registry.Unregister

		DeferCleanup(func() {
			operatormetrics.Register = originalRegister
			operatormetrics.Unregister = originalUnregister
		})
	})

	newCollector := func() prometheus.Collector {
		return prometheus.NewGauge(prometheus.GaugeOpts{
			Name: testMetricName,
			Help: "A custom metric used for testing.",
		})
	}

	It("should register custom collectors when allowlist is nil", func() {
		err := RegisterCustomCollectors(
			nil,
			CustomCollector{
				MetricName: testMetricName,
				Collector:  newCollector(),
			},
		)
		Expect(err).ToNot(HaveOccurred())

		families, err := registry.Gather()
		Expect(err).ToNot(HaveOccurred())
		Expect(families).To(HaveLen(1))
		Expect(families[0].GetName()).To(Equal(testMetricName))
	})

	It("should register an allowed custom collector", func() {
		err := RegisterCustomCollectors(
			map[string]bool{
				testMetricName: true,
			},
			CustomCollector{
				MetricName: testMetricName,
				Collector:  newCollector(),
			},
		)
		Expect(err).ToNot(HaveOccurred())

		families, err := registry.Gather()
		Expect(err).ToNot(HaveOccurred())
		Expect(families).To(HaveLen(1))
	})

	It("should skip a custom collector not included in the allowlist", func() {
		err := RegisterCustomCollectors(
			map[string]bool{},
			CustomCollector{
				MetricName: testMetricName,
				Collector:  newCollector(),
			},
		)
		Expect(err).ToNot(HaveOccurred())

		families, err := registry.Gather()
		Expect(err).ToNot(HaveOccurred())
		Expect(families).To(BeEmpty())
	})

	It("should reject a custom collector without a metric name", func() {
		err := RegisterCustomCollectors(
			nil,
			CustomCollector{
				Collector: newCollector(),
			},
		)

		Expect(err).To(MatchError("custom collector metric name cannot be empty"))
	})

	It("should reject a nil custom collector", func() {
		err := RegisterCustomCollectors(
			nil,
			CustomCollector{
				MetricName: testMetricName,
			},
		)

		Expect(err).To(MatchError(
			`custom collector for metric "test_custom_metric" is nil`,
		))
	})

	It("should propagate collector registration errors", func() {
		expectedErr := errors.New("registration failed")

		operatormetrics.Register = func(prometheus.Collector) error {
			return expectedErr
		}

		err := RegisterCustomCollectors(
			nil,
			CustomCollector{
				MetricName: testMetricName,
				Collector:  newCollector(),
			},
		)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring(
			`registering custom collector for metric "test_custom_metric"`,
		))
		Expect(errors.Is(err, expectedErr)).To(BeTrue())
	})

	It("should allow re-registering a custom collector", func() {
		Expect(RegisterCustomCollectors(
			nil,
			CustomCollector{
				MetricName: testMetricName,
				Collector:  newCollector(),
			},
		)).To(Succeed())

		Expect(RegisterCustomCollectors(
			nil,
			CustomCollector{
				MetricName: testMetricName,
				Collector:  newCollector(),
			},
		)).To(Succeed())

		families, err := registry.Gather()
		Expect(err).ToNot(HaveOccurred())
		Expect(families).To(HaveLen(1))
	})

	It("should unregister a previously registered collector when disallowed", func() {
		collector := CustomCollector{
			MetricName: testMetricName,
			Collector:  newCollector(),
		}

		Expect(RegisterCustomCollectors(nil, collector)).To(Succeed())

		Expect(RegisterCustomCollectors(
			map[string]bool{},
			collector,
		)).To(Succeed())

		families, err := registry.Gather()
		Expect(err).ToNot(HaveOccurred())
		Expect(families).To(BeEmpty())
	})
})
