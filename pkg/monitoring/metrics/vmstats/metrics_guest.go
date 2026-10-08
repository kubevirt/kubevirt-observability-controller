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

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
)

var (
	guestMetricsList = []operatormetrics.Metric{
		guestOsInfo, guestHostname, guestTimezone,
		guestUserCount, filesystemCapacityBytes, filesystemUsedBytes,
		guestInterfaceInfo,
	}

	guestOsInfo = operatormetrics.NewGauge(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_guest_os_info",
		Help: "Guest OS information from the guest agent.",
	})
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
	guestInterfaceInfo = operatormetrics.NewGauge(operatormetrics.MetricOpts{
		Name: "kubevirt_vmi_guest_interface_info",
		Help: "Guest network interface information from the guest agent.",
	})
)

func collectGuestMetrics(report *VMIReport) []operatormetrics.CollectorResult {
	var crs []operatormetrics.CollectorResult //nolint:prealloc
	crs = append(crs, collectGuestOsInfo(report)...)
	crs = append(crs, collectGuestHostname(report)...)
	crs = append(crs, collectGuestTimezone(report)...)
	crs = append(crs, collectGuestUsers(report)...)
	crs = append(crs, collectGuestFilesystems(report)...)
	crs = append(crs, collectGuestInterfaces(report)...)
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

func collectGuestOsInfo(report *VMIReport) []operatormetrics.CollectorResult {
	raw := guestReturn(report.Stats.GuestGetOsInfo)
	if raw == nil {
		return nil
	}
	var info struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		Version       string `json:"version"`
		KernelRelease string `json:"kernel-release"`
		Machine       string `json:"machine"`
	}
	if err := json.Unmarshal(raw, &info); err != nil {
		return nil
	}
	return []operatormetrics.CollectorResult{
		report.newCollectorResultWithLabels(guestOsInfo, 1.0, map[string]string{
			"os_name":        info.Name,
			"os_version":     info.Version,
			"os_id":          info.ID,
			"kernel_release": info.KernelRelease,
			"machine":        info.Machine,
		}),
	}
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

func collectGuestInterfaces(report *VMIReport) []operatormetrics.CollectorResult {
	raw := guestReturn(report.Stats.GuestNetworkGetInterfaces)
	if raw == nil {
		return nil
	}
	var ifaces []struct {
		Name        string `json:"name"`
		MAC         string `json:"hardware-address"`
		IPAddresses []struct {
			Address string `json:"ip-address"`
		} `json:"ip-addresses"`
	}
	if err := json.Unmarshal(raw, &ifaces); err != nil {
		return nil
	}
	var crs []operatormetrics.CollectorResult
	for _, iface := range ifaces {
		ipAddr := ""
		if len(iface.IPAddresses) > 0 {
			ipAddr = iface.IPAddresses[0].Address
		}
		crs = append(crs, report.newCollectorResultWithLabels(guestInterfaceInfo, 1.0, map[string]string{
			"interface_name": iface.Name,
			"mac":            iface.MAC,
			"ip_address":     ipAddr,
		}))
	}
	return crs
}
