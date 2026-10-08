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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Parse", func() {
	It("should return one entry per device, guest OS version and driver version", func() {
		entries, err := Parse([]byte(`
version: v1alpha1
devices:
  - deviceID: "1041"
    deviceName: "VirtIO network device"
    current:
      - guestOSVersionID: "2022"
        driverVersion: "100.95.104.26200"
      - guestOSVersionID: "2022"
        driverVersion: "100.95.104.26100"
      - guestOSVersionID: "2019"
        driverVersion: "100.94.104.25200"
`))

		Expect(err).ToNot(HaveOccurred())
		Expect(entries).To(ConsistOf(
			DriverEntry{DeviceID: "1041", GuestOSVersionID: "2022", DriverVersion: "100.95.104.26200"},
			DriverEntry{DeviceID: "1041", GuestOSVersionID: "2022", DriverVersion: "100.95.104.26100"},
			DriverEntry{DeviceID: "1041", GuestOSVersionID: "2019", DriverVersion: "100.94.104.25200"},
		))
	})

	It("should reject a config whose version is not v1alpha1", func() {
		_, err := Parse([]byte(`
version: v2
devices:
  - deviceID: "1041"
    current:
      - guestOSVersionID: "2022"
        driverVersion: "100.95.104.26200"
`))

		Expect(err).To(MatchError(ContainSubstring("v2")))
	})

	It("should skip entries missing a device ID, guest OS version or driver version", func() {
		entries, err := Parse([]byte(`
version: v1alpha1
devices:
  - deviceID: ""
    current:
      - guestOSVersionID: "2022"
        driverVersion: "100.95.104.26200"
  - deviceID: "1041"
    current:
      - guestOSVersionID: ""
        driverVersion: "100.95.104.26200"
      - guestOSVersionID: "2022"
        driverVersion: ""
      - guestOSVersionID: "2019"
        driverVersion: "100.94.104.25200"
`))

		Expect(err).ToNot(HaveOccurred())
		Expect(entries).To(ConsistOf(
			DriverEntry{DeviceID: "1041", GuestOSVersionID: "2019", DriverVersion: "100.94.104.25200"},
		))
	})

	It("should collapse duplicate tuples into a single entry", func() {
		entries, err := Parse([]byte(`
version: v1alpha1
devices:
  - deviceID: "1041"
    current:
      - guestOSVersionID: "2022"
        driverVersion: "100.95.104.26200"
      - guestOSVersionID: "2022"
        driverVersion: "100.95.104.26200"
`))

		Expect(err).ToNot(HaveOccurred())
		Expect(entries).To(ConsistOf(
			DriverEntry{DeviceID: "1041", GuestOSVersionID: "2022", DriverVersion: "100.95.104.26200"},
		))
	})

	It("should return an error for malformed YAML", func() {
		_, err := Parse([]byte("version: v1alpha1\ndevices: [oops"))

		Expect(err).To(HaveOccurred())
	})
})
