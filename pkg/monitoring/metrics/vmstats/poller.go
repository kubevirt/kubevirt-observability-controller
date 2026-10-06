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
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
	k6tv1 "kubevirt.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

var pollerLog = ctrl.Log.WithName("vmstats-poller")

type PollerConfig struct {
	PollInterval  time.Duration
	MaxConcurrent int
	Port          int
}

type Poller struct {
	config   PollerConfig
	cache    *StatsCache
	client   *VMStatsClient
	vmiStore cache.Store
	podStore cache.Store

	// polling guards against overlapping poll cycles. When a poll takes
	// longer than PollInterval the next tick is skipped instead of piling
	// up concurrent polls that would overwhelm virt-handler / libvirt.
	polling atomic.Bool
}

func NewPoller(
	config PollerConfig, statsCache *StatsCache, client *VMStatsClient,
	vmiStore cache.Store, podStore cache.Store,
) *Poller {
	return &Poller{
		config:   config,
		cache:    statsCache,
		client:   client,
		vmiStore: vmiStore,
		podStore: podStore,
	}
}

func (p *Poller) Start(ctx context.Context) error {
	ticker := time.NewTicker(p.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if !p.polling.CompareAndSwap(false, true) {
				pollerLog.V(4).Info("previous poll still running, skipping tick")
				continue
			}
			p.pollOnce()
			p.polling.Store(false)
		}
	}
}

func (p *Poller) pollOnce() {
	if p.vmiStore == nil || p.podStore == nil {
		pollerLog.V(4).Info("stores not ready, skipping poll")
		return
	}
	nodeVMIs := groupVMIsByNode(p.vmiStore.List())

	activeKeys := make(map[string]bool)
	var activeKeysMu sync.Mutex

	if len(nodeVMIs) > 0 {
		sem := make(chan struct{}, p.config.MaxConcurrent)
		var wg sync.WaitGroup

		for node, vmis := range nodeVMIs {
			vmiLookup := make(map[string]*k6tv1.VirtualMachineInstance, len(vmis))
			for _, vmi := range vmis {
				key := vmi.Namespace + "/" + vmi.Name
				vmiLookup[key] = vmi
				activeKeysMu.Lock()
				activeKeys[key] = true
				activeKeysMu.Unlock()
			}

			podIP, err := findVirtHandlerPodIP(p.podStore, node)
			if err != nil {
				pollerLog.V(4).Info("skipping node: no virt-handler pod", "node", node, "error", err)
				continue
			}

			wg.Add(1)
			sem <- struct{}{}
			go func(node, podIP string, vmiLookup map[string]*k6tv1.VirtualMachineInstance) {
				defer wg.Done()
				defer func() { <-sem }()

				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				results, err := p.client.FetchNodeVMStats(ctx, podIP)
				if err != nil {
					pollerLog.V(4).Info("failed to fetch node vmstats", "node", node, "error", err)
					return
				}

				for key, result := range results {
					if result.Error != "" {
						pollerLog.V(4).Info("vmstats error for VMI", "vmi", key, "error", result.Error)
						continue
					}
					if result.Stats == nil {
						continue
					}
					vmi, ok := vmiLookup[key]
					if !ok {
						continue
					}
					p.cache.Store(key, vmi, result.Stats)
				}
			}(node, podIP, vmiLookup)
		}

		wg.Wait()
	}

	p.cache.Prune(activeKeys)
}

// groupVMIsByNode groups VMIs by their scheduled node, filtering out VMIs
// whose lifecycle state makes stats collection futile or harmful:
//   - VMIs not in Running phase (qemu-ga is unavailable)
//   - VMIs with an active live migration (qemu-ga is unresponsive and
//     dirty-rate calculation conflicts with migration)
func groupVMIsByNode(objs []any) map[string][]*k6tv1.VirtualMachineInstance {
	result := make(map[string][]*k6tv1.VirtualMachineInstance)
	for _, obj := range objs {
		vmi, ok := obj.(*k6tv1.VirtualMachineInstance)
		if !ok || vmi.Status.NodeName == "" {
			continue
		}
		if !isEligibleForStats(vmi) {
			continue
		}
		result[vmi.Status.NodeName] = append(result[vmi.Status.NodeName], vmi)
	}
	return result
}

// isEligibleForStats returns true when a VMI is in a lifecycle state where
// stats collection is safe and useful. The checks mirror the existing
// filterDirtyRateEligible logic in kubevirt's virt-handler.
func isEligibleForStats(vmi *k6tv1.VirtualMachineInstance) bool {
	if vmi.Status.Phase != k6tv1.Running {
		return false
	}
	if ms := vmi.Status.MigrationState; ms != nil && ms.StartTimestamp != nil && !ms.Completed && !ms.Failed {
		return false
	}
	return true
}

func findVirtHandlerPodIP(podStore cache.Store, nodeName string) (string, error) {
	for _, obj := range podStore.List() {
		pod, ok := obj.(*k8sv1.Pod)
		if !ok {
			continue
		}
		if pod.Labels["kubevirt.io"] == "virt-handler" &&
			pod.Spec.NodeName == nodeName &&
			pod.Status.Phase == k8sv1.PodRunning &&
			pod.Status.PodIP != "" {
			return pod.Status.PodIP, nil
		}
	}
	return "", fmt.Errorf("no running virt-handler pod found for node %s", nodeName)
}
