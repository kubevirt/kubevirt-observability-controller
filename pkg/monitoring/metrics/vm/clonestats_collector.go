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

package vm

import (
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"k8s.io/client-go/tools/cache"
	clonev1 "kubevirt.io/api/clone/v1beta1"

	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/metrics/internal/inventory"
)

// NewVMCloneStatsCollector collects clone metrics from the current informer store.
func NewVMCloneStatsCollector(getStore func() cache.Store) operatormetrics.Collector {
	return inventory.NewCollector([]operatormetrics.Metric{
		vmCloneInfo,
		vmCloneCreationTimestamp,
	}, getStore, reportVMCloneStats)
}

var (
	vmCloneInfo = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmclone_info",
			Help: "Information about VirtualMachineClones.",
		},
		[]string{
			"namespace", "name", "uid", "source", "source_kind", "target_vm",
			"snapshot_name", "restore_name", "phase",
		},
	)

	vmCloneCreationTimestamp = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmclone_create_date_timestamp_seconds",
			Help: "Virtual Machine Clone creation timestamp.",
		},
		[]string{"namespace", "name"},
	)
)

func reportVMCloneStats(clones []*clonev1.VirtualMachineClone) []operatormetrics.CollectorResult {
	results := make([]operatormetrics.CollectorResult, 0, 2*len(clones))
	for _, vmClone := range clones {
		results = append(results, collectVMCloneInfo(vmClone))
		results = append(results, collectVMCloneCreationTimestamp(vmClone)...)
	}
	return results
}

func collectVMCloneInfo(vmClone *clonev1.VirtualMachineClone) operatormetrics.CollectorResult {
	return operatormetrics.CollectorResult{
		Metric: vmCloneInfo,
		Value:  1,
		Labels: []string{
			vmClone.Namespace,
			vmClone.Name,
			string(vmClone.UID),
			inventory.TypedLocalObjectName(vmClone.Spec.Source),
			inventory.TypedLocalObjectKind(vmClone.Spec.Source),
			cloneTargetVM(vmClone),
			inventory.OptionalStringLabel(vmClone.Status.SnapshotName),
			inventory.OptionalStringLabel(vmClone.Status.RestoreName),
			inventory.ResourcePhaseLabel(string(vmClone.Status.Phase)),
		},
	}
}

func cloneTargetVM(vmClone *clonev1.VirtualMachineClone) string {
	if name := inventory.TypedLocalObjectName(vmClone.Spec.Target); name != inventory.None {
		return name
	}
	return inventory.OptionalStringLabel(vmClone.Status.TargetName)
}

func collectVMCloneCreationTimestamp(vmClone *clonev1.VirtualMachineClone) []operatormetrics.CollectorResult {
	return inventory.CollectUnixTimestamp(
		vmCloneCreationTimestamp,
		vmClone.CreationTimestamp,
		[]string{vmClone.Namespace, vmClone.Name},
	)
}
