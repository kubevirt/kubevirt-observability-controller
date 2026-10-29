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
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
)

// CustomCollector associates a Prometheus collector with the metric family
// it exposes. The metric name is used to apply the metrics allowlist.
type CustomCollector struct {
	MetricName string
	Collector  prometheus.Collector
}

// RegisterCustomCollectors registers custom Prometheus collectors while
// respecting the metrics allowlist.
// A nil allowlist enables all collectors. An empty non-nil allowlist disables
// all collectors.
func RegisterCustomCollectors(
	allowlist map[string]bool,
	collectors ...CustomCollector,
) error {
	for _, collector := range collectors {
		if collector.MetricName == "" {
			return fmt.Errorf("custom collector metric name cannot be empty")
		}

		if collector.Collector == nil {
			return fmt.Errorf(
				"custom collector for metric %q is nil",
				collector.MetricName,
			)
		}

		if allowlist != nil && !allowlist[collector.MetricName] {
			// Best effort cleanup in case the collector was previously
			// registered and is no longer allowed.
			operatormetrics.Unregister(collector.Collector)
			continue
		}

		// Custom collectors are not tracked by the operator-observability
		// registry. Unregister first to make registration idempotent.
		operatormetrics.Unregister(collector.Collector)

		if err := operatormetrics.Register(collector.Collector); err != nil {
			return fmt.Errorf(
				"registering custom collector for metric %q: %w",
				collector.MetricName,
				err,
			)
		}
	}

	return nil
}
