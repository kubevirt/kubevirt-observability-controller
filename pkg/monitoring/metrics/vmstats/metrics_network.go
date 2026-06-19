package vmstats

import "github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"

var (
	networkMetricsList = []operatormetrics.Metric{
		networkReceiveBytes, networkTransmitBytes,
		networkReceivePackets, networkTransmitPackets,
		networkReceiveErrors, networkTransmitErrors,
		networkReceivePacketsDropped, networkTransmitPacketsDropped,
	}

	networkReceiveBytes = operatormetrics.NewCounter(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_network_receive_bytes_total",
		Help: "Total network traffic received in bytes.",
	})
	networkTransmitBytes = operatormetrics.NewCounter(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_network_transmit_bytes_total",
		Help: "Total network traffic transmitted in bytes.",
	})
	networkReceivePackets = operatormetrics.NewCounter(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_network_receive_packets_total",
		Help: "Total network traffic received packets.",
	})
	networkTransmitPackets = operatormetrics.NewCounter(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_network_transmit_packets_total",
		Help: "Total network traffic transmitted packets.",
	})
	networkReceiveErrors = operatormetrics.NewCounter(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_network_receive_errors_total",
		Help: "Total network received error packets.",
	})
	networkTransmitErrors = operatormetrics.NewCounter(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_network_transmit_errors_total",
		Help: "Total network transmitted error packets.",
	})
	networkReceivePacketsDropped = operatormetrics.NewCounter(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_network_receive_packets_dropped_total",
		Help: "The total number of rx packets dropped on vNIC interfaces.",
	})
	networkTransmitPacketsDropped = operatormetrics.NewCounter(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_network_transmit_packets_dropped_total",
		Help: "The total number of tx packets dropped on vNIC interfaces.",
	})
)

func collectNetworkMetrics(report *VMIReport) []operatormetrics.CollectorResult {
	var crs []operatormetrics.CollectorResult

	if report.Stats.DomainStats.Net == nil {
		return crs
	}

	for _, net := range report.Stats.DomainStats.Net {
		if !net.NameSet {
			continue
		}

		iface := net.Name
		if net.AliasSet {
			iface = net.Alias
		}
		netLabels := map[string]string{"interface": iface}

		if net.RxBytesSet {
			crs = append(crs, report.newCollectorResultWithLabels(networkReceiveBytes, float64(net.RxBytes), netLabels))
		}
		if net.TxBytesSet {
			crs = append(crs, report.newCollectorResultWithLabels(networkTransmitBytes, float64(net.TxBytes), netLabels))
		}
		if net.RxPktsSet {
			crs = append(crs, report.newCollectorResultWithLabels(networkReceivePackets, float64(net.RxPkts), netLabels))
		}
		if net.TxPktsSet {
			crs = append(crs, report.newCollectorResultWithLabels(networkTransmitPackets, float64(net.TxPkts), netLabels))
		}
		if net.RxErrsSet {
			crs = append(crs, report.newCollectorResultWithLabels(networkReceiveErrors, float64(net.RxErrs), netLabels))
		}
		if net.TxErrsSet {
			crs = append(crs, report.newCollectorResultWithLabels(networkTransmitErrors, float64(net.TxErrs), netLabels))
		}
		if net.RxDropSet {
			crs = append(crs, report.newCollectorResultWithLabels(networkReceivePacketsDropped, float64(net.RxDrop), netLabels))
		}
		if net.TxDropSet {
			crs = append(crs, report.newCollectorResultWithLabels(networkTransmitPacketsDropped, float64(net.TxDrop), netLabels))
		}
	}

	return crs
}
