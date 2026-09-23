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
	"fmt"

	"sigs.k8s.io/yaml"

	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/metrics"
)

const SupportedVersion = "v1alpha1"

type DriverEntry struct {
	DeviceID         string
	DeviceName       string
	GuestOSVersionID string
	DriverVersion    string
}

type driversConfig struct {
	Version string         `json:"version"`
	Devices []deviceConfig `json:"devices"`
}

type deviceConfig struct {
	DeviceID   string          `json:"deviceID"`
	DeviceName string          `json:"deviceName"`
	Current    []driverVersion `json:"current"`
}

type driverVersion struct {
	GuestOSVersionID string `json:"guestOSVersionID"`
	DriverVersion    string `json:"driverVersion"`
}

func Parse(data []byte) ([]DriverEntry, error) {
	var cfg driversConfig
	if err := yaml.UnmarshalStrict(data, &cfg); err != nil {
		return nil, err
	}

	if cfg.Version != SupportedVersion {
		return nil, fmt.Errorf(
			"unsupported drivers config version %q, expected %q",
			cfg.Version, SupportedVersion,
		)
	}

	var entries []DriverEntry
	seen := make(map[DriverEntry]bool)

	for i, device := range cfg.Devices {
		deviceID, err := metrics.NormalizeDeviceID(device.DeviceID)
		if err != nil {
			return nil, fmt.Errorf("device %d in drivers config: %w", i, err)
		}

		if len(device.Current) == 0 {
			return nil, fmt.Errorf("device %q in drivers config declares no current driver versions", deviceID)
		}

		for _, version := range device.Current {
			if version.GuestOSVersionID == "" {
				return nil, fmt.Errorf("device %q in drivers config has a version without a guestOSVersionID", deviceID)
			}

			if version.DriverVersion == "" {
				return nil, fmt.Errorf(
					"device %q, guest OS version %q in drivers config has no driverVersion",
					deviceID, version.GuestOSVersionID,
				)
			}

			entry := DriverEntry{
				DeviceID:         deviceID,
				DeviceName:       device.DeviceName,
				GuestOSVersionID: version.GuestOSVersionID,
				DriverVersion:    version.DriverVersion,
			}

			if seen[entry] {
				continue
			}
			seen[entry] = true

			entries = append(entries, entry)
		}
	}

	return entries, nil
}
