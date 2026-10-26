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
	"encoding/json"
	"fmt"
	"math"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/metrics"
)

var guestLog = ctrl.Log.WithName("vmstats-guest")

var (
	guestMetricsList = []operatormetrics.Metric{
		guestHostname, guestTimezone,
		guestUserCount, filesystemCapacityBytes, filesystemUsedBytes,
		guestDeviceDriverTimestamp,
	}

	guestHostname = operatormetrics.NewGauge(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_guest_hostname",
		Help: "Guest hostname from the guest agent.",
	})
	guestTimezone = operatormetrics.NewGauge(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_guest_timezone",
		Help: "Guest timezone from the guest agent.",
	})
	guestUserCount = operatormetrics.NewGauge(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_guest_user_count",
		Help: "Number of logged-in users in the guest.",
	})
	filesystemCapacityBytes = operatormetrics.NewGauge(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_filesystem_capacity_bytes",
		Help: "Total VM filesystem capacity in bytes.",
	})
	filesystemUsedBytes = operatormetrics.NewGauge(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_filesystem_used_bytes",
		Help: "Used VM filesystem capacity in bytes.",
	})
	guestDeviceDriverTimestamp = operatormetrics.NewGauge(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_guest_device_driver_timestamp_seconds",
		Help: "Release date of the driver of a guest device, in seconds since the epoch, " +
			"as reported by the guest agent. Windows guests only.",
	})
)

func collectGuestMetrics(report *VMIReport) []operatormetrics.CollectorResult {
	var crs []operatormetrics.CollectorResult //nolint:prealloc
	crs = append(crs, collectGuestHostname(report)...)
	crs = append(crs, collectGuestTimezone(report)...)
	crs = append(crs, collectGuestUsers(report)...)
	crs = append(crs, collectGuestFilesystems(report)...)
	crs = append(crs, collectGuestDevices(report)...)
	return crs
}

func guestReturn(payload string) json.RawMessage {
	if payload == "" {
		return nil
	}
	var env struct {
		Return json.RawMessage `json:"return"`
	}
	if err := json.Unmarshal([]byte(payload), &env); err != nil {
		return nil
	}
	return env.Return
}

func collectGuestHostname(report *VMIReport) []operatormetrics.CollectorResult {
	raw := guestReturn(report.Stats.GuestGetHostName)
	if raw == nil {
		return nil
	}
	var info struct {
		HostName string `json:"host-name"`
	}
	if err := json.Unmarshal(raw, &info); err != nil {
		return nil
	}
	return []operatormetrics.CollectorResult{
		report.newCollectorResultWithLabels(guestHostname, 1.0, map[string]string{
			"hostname": info.HostName,
		}),
	}
}

func collectGuestTimezone(report *VMIReport) []operatormetrics.CollectorResult {
	raw := guestReturn(report.Stats.GuestGetTimezone)
	if raw == nil {
		return nil
	}
	var info struct {
		Zone   string `json:"zone"`
		Offset int    `json:"offset"`
	}
	if err := json.Unmarshal(raw, &info); err != nil {
		return nil
	}
	return []operatormetrics.CollectorResult{
		report.newCollectorResultWithLabels(guestTimezone, 1.0, map[string]string{
			"timezone": info.Zone,
			"offset":   fmt.Sprintf("%d", info.Offset),
		}),
	}
}

func collectGuestUsers(report *VMIReport) []operatormetrics.CollectorResult {
	raw := guestReturn(report.Stats.GuestGetUsers)
	if raw == nil {
		return nil
	}
	var users []json.RawMessage
	if err := json.Unmarshal(raw, &users); err != nil {
		return nil
	}
	return []operatormetrics.CollectorResult{
		report.newCollectorResult(guestUserCount, float64(len(users))),
	}
}

func collectGuestFilesystems(report *VMIReport) []operatormetrics.CollectorResult {
	raw := guestReturn(report.Stats.GuestGetFsInfo)
	if raw == nil {
		return nil
	}
	var filesystems []struct {
		Name       string `json:"name"`
		Mountpoint string `json:"mountpoint"`
		Type       string `json:"type"`
		Total      uint64 `json:"total-bytes"`
		Used       uint64 `json:"used-bytes"`
	}
	if err := json.Unmarshal(raw, &filesystems); err != nil {
		return nil
	}
	var crs []operatormetrics.CollectorResult
	for _, fs := range filesystems {
		labels := map[string]string{
			"disk_name":        fs.Name,
			"mount_point":      fs.Mountpoint,
			"file_system_type": fs.Type,
		}
		crs = append(crs,
			report.newCollectorResultWithLabels(filesystemCapacityBytes, float64(fs.Total), labels),
			report.newCollectorResultWithLabels(filesystemUsedBytes, float64(fs.Used), labels),
		)
	}
	return crs
}

func collectGuestDevices(report *VMIReport) []operatormetrics.CollectorResult {
	raw := guestReturn(report.Stats.GuestGetDevices)
	if raw == nil {
		return nil
	}

	var devices []guestDevice
	if err := json.Unmarshal(raw, &devices); err != nil {
		guestLog.V(2).Info("discarding unparseable guest device data",
			"namespace", report.VMI.Namespace, "name", report.VMI.Name, "error", err)
		return nil
	}

	osVersionID := report.VMI.Status.GuestOSInfo.VersionID

	var crs []operatormetrics.CollectorResult
	seen := make(map[deviceKey]struct{}, len(devices))
	for _, d := range devices {
		if d.DriverDate == nil || *d.DriverDate <= 0 {
			continue
		}
		key := deviceKey{
			name:    d.DriverName,
			version: d.DriverVersion,
		}
		if d.ID != nil {
			key.deviceID = formatPCIID(d.ID.DeviceID)
		}

		if _, dup := seen[key]; dup {
			guestLog.V(2).Info("dropping guest device with a duplicate label set",
				"namespace", report.VMI.Namespace, "name", report.VMI.Name,
				"driver_name", key.name, "driver_version", key.version, "device_id", key.deviceID)
			continue
		}
		seen[key] = struct{}{}

		crs = append(crs, report.newCollectorResultWithLabels(
			guestDeviceDriverTimestamp, nanosecondsToSeconds(uint64(*d.DriverDate)), map[string]string{
				"driver_name":         key.name,
				"driver_version":      key.version,
				"device_id":           key.deviceID,
				"guest_os_version_id": osVersionID,
			}))
	}
	return crs
}

type guestDevice struct {
	DriverName    string `json:"driver-name"`
	DriverDate    *int64 `json:"driver-date"`
	DriverVersion string `json:"driver-version"`
	ID            *struct {
		DeviceID *int64 `json:"device-id"`
	} `json:"id"`
}

type deviceKey struct {
	name     string
	version  string
	deviceID string
}

func formatPCIID(id *int64) string {
	if id == nil || *id < 0 {
		return ""
	}
	if *id > math.MaxUint16 {
		// Not a PCI device ID. Render it in full rather than truncating, so a
		// malformed value cannot collide with a real one. NormalizeDeviceID
		// bounds the drivers config to 16 bits, so it matches no entry there.
		return fmt.Sprintf("%x", *id)
	}
	return metrics.FormatDeviceID(uint16(*id))
}
