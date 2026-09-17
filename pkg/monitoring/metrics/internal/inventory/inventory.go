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

package inventory

import (
	"strings"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

const (
	None       = ""
	PhaseUnset = "unset"
)

func listStoreObjects[T any](store cache.Store) []*T {
	cachedObjs := store.List()
	items := make([]*T, 0, len(cachedObjs))
	for _, obj := range cachedObjs {
		typed, ok := obj.(*T)
		if !ok {
			continue
		}
		items = append(items, typed)
	}
	return items
}

func ResourcePhaseLabel(phase string) string {
	if phase == "" {
		return PhaseUnset
	}
	return strings.ToLower(phase)
}

func TypedLocalObjectName(ref *k8sv1.TypedLocalObjectReference) string {
	if ref == nil {
		return None
	}
	return ref.Name
}

func TypedLocalObjectKind(ref *k8sv1.TypedLocalObjectReference) string {
	if ref == nil {
		return None
	}
	return ref.Kind
}

func OptionalStringLabel(value *string) string {
	if value == nil || *value == "" {
		return None
	}
	return *value
}

func BoolGaugeValue(enabled bool) float64 {
	if enabled {
		return 1
	}
	return 0
}

func CollectUnixTimestamp(
	metric operatormetrics.Metric,
	timestamp metav1.Time,
	labels []string,
) []operatormetrics.CollectorResult {
	if timestamp.IsZero() {
		return nil
	}
	return []operatormetrics.CollectorResult{{
		Metric: metric,
		Value:  float64(timestamp.Unix()),
		Labels: labels,
	}}
}

func CollectOptionalUnixTimestamp(
	metric operatormetrics.Metric,
	timestamp *metav1.Time,
	labels []string,
) []operatormetrics.CollectorResult {
	if timestamp == nil {
		return nil
	}
	return CollectUnixTimestamp(metric, *timestamp, labels)
}

// NewCollector reads the active informer store on each scrape. Stores may be
// initialized or replaced after the collector is registered.
func NewCollector[T any](
	metrics []operatormetrics.Metric,
	getStore func() cache.Store,
	report func([]*T) []operatormetrics.CollectorResult,
) operatormetrics.Collector {
	return operatormetrics.Collector{
		Metrics: metrics,
		CollectCallback: func() []operatormetrics.CollectorResult {
			store := getStore()
			if store == nil {
				return nil
			}
			return report(listStoreObjects[T](store))
		},
	}
}
