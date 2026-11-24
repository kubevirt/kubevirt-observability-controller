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
	It("should return one entry per device and guest OS version, carrying the build number", func() {
		entries, err := Parse([]byte(`
version: v1alpha2
devices:
  - deviceID: "1041"
    deviceName: "VirtIO network device"
    minimum:
      - guestOSVersionID: "2022"
        driverVersion: "100.95.104.26200"
      - guestOSVersionID: "2019"
        driverVersion: "100.94.104.25200"
`))

		Expect(err).ToNot(HaveOccurred())
		Expect(entries).To(ConsistOf(
			DriverEntry{
				DeviceID: "1041", DeviceName: "VirtIO network device",
				GuestOSVersionID: "2022", DriverVersion: "100.95.104.26200", BuildNumber: 26200,
			},
			DriverEntry{
				DeviceID: "1041", DeviceName: "VirtIO network device",
				GuestOSVersionID: "2019", DriverVersion: "100.94.104.25200", BuildNumber: 25200,
			},
		))
	})

	It("should keep entries whose device name is missing", func() {
		entries, err := Parse([]byte(`
version: v1alpha2
devices:
  - deviceID: "1041"
    minimum:
      - guestOSVersionID: "2022"
        driverVersion: "100.95.104.26200"
`))

		Expect(err).ToNot(HaveOccurred())
		Expect(entries).To(ConsistOf(
			DriverEntry{
				DeviceID: "1041", GuestOSVersionID: "2022",
				DriverVersion: "100.95.104.26200", BuildNumber: 26200,
			},
		))
	})

	It("should normalize device IDs to the form the device metric labels them with", func() {
		entries, err := Parse([]byte(`
version: v1alpha2
devices:
  - deviceID: "0x104a"
    minimum:
      - guestOSVersionID: "2022"
        driverVersion: "100.95.104.26200"
`))

		Expect(err).ToNot(HaveOccurred())
		Expect(entries).To(ConsistOf(
			DriverEntry{
				DeviceID: "104a", GuestOSVersionID: "2022",
				DriverVersion: "100.95.104.26200", BuildNumber: 26200,
			},
		))
	})

	It("should reject a config whose device ID is not a device ID", func() {
		_, err := Parse([]byte(`
version: v1alpha2
devices:
  - deviceID: "1041"
    minimum:
      - guestOSVersionID: "2022"
        driverVersion: "100.95.104.26200"
  - deviceID: "virtio-net"
    minimum:
      - guestOSVersionID: "2022"
        driverVersion: "100.95.104.26200"
`))

		Expect(err).To(MatchError(ContainSubstring("virtio-net")))
	})

	It("should reject a config whose version is not v1alpha2", func() {
		_, err := Parse([]byte(`
version: v2
devices:
  - deviceID: "1041"
    minimum:
      - guestOSVersionID: "2022"
        driverVersion: "100.95.104.26200"
`))

		Expect(err).To(MatchError(ContainSubstring("v2")))
	})

	It("should reject a config whose device ID is missing", func() {
		_, err := Parse([]byte(`
version: v1alpha2
devices:
  - deviceID: ""
    minimum:
      - guestOSVersionID: "2022"
        driverVersion: "100.95.104.26200"
`))

		Expect(err).To(MatchError(ContainSubstring("device 0")))
	})

	It("should reject a config whose guest OS version is missing", func() {
		_, err := Parse([]byte(`
version: v1alpha2
devices:
  - deviceID: "1041"
    minimum:
      - guestOSVersionID: ""
        driverVersion: "100.95.104.26200"
`))

		Expect(err).To(MatchError(ContainSubstring("guestOSVersionID")))
	})

	It("should reject a config whose driver version is missing", func() {
		_, err := Parse([]byte(`
version: v1alpha2
devices:
  - deviceID: "1041"
    minimum:
      - guestOSVersionID: "2022"
        driverVersion: ""
`))

		Expect(err).To(MatchError(ContainSubstring("driverVersion")))
	})

	It("should trim surrounding whitespace off the values it labels series with", func() {
		entries, err := Parse([]byte(`
version: v1alpha2
devices:
  - deviceID: "1041"
    minimum:
      - guestOSVersionID: " 2022 "
        driverVersion: " 100.95.104.26200 "
`))

		Expect(err).ToNot(HaveOccurred())
		Expect(entries).To(ConsistOf(
			DriverEntry{
				DeviceID: "1041", GuestOSVersionID: "2022",
				DriverVersion: "100.95.104.26200", BuildNumber: 26200,
			},
		))
	})

	It("should reject a driver version that is only whitespace", func() {
		_, err := Parse([]byte(`
version: v1alpha2
devices:
  - deviceID: "1041"
    minimum:
      - guestOSVersionID: "2022"
        driverVersion: "   "
`))

		Expect(err).To(MatchError(ContainSubstring("driverVersion")))
	})

	It("should reject a device that declares no minimum driver versions", func() {
		_, err := Parse([]byte(`
version: v1alpha2
devices:
  - deviceID: "1041"
`))

		Expect(err).To(MatchError(ContainSubstring("no minimum driver versions")))
	})

	It("should reject a driver version with no build number", func() {
		_, err := Parse([]byte(`
version: v1alpha2
devices:
  - deviceID: "1041"
    minimum:
      - guestOSVersionID: "2022"
        driverVersion: "100.95.104.70000"
`))

		Expect(err).To(MatchError(ContainSubstring("100.95.104.70000")))
	})

	It("should reject a config with a misspelled field", func() {
		_, err := Parse([]byte(`
version: v1alpha2
devices:
  - deviceID: "1041"
    minnimum:
      - guestOSVersionID: "2022"
        driverVersion: "100.95.104.26200"
`))

		Expect(err).To(MatchError(ContainSubstring("minnimum")))
	})

	It("should reject a config with a duplicated field", func() {
		_, err := Parse([]byte(`
version: v1alpha2
version: v1alpha2
devices: []
`))

		Expect(err).To(HaveOccurred())
	})

	It("should reject a second minimum for the same device and guest OS version", func() {
		_, err := Parse([]byte(`
version: v1alpha2
devices:
  - deviceID: "1041"
    minimum:
      - guestOSVersionID: "2022"
        driverVersion: "100.95.104.26200"
      - guestOSVersionID: "2022"
        driverVersion: "100.95.104.26100"
`))

		Expect(err).To(MatchError(ContainSubstring("more than one minimum")))
	})

	It("should return an error for malformed YAML", func() {
		_, err := Parse([]byte("version: v1alpha2\ndevices: [oops"))

		Expect(err).To(HaveOccurred())
	})
})
