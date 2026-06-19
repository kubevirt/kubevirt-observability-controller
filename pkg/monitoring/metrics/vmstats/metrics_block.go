package vmstats

import "github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"

var (
	blockMetricsList = []operatormetrics.Metric{
		storageIopsRead, storageIopsWrite,
		storageReadTrafficBytes, storageWriteTrafficBytes,
		storageReadTimesSeconds, storageWriteTimesSeconds,
		storageFlushRequests, storageFlushTimesSeconds,
	}

	storageIopsRead = operatormetrics.NewCounter(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_storage_iops_read_total",
		Help: "Total number of I/O read operations.",
	})
	storageIopsWrite = operatormetrics.NewCounter(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_storage_iops_write_total",
		Help: "Total number of I/O write operations.",
	})
	storageReadTrafficBytes = operatormetrics.NewCounter(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_storage_read_traffic_bytes_total",
		Help: "Total number of bytes read from storage.",
	})
	storageWriteTrafficBytes = operatormetrics.NewCounter(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_storage_write_traffic_bytes_total",
		Help: "Total number of written bytes.",
	})
	storageReadTimesSeconds = operatormetrics.NewCounter(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_storage_read_times_seconds_total",
		Help: "Total time spent on read operations.",
	})
	storageWriteTimesSeconds = operatormetrics.NewCounter(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_storage_write_times_seconds_total",
		Help: "Total time spent on write operations.",
	})
	storageFlushRequests = operatormetrics.NewCounter(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_storage_flush_requests_total",
		Help: "Total storage flush requests.",
	})
	storageFlushTimesSeconds = operatormetrics.NewCounter(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_storage_flush_times_seconds_total",
		Help: "Total time spent on cache flushing.",
	})
)

func collectBlockMetrics(report *VMIReport) []operatormetrics.CollectorResult {
	var crs []operatormetrics.CollectorResult

	if report.Stats.DomainStats.Block == nil {
		return crs
	}

	for _, block := range report.Stats.DomainStats.Block {
		if !block.NameSet {
			continue
		}

		drive := block.Name
		if block.Alias != "" {
			drive = block.Alias
		}
		blkLabels := map[string]string{"drive": drive}

		if block.RdReqsSet {
			crs = append(crs, report.newCollectorResultWithLabels(storageIopsRead, float64(block.RdReqs), blkLabels))
		}
		if block.WrReqsSet {
			crs = append(crs, report.newCollectorResultWithLabels(storageIopsWrite, float64(block.WrReqs), blkLabels))
		}
		if block.RdBytesSet {
			crs = append(crs, report.newCollectorResultWithLabels(storageReadTrafficBytes, float64(block.RdBytes), blkLabels))
		}
		if block.WrBytesSet {
			crs = append(crs, report.newCollectorResultWithLabels(storageWriteTrafficBytes, float64(block.WrBytes), blkLabels))
		}
		if block.RdTimesSet {
			rdTime := nanosecondsToSeconds(block.RdTimes)
			crs = append(crs, report.newCollectorResultWithLabels(storageReadTimesSeconds, rdTime, blkLabels))
		}
		if block.WrTimesSet {
			wrTime := nanosecondsToSeconds(block.WrTimes)
			crs = append(crs, report.newCollectorResultWithLabels(storageWriteTimesSeconds, wrTime, blkLabels))
		}
		if block.FlReqsSet {
			crs = append(crs, report.newCollectorResultWithLabels(storageFlushRequests, float64(block.FlReqs), blkLabels))
		}
		if block.FlTimesSet {
			flTime := nanosecondsToSeconds(block.FlTimes)
			crs = append(crs, report.newCollectorResultWithLabels(storageFlushTimesSeconds, flTime, blkLabels))
		}
	}

	return crs
}
