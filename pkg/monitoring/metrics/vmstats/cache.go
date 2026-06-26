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
	"sync"

	k6tv1 "kubevirt.io/api/core/v1"
)

type CachedStats struct {
	VMI   *k6tv1.VirtualMachineInstance
	Stats *VMStats
}

type StatsCache struct {
	mu    sync.RWMutex
	items map[string]*CachedStats
}

func NewStatsCache() *StatsCache {
	return &StatsCache{
		items: make(map[string]*CachedStats),
	}
}

func (c *StatsCache) Store(key string, vmi *k6tv1.VirtualMachineInstance, stats *VMStats) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = &CachedStats{VMI: vmi, Stats: stats}
}

func (c *StatsCache) List() []*CachedStats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result := make([]*CachedStats, 0, len(c.items))
	for _, v := range c.items {
		result = append(result, v)
	}
	return result
}

func (c *StatsCache) Prune(activeKeys map[string]bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.items {
		if !activeKeys[k] {
			delete(c.items, k)
		}
	}
}

func (c *StatsCache) Remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}
