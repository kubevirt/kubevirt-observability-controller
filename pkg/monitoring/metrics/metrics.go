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
	"sync/atomic"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"k8s.io/client-go/tools/cache"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/metrics/vm"
	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/metrics/vmi"
)

var (
	storesRef   atomic.Pointer[Stores]
	indexersRef atomic.Pointer[Indexers]
)

func getStores() *Stores     { return storesRef.Load() }
func getIndexers() *Indexers { return indexersRef.Load() }

// SetupMetrics registers metric collectors filtered by the given allowlist.
// A nil allowlist registers all metrics. An empty (non-nil) map registers none.
func SetupMetrics(metricsStores *Stores, metricsIndexers *Indexers, allowlist map[string]bool) error {
	if metricsStores == nil {
		metricsStores = &Stores{}
	}
	storesRef.Store(metricsStores)

	if metricsIndexers == nil {
		metricsIndexers = &Indexers{}
	}
	indexersRef.Store(metricsIndexers)

	operatormetrics.Register = ctrlmetrics.Registry.Register
	operatormetrics.Unregister = ctrlmetrics.Registry.Unregister

	allCollectors := []operatormetrics.Collector{
		vmi.NewMigrationStatsCollector(func() cache.Store {
			idx := getIndexers()
			if idx == nil {
				return nil
			}
			return idx.VMIMigration
		}),
		VMIStatsCollector,
		VMStatsCollector,
		vm.NewVMSnapshotStatsCollector(inventoryStore(func(s *Stores) cache.Store { return s.VMSnapshot })),
		vm.NewVMRestoreStatsCollector(inventoryStore(func(s *Stores) cache.Store { return s.VMRestore })),
		vm.NewVMExportStatsCollector(inventoryStore(func(s *Stores) cache.Store { return s.VMExport })),
		vm.NewVMCloneStatsCollector(inventoryStore(func(s *Stores) cache.Store { return s.VMClone })),
		vm.NewVMPoolStatsCollector(inventoryStore(func(s *Stores) cache.Store { return s.VMPool })),
	}

	if allowlist == nil {
		return operatormetrics.RegisterCollector(allCollectors...)
	}

	var filtered []operatormetrics.Collector
	for _, c := range allCollectors {
		if fc := filterCollector(c, allowlist); fc != nil {
			filtered = append(filtered, *fc)
		}
	}

	if len(filtered) == 0 {
		return nil
	}

	return operatormetrics.RegisterCollector(filtered...)
}

func filterCollector(c operatormetrics.Collector, allowlist map[string]bool) *operatormetrics.Collector {
	var kept []operatormetrics.Metric
	for _, m := range c.Metrics {
		if allowlist[m.GetOpts().Name] {
			kept = append(kept, m)
		}
	}

	if len(kept) == 0 {
		return nil
	}

	allowedSet := make(map[string]bool, len(kept))
	for _, m := range kept {
		allowedSet[m.GetOpts().Name] = true
	}

	originalCallback := c.CollectCallback
	return &operatormetrics.Collector{
		Metrics: kept,
		CollectCallback: func() []operatormetrics.CollectorResult {
			results := originalCallback()
			filtered := make([]operatormetrics.CollectorResult, 0, len(results))
			for _, r := range results {
				if allowedSet[r.Metric.GetOpts().Name] {
					filtered = append(filtered, r)
				}
			}
			return filtered
		},
	}
}

func GetStores() *Stores { return getStores() }

func SetStores(s *Stores, i *Indexers) {
	storesRef.Store(s)
	indexersRef.Store(i)
}

func ListMetrics() []operatormetrics.Metric {
	return operatormetrics.ListMetrics()
}

func inventoryStore(selectStore func(*Stores) cache.Store) func() cache.Store {
	return func() cache.Store {
		stores := getStores()
		if stores == nil {
			return nil
		}
		return selectStore(stores)
	}
}
