package main

import (
	"fmt"

	"sigs.k8s.io/yaml"

	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/rules"
)

const ruleNamespace = "kubevirt"

func main() {
	if err := rules.SetupRules(ruleNamespace, nil, nil); err != nil {
		panic(err)
	}

	pr, err := rules.BuildPrometheusRule("kubevirt-observability-rules", ruleNamespace, nil)
	if err != nil {
		panic(err)
	}

	out, err := yaml.Marshal(pr.Spec)
	if err != nil {
		panic(err)
	}

	fmt.Print(string(out))
}
