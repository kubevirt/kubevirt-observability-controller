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

package recordingrules

import (
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatorrules"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/groups"
)

var networkAttachmentDefinitionRecordingRules = []operatorrules.RecordingRule{
	{
		MetricsOpts: operatormetrics.MetricOpts{
			Name:        "cluster:kubevirt_network_attachment_definition_info:sum",
			ExtraFields: groups.Tag(groups.Inventory),
			Help: "Number of VirtualMachine interfaces attached to a NetworkAttachmentDefinition with this configuration. " +
				"Keeps cni_type, ipam_type, udn_role, ovn_topology, and ovn_persistent_ips. " +
				"A namespace/name reference is counted on that NetworkAttachmentDefinition. " +
				"A NetworkAttachmentDefinition with more than one CNI plugin counts those interfaces on each plugin.",
		},
		MetricType: operatormetrics.GaugeType,
		Expr: intstr.FromString(
			"sum by (cni_type, ipam_type, udn_role, ovn_topology, ovn_persistent_ips) (" +
				"kubevirt_network_attachment_definition_info " +
				"* on (namespace, network) group_left() " +
				"count by (namespace, network) (" +
				"label_replace(label_replace(label_replace(" +
				"kubevirt_vm_vnic_info{network=~\".+/.+\"}, " +
				"\"vm_namespace\", \"$1\", \"namespace\", \"(.+)\"), " +
				"\"namespace\", \"$1\", \"network\", \"([^/]+)/.+\"), " +
				"\"network\", \"$1\", \"network\", \"[^/]+/(.+)\") " +
				"or kubevirt_vm_vnic_info{network!=\"pod networking\",network!~\".+/.+\"}" +
				"))",
		),
	},
}
