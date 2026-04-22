package metrics

import "k8s.io/client-go/tools/cache"

type Stores struct {
	VM                  cache.Store
	VMI                 cache.Store
	PVC                 cache.Store
	Instancetype        cache.Store
	ClusterInstancetype cache.Store
	Preference          cache.Store
	ClusterPreference   cache.Store
	ControllerRevision  cache.Store
}

type Indexers struct {
	VMIMigration cache.Indexer
	KVPod        cache.Indexer
}
