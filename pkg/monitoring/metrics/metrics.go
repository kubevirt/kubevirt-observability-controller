package metrics

import (
	"sync/atomic"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var (
	storesRef   atomic.Pointer[Stores]
	indexersRef atomic.Pointer[Indexers]
)

func getStores() *Stores     { return storesRef.Load() }
func getIndexers() *Indexers { return indexersRef.Load() }

func SetupMetrics(metricsStores *Stores, metricsIndexers *Indexers) error {
	if metricsStores == nil {
		metricsStores = &Stores{}
	}
	storesRef.Store(metricsStores)

	if metricsIndexers == nil {
		metricsIndexers = &Indexers{}
	}
	indexersRef.Store(metricsIndexers)

	operatormetrics.Register = ctrlmetrics.Registry.Register

	return operatormetrics.RegisterCollector(
		MigrationStatsCollector,
		VMIStatsCollector,
		VMStatsCollector,
	)
}

func SetStores(s *Stores, i *Indexers) {
	storesRef.Store(s)
	indexersRef.Store(i)
}

func ListMetrics() []operatormetrics.Metric {
	return operatormetrics.ListMetrics()
}
