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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k6tv1 "kubevirt.io/api/core/v1"
)

func vmiWithOSID(id string) *k6tv1.VirtualMachineInstance {
	return &k6tv1.VirtualMachineInstance{
		Status: k6tv1.VirtualMachineInstanceStatus{
			GuestOSInfo: k6tv1.VirtualMachineInstanceGuestOSInfo{ID: id},
		},
	}
}

var _ = Describe("resolveVMStatsRequest", func() {
	It("requests only libvirt categories when no guest OS info is reported", func() {
		req := NewRequestPolicy(nil).Resolve(vmiWithOSID(""), "")

		Expect(req.DomainStats).ToNot(BeNil())
		Expect(req.DirtyRate).ToNot(BeNil())

		// No guest agent data available, so nothing guest-related is requested.
		Expect(req.GuestGetHostName).To(BeNil())
		Expect(req.GuestGetTimezone).To(BeNil())
		Expect(req.GuestGetUsers).To(BeNil())
		Expect(req.GuestGetFsInfo).To(BeNil())
		Expect(req.GuestGetDevices).To(BeNil())
	})

	It("never requests categories the VMI status already answers", func() {
		for _, id := range []string{"", "rhel", "mswindows"} {
			req := NewRequestPolicy(nil).Resolve(vmiWithOSID(id), "8.2.0")

			Expect(req.GuestGetOsInfo).To(BeNil(), "os info comes from status.guestOSInfo")
			Expect(req.GuestNetworkGetInterfaces).To(BeNil(), "interfaces come from status.interfaces")
		}
	})

	It("adds guest-common and Windows categories for mswindows guests", func() {
		req := NewRequestPolicy(nil).Resolve(vmiWithOSID("mswindows"), "5.2.0")

		Expect(req.DomainStats).ToNot(BeNil())
		Expect(req.DirtyRate).ToNot(BeNil())
		Expect(req.GuestGetHostName).ToNot(BeNil())
		Expect(req.GuestGetTimezone).ToNot(BeNil())
		// Filesystem info is guest-common (every guest has filesystems).
		Expect(req.GuestGetFsInfo).ToNot(BeNil())

		Expect(req.GuestGetUsers).ToNot(BeNil())
		Expect(req.GuestGetDevices).ToNot(BeNil())
	})

	It("adds only guest-common categories for a known Linux distro", func() {
		req := NewRequestPolicy(nil).Resolve(vmiWithOSID("rhel"), "8.2.0")

		Expect(req.DomainStats).ToNot(BeNil())
		Expect(req.DirtyRate).ToNot(BeNil())
		Expect(req.GuestGetHostName).ToNot(BeNil())
		Expect(req.GuestGetTimezone).ToNot(BeNil())
		Expect(req.GuestGetFsInfo).ToNot(BeNil())

		// Users and devices are Windows-family extras, not requested for Linux.
		Expect(req.GuestGetUsers).To(BeNil())
		Expect(req.GuestGetDevices).To(BeNil())
	})

	It("treats any non-empty, non-Windows ID as Linux", func() {
		req := NewRequestPolicy(nil).Resolve(vmiWithOSID("some-new-distro"), "8.2.0")

		Expect(req.GuestGetFsInfo).ToNot(BeNil())
		Expect(req.GuestGetUsers).To(BeNil())
		Expect(req.GuestGetDevices).To(BeNil())
	})

	DescribeTable("gates guest-get-devices on the reported guest agent version",
		func(agentVer string, requested bool) {
			req := NewRequestPolicy(nil).Resolve(vmiWithOSID("mswindows"), agentVer)

			if requested {
				Expect(req.GuestGetDevices).ToNot(BeNil())
			} else {
				Expect(req.GuestGetDevices).To(BeNil())
			}
			// The rest of the Windows family is version independent.
			Expect(req.GuestGetUsers).ToNot(BeNil())
		},
		Entry("no poll has reported a version yet", "", false),
		Entry("below the minimum", "5.1.0", false),
		Entry("at the minimum", "5.2.0", true),
		Entry("above the minimum", "8.2.0", true),
		Entry("major bump above the minimum", "10.0.0", true),
		Entry("major below the minimum", "4.9.9", false),
		Entry("distro suffixed", "7.2.0-1.el9", true),
		Entry("major.minor only", "6.2", true),
		Entry("unparseable", "unknown", false),
	)

	It("classifies the OS ID case-insensitively", func() {
		req := NewRequestPolicy(nil).Resolve(vmiWithOSID("MSWindows"), "8.2.0")

		Expect(req.GuestGetUsers).ToNot(BeNil())
		Expect(req.GuestGetDevices).ToNot(BeNil())
	})
})

var _ = Describe("RequestPolicy.Resolve under a restricted allowlist", func() {
	deviceOnly := func() *RequestPolicy {
		return NewRequestPolicy(map[string]bool{
			"kubevirt_vmi_guest_device_driver_timestamp_seconds": true,
		})
	}

	It("requests only guest-get-devices once the agent version is known", func() {
		req := deviceOnly().Resolve(vmiWithOSID("mswindows"), "5.2.0")

		Expect(req).ToNot(BeNil())
		Expect(req.GuestGetDevices).ToNot(BeNil())
		Expect(req.DomainStats).To(BeNil())
		Expect(req.DirtyRate).To(BeNil())
		Expect(req.GuestGetOsInfo).To(BeNil())
		Expect(req.GuestGetHostName).To(BeNil())
		Expect(req.GuestGetTimezone).To(BeNil())
		Expect(req.GuestGetUsers).To(BeNil())
		Expect(req.GuestGetFsInfo).To(BeNil())
		Expect(req.GuestNetworkGetInterfaces).To(BeNil())
	})

	It("bootstraps with guest-get-hostname when no poll has reported a version yet", func() {
		req := deviceOnly().Resolve(vmiWithOSID("mswindows"), "")

		Expect(req).ToNot(BeNil())
		Expect(req.GuestGetHostName).ToNot(BeNil(),
			"the reply has to carry GuestAgentVersion before devices can be gated")
		Expect(req.GuestGetDevices).To(BeNil())
		Expect(req.DomainStats).To(BeNil())
	})

	It("does not bootstrap when the reported version is below the minimum", func() {
		Expect(deviceOnly().Resolve(vmiWithOSID("mswindows"), "5.1.0")).To(BeNil())
	})

	It("does not bootstrap when the reported version is unparseable", func() {
		Expect(deviceOnly().Resolve(vmiWithOSID("mswindows"), "unknown")).To(BeNil())
	})

	It("requests nothing for a Linux guest", func() {
		Expect(deviceOnly().Resolve(vmiWithOSID("rhel"), "8.2.0")).To(BeNil())
	})

	It("requests nothing for a guest that has not reported an OS", func() {
		Expect(deviceOnly().Resolve(vmiWithOSID(""), "")).To(BeNil())
	})

	It("requests nothing at all when the allowlist keeps no vmstats metric", func() {
		policy := NewRequestPolicy(map[string]bool{})

		Expect(policy.Resolve(vmiWithOSID("mswindows"), "8.2.0")).To(BeNil())
		Expect(policy.Resolve(vmiWithOSID("rhel"), "8.2.0")).To(BeNil())
	})

	It("keeps the libvirt categories for a domainStats-only allowlist", func() {
		req := NewRequestPolicy(map[string]bool{"kubevirt_vmi_memory_resident_bytes": true}).
			Resolve(vmiWithOSID("rhel"), "8.2.0")

		Expect(req).ToNot(BeNil())
		Expect(req.DomainStats).ToNot(BeNil())
		Expect(req.DirtyRate).To(BeNil())
		Expect(req.GuestGetOsInfo).To(BeNil())
		Expect(req.GuestGetFsInfo).To(BeNil())
		Expect(req.GuestNetworkGetInterfaces).To(BeNil())
	})

	It("keeps a guest-common category without dragging in its siblings", func() {
		req := NewRequestPolicy(map[string]bool{"kubevirt_vmi_guest_hostname": true}).
			Resolve(vmiWithOSID("rhel"), "8.2.0")

		Expect(req).ToNot(BeNil())
		Expect(req.GuestGetHostName).ToNot(BeNil())
		Expect(req.GuestGetOsInfo).To(BeNil())
		Expect(req.GuestGetTimezone).To(BeNil())
		Expect(req.DomainStats).To(BeNil())
	})
})

var _ = Describe("NewRequestPolicy", func() {
	It("allows every category when the allowlist is nil", func() {
		policy := NewRequestPolicy(nil)

		for _, c := range allCategories {
			Expect(policy.allows(c)).To(BeTrue(), string(c))
		}
		Expect(policy.Enabled()).To(BeTrue())
	})

	It("allows no category when the allowlist is empty", func() {
		policy := NewRequestPolicy(map[string]bool{})

		for _, c := range allCategories {
			Expect(policy.allows(c)).To(BeFalse(), string(c))
		}
		Expect(policy.Enabled()).To(BeFalse())
	})

	It("allows only guest-get-devices for a device-driver-only allowlist", func() {
		policy := NewRequestPolicy(map[string]bool{
			"kubevirt_vmi_guest_device_driver_timestamp_seconds": true,
		})

		Expect(policy.allows(catGuestGetDevices)).To(BeTrue())
		Expect(policy.Enabled()).To(BeTrue())
		for _, c := range allCategories {
			if c == catGuestGetDevices {
				continue
			}
			Expect(policy.allows(c)).To(BeFalse(), string(c))
		}
	})

	It("allows domainStats for any metric sourced from it", func() {
		policy := NewRequestPolicy(map[string]bool{"kubevirt_vmi_memory_resident_bytes": true})

		Expect(policy.allows(catDomainStats)).To(BeTrue())
		Expect(policy.allows(catDirtyRate)).To(BeFalse())
	})

	It("allows both libvirt categories for the dirty-rate metric", func() {
		policy := NewRequestPolicy(map[string]bool{"kubevirt_vmi_dirty_rate_bytes_per_second": true})

		Expect(policy.allows(catDirtyRate)).To(BeTrue())
		Expect(policy.allows(catDomainStats)).To(BeTrue())
	})

	It("ignores metric names that no category produces", func() {
		policy := NewRequestPolicy(map[string]bool{"kubevirt_vmi_info": true})

		Expect(policy.Enabled()).To(BeFalse())
	})

	It("maps every registered vmstats metric to at least one category", func() {
		for _, m := range vmStatsCollector(nil).Metrics {
			name := m.GetOpts().Name
			Expect(NewRequestPolicy(map[string]bool{name: true}).Enabled()).To(BeTrue(), name)
		}
	})
})

var _ = Describe("buildRequest", func() {
	It("maps every known category to a VMStatsRequest field", func() {
		set := make(map[category]bool)
		for _, c := range allCategories {
			set[c] = true
		}
		req := buildRequest(set)

		Expect(req.DomainStats).ToNot(BeNil())
		Expect(req.DirtyRate).ToNot(BeNil())
		Expect(req.GuestGetHostName).ToNot(BeNil())
		Expect(req.GuestGetTimezone).ToNot(BeNil())
		Expect(req.GuestGetUsers).ToNot(BeNil())
		Expect(req.GuestGetFsInfo).ToNot(BeNil())
		Expect(req.GuestGetDevices).ToNot(BeNil())
	})
})
