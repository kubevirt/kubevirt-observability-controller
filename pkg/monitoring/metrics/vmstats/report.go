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
