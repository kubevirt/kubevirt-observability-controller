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
	"strings"

	k6tv1 "kubevirt.io/api/core/v1"
)

type category string

const (
	catDomainStats               category = "domainStats"
	catDirtyRate                 category = "dirtyRate"
	catGuestGetOsInfo            category = "guestGetOsInfo"
	catGuestGetHostName          category = "guestGetHostName"
	catGuestGetTimezone          category = "guestGetTimezone"
	catGuestGetUsers             category = "guestGetUsers"
	catGuestGetFsInfo            category = "guestGetFsInfo"
	catGuestNetworkGetInterfaces category = "guestNetworkGetInterfaces"
)

var allCategories = []category{
	catDomainStats,
	catDirtyRate,
	catGuestGetOsInfo,
	catGuestGetHostName,
	catGuestGetTimezone,
	catGuestGetUsers,
	catGuestGetFsInfo,
	catGuestNetworkGetInterfaces,
}

var libvirtCategories = []category{
	catDomainStats,
	catDirtyRate,
}

var guestCommonCategories = []category{
	catGuestGetOsInfo,
	catGuestGetHostName,
	catGuestGetTimezone,
	catGuestGetFsInfo,
}

var osFamilyCategories = map[string][]category{
	"windows": {catGuestGetUsers},
	"linux":   {catGuestNetworkGetInterfaces},
}

// resolveVMStatsRequest decides which stats categories to request for a single
// VMI. libvirt categories are always requested, guest-agent categories are
// added only when the guest agent has reported an OS, OS-family specific are
// added based on that returned value.
func resolveVMStatsRequest(vmi *k6tv1.VirtualMachineInstance) *VMStatsRequest {
	set := make(map[category]bool)
	addCategories(set, libvirtCategories)

	id := strings.ToLower(strings.TrimSpace(vmi.Status.GuestOSInfo.ID))
	if id == "" {
		// No guest agent data available; nothing guest-related to collect.
		return buildRequest(set)
	}

	addCategories(set, guestCommonCategories)
	addCategories(set, osFamilyCategories[osFamily(id)])

	return buildRequest(set)
}

func osFamily(id string) string {
	if id == "mswindows" {
		return "windows"
	}
	return "linux"
}

func addCategories(set map[category]bool, cats []category) {
	for _, c := range cats {
		set[c] = true
	}
}

func buildRequest(set map[category]bool) *VMStatsRequest {
	req := &VMStatsRequest{}
	for c := range set {
		switch c {
		case catDomainStats:
			req.DomainStats = enabled
		case catDirtyRate:
			req.DirtyRate = enabled
		case catGuestGetOsInfo:
			req.GuestGetOsInfo = enabled
		case catGuestGetHostName:
			req.GuestGetHostName = enabled
		case catGuestGetTimezone:
			req.GuestGetTimezone = enabled
		case catGuestGetUsers:
			req.GuestGetUsers = enabled
		case catGuestGetFsInfo:
			req.GuestGetFsInfo = enabled
		case catGuestNetworkGetInterfaces:
			req.GuestNetworkGetInterfaces = enabled
		}
	}
	return req
}
