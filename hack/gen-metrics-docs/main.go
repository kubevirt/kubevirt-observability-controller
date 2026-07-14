package main

import (
	"fmt"

	"github.com/rhobs/operator-observability-toolkit/pkg/docs"

	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/metrics"
	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/rules"
)

const title = "KubeVirt Observability Controller Metrics and Recording Rules"

func main() {
	err := metrics.SetupMetrics(nil, nil, nil)
	if err != nil {
		panic(err)
	}

	err = rules.SetupRules("", nil, nil)
	if err != nil {
		panic(err)
	}

	metricsList := metrics.ListMetrics()
	rulesList := rules.ListRecordingRules()

	docsString := docs.BuildMetricsDocs(title, metricsList, rulesList)
	fmt.Print(docsString)
}
