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
	"strings"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	k6tv1 "kubevirt.io/api/core/v1"
)

var labelFormatter = strings.NewReplacer(".", "_", "/", "_", "-", "_")

const labelPrefix = "kubernetes_vmi_label_"

type VMIReport struct {
	VMI           *k6tv1.VirtualMachineInstance
	Stats         *VMStats
	runtimeLabels map[string]string
}

func NewVMIReport(vmi *k6tv1.VirtualMachineInstance, stats *VMStats) *VMIReport {
	r := &VMIReport{VMI: vmi, Stats: stats}
	r.buildRuntimeLabels()
	return r
}

func (r *VMIReport) buildRuntimeLabels() {
	r.runtimeLabels = make(map[string]string)
	for label, val := range r.VMI.Labels {
		key := labelPrefix + labelFormatter.Replace(label)
		r.runtimeLabels[key] = val
	}
}

func (r *VMIReport) newCollectorResult(metric operatormetrics.Metric, value float64) operatormetrics.CollectorResult {
	return r.newCollectorResultWithLabels(metric, value, nil)
}

func (r *VMIReport) newCollectorResultWithLabels(
	metric operatormetrics.Metric, value float64, additionalLabels map[string]string,
) operatormetrics.CollectorResult {
	labels := map[string]string{
		"node":      r.VMI.Status.NodeName,
		"namespace": r.VMI.Namespace,
		"name":      r.VMI.Name,
	}
	for k, v := range r.runtimeLabels {
		labels[k] = v
	}
	for k, v := range additionalLabels {
		labels[k] = v
	}
	return operatormetrics.CollectorResult{
		Metric:      metric,
		ConstLabels: labels,
		Value:       value,
	}
}
