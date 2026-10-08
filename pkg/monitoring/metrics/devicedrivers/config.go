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
)

const SupportedVersion = "v1alpha1"

type DriverEntry struct {
	DeviceID         string
	GuestOSVersionID string
	DriverVersion    string
}

type driversConfig struct {
	Version string         `json:"version"`
	Devices []deviceConfig `json:"devices"`
}

type deviceConfig struct {
	DeviceID string          `json:"deviceID"`
	Current  []driverVersion `json:"current"`
}

type driverVersion struct {
	GuestOSVersionID string `json:"guestOSVersionID"`
	DriverVersion    string `json:"driverVersion"`
}

func Parse(data []byte) ([]DriverEntry, error) {
	var cfg driversConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
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

	for _, device := range cfg.Devices {
		for _, version := range device.Current {
			entry := DriverEntry{
				DeviceID:         device.DeviceID,
				GuestOSVersionID: version.GuestOSVersionID,
				DriverVersion:    version.DriverVersion,
			}

			if entry.DeviceID == "" ||
				entry.GuestOSVersionID == "" ||
				entry.DriverVersion == "" {
				continue
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
