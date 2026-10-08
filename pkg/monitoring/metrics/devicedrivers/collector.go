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

package devicedrivers

import (
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/metrics"
)

var latestVersionInfo = operatormetrics.NewGaugeVec(
	operatormetrics.MetricOpts{
		Name: "kubevirt_guest_device_driver_latest_version_info",
		Help: "Latest available guest device driver version for a device and " +
			"guest OS version, as declared in the guest device drivers ConfigMap.",
	},
	[]string{"device_id", "guest_os_version_id", "driver_version"},
)

func RegisterCollector(cache *DriversCache, allowlist map[string]bool) error {
	operatormetrics.Register = ctrlmetrics.Registry.Register
	operatormetrics.Unregister = ctrlmetrics.Registry.Unregister

	collector := driversCollector(cache)

	if allowlist == nil {
		return operatormetrics.RegisterCollector(collector)
	}

	filtered := metrics.FilterCollector(collector, allowlist)
	if filtered == nil {
		return nil
	}

	return operatormetrics.RegisterCollector(*filtered)
}

func driversCollector(cache *DriversCache) operatormetrics.Collector {
	return operatormetrics.Collector{
		Metrics: []operatormetrics.Metric{latestVersionInfo},
		CollectCallback: func() []operatormetrics.CollectorResult {
			entries := cache.List()
			if len(entries) == 0 {
				return nil
			}

			crs := make([]operatormetrics.CollectorResult, 0, len(entries))
			for _, entry := range entries {
				crs = append(crs, operatormetrics.CollectorResult{
					Metric: latestVersionInfo,
					Labels: []string{
						entry.DeviceID,
						entry.GuestOSVersionID,
						entry.DriverVersion,
					},
					Value: 1.0,
				})
			}

			return crs
		},
	}
}
