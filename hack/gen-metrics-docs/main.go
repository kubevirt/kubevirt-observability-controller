package main

import (
	"fmt"
	"strings"

	"github.com/rhobs/operator-observability-toolkit/pkg/docs"

	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/groups"
	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/metrics"
	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/metrics/devicedrivers"
	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/metrics/vmstats"
	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/rules"
)

const title = "KubeVirt Observability Controller Metrics and Recording Rules"

// The Group column is the whole point of the custom template: it is what an
// operator needs in order to turn a metric on or off, and it already rides
// along in ExtraFields.
const metricsTemplate = `# {{.Title}}

| Name | Group | Kind | Type | Description |
|------|-------|------|------|-------------|
{{- range .Metrics }}
| {{.Name}} | ` + "`{{ index .ExtraFields \"group\" }}`" + ` | {{.Kind}} | {{.Type}} | {{.Description}} |
{{- end }}
`

const groupsIntro = `
## Groups

Metrics, recording rules, and alerts are selected by group. Groups marked
` + "`on`" + ` are enabled when no flag selects otherwise, which are exactly the
ones KubeVirt does not publish itself. See
[deployment.md](deployment.md#selecting-what-to-expose) for the flag grammar.

| Group | Default | Description |
|-------|---------|-------------|
`

const footer = `
## Developing new metrics

All metrics documented here are auto-generated and reflect exactly what is being
exposed. After developing new metrics or changing old ones please regenerate
this document.
`

func main() {
	if err := metrics.SetupMetrics(nil, nil, nil); err != nil {
		panic(err)
	}

	if err := vmstats.RegisterCollector(vmstats.NewStatsCache(), nil); err != nil {
		panic(err)
	}

	if err := devicedrivers.RegisterCollector(devicedrivers.NewDriversCache(), nil); err != nil {
		panic(err)
	}

	if err := rules.SetupRules("", nil, nil); err != nil {
		panic(err)
	}

	metricsList := metrics.ListMetrics()
	rulesList := rules.ListRecordingRules()

	var out strings.Builder
	out.WriteString(docs.BuildMetricsDocsWithCustomTemplate(title, metricsList, rulesList, metricsTemplate))
	out.WriteString(groupsIntro)
	for _, g := range groups.All {
		state := "off"
		if g.Default {
			state = "on"
		}
		fmt.Fprintf(&out, "| `%s` | %s | %s |\n", g.Name, state, g.Description)
	}
	out.WriteString(footer)

	fmt.Print(out.String())
}
