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
		req := resolveVMStatsRequest(vmiWithOSID(""))

		Expect(req.DomainStats).ToNot(BeNil())
		Expect(req.DirtyRate).ToNot(BeNil())

		// No guest agent data available, so nothing guest-related is requested.
		Expect(req.GuestGetOsInfo).To(BeNil())
		Expect(req.GuestGetHostName).To(BeNil())
		Expect(req.GuestGetTimezone).To(BeNil())
		Expect(req.GuestGetUsers).To(BeNil())
		Expect(req.GuestGetFsInfo).To(BeNil())
		Expect(req.GuestNetworkGetInterfaces).To(BeNil())
	})

	It("adds guest-common and Windows categories for mswindows guests", func() {
		req := resolveVMStatsRequest(vmiWithOSID("mswindows"))

		Expect(req.DomainStats).ToNot(BeNil())
		Expect(req.DirtyRate).ToNot(BeNil())
		Expect(req.GuestGetOsInfo).ToNot(BeNil())
		Expect(req.GuestGetHostName).ToNot(BeNil())
		Expect(req.GuestGetTimezone).ToNot(BeNil())
		// Filesystem info is guest-common (every guest has filesystems).
		Expect(req.GuestGetFsInfo).ToNot(BeNil())

		Expect(req.GuestGetUsers).ToNot(BeNil())

		// Network interfaces are a Linux-family extra, not requested for Windows.
		Expect(req.GuestNetworkGetInterfaces).To(BeNil())
	})

	It("adds guest-common and Linux categories for a known Linux distro", func() {
		req := resolveVMStatsRequest(vmiWithOSID("rhel"))

		Expect(req.DomainStats).ToNot(BeNil())
		Expect(req.DirtyRate).ToNot(BeNil())
		Expect(req.GuestGetOsInfo).ToNot(BeNil())
		Expect(req.GuestGetHostName).ToNot(BeNil())
		Expect(req.GuestGetTimezone).ToNot(BeNil())
		Expect(req.GuestGetFsInfo).ToNot(BeNil())

		Expect(req.GuestNetworkGetInterfaces).ToNot(BeNil())

		// Users is a Windows-family extra, not requested for Linux.
		Expect(req.GuestGetUsers).To(BeNil())
	})

	It("treats any non-empty, non-Windows ID as Linux", func() {
		req := resolveVMStatsRequest(vmiWithOSID("some-new-distro"))

		Expect(req.GuestGetOsInfo).ToNot(BeNil())
		Expect(req.GuestGetFsInfo).ToNot(BeNil())
		Expect(req.GuestNetworkGetInterfaces).ToNot(BeNil())
		Expect(req.GuestGetUsers).To(BeNil())
	})

	It("classifies the OS ID case-insensitively", func() {
		req := resolveVMStatsRequest(vmiWithOSID("MSWindows"))

		Expect(req.GuestGetUsers).ToNot(BeNil())
		Expect(req.GuestNetworkGetInterfaces).To(BeNil())
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
		Expect(req.GuestGetOsInfo).ToNot(BeNil())
		Expect(req.GuestGetHostName).ToNot(BeNil())
		Expect(req.GuestGetTimezone).ToNot(BeNil())
		Expect(req.GuestGetUsers).ToNot(BeNil())
		Expect(req.GuestGetFsInfo).ToNot(BeNil())
		Expect(req.GuestNetworkGetInterfaces).ToNot(BeNil())
	})
})
