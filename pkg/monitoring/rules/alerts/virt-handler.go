package alerts

import (
	"fmt"

	promv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

func virtHandlerAlerts(namespace string) []promv1.Rule {
	return []promv1.Rule{
		{
			Alert: "VirtHandlerDaemonSetRolloutFailing",
			Expr: intstr.FromString(
				fmt.Sprintf(
					"(%s - %s) != 0",
					fmt.Sprintf(
						"kube_daemonset_status_number_ready{namespace='%s', daemonset='virt-handler'}",
						namespace,
					),
					fmt.Sprintf(
						"kube_daemonset_status_desired_number_scheduled"+
							"{namespace='%s', daemonset='virt-handler'}",
						namespace,
					),
				),
			),
			For: ptr.To(promv1.Duration("15m")),
			Annotations: map[string]string{
				summaryAnnotationKey: "Some virt-handlers failed to roll out",
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "warning",
				operatorHealthImpactLabelKey: "warning",
			},
		},
		{
			Alert: "VirtHandlerRESTErrorsBurst",
			Expr:  intstr.FromString(getErrorRatio(namespace, "virt-handler") + " >= 0.8"),
			For:   ptr.To(promv1.Duration("5m")),
			Annotations: map[string]string{
				summaryAnnotationKey: getRestCallsFailedWarning("virt-handler"),
			},
			Labels: map[string]string{
				severityAlertLabelKey:        "critical",
				operatorHealthImpactLabelKey: "critical",
			},
		},
	}
}
