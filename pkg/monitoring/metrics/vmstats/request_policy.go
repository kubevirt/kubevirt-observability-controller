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
	"regexp"
	"strconv"
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
	catGuestGetDevices           category = "guestGetDevices"
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
	catGuestGetDevices,
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

const windowsFamily = "windows"

var osFamilyCategories = map[string][]category{
	windowsFamily: {catGuestGetUsers},
	"linux":       {catGuestNetworkGetInterfaces},
}

var guestGetDevicesMinAgent = agentVersion{major: 5, minor: 2, known: true}

// resolveVMStatsRequest decides which stats categories to request for a single
// VMI. libvirt categories are always requested, guest-agent categories are
// added only when the guest agent has reported an OS, OS-family specific are
// added based on that returned value.
func resolveVMStatsRequest(vmi *k6tv1.VirtualMachineInstance, agentVer string) *VMStatsRequest {
	set := make(map[category]bool)
	addCategories(set, libvirtCategories)

	id := strings.ToLower(strings.TrimSpace(vmi.Status.GuestOSInfo.ID))
	if id == "" {
		// No guest agent data available; nothing guest-related to collect.
		return buildRequest(set)
	}

	family := osFamily(id)
	addCategories(set, guestCommonCategories)
	addCategories(set, osFamilyCategories[family])

	if family == windowsFamily && parseAgentVersion(agentVer).atLeast(guestGetDevicesMinAgent) {
		set[catGuestGetDevices] = true
	}

	return buildRequest(set)
}

type agentVersion struct {
	major, minor int
	known        bool
}

var agentVersionPattern = regexp.MustCompile(`^(\d+)\.(\d+)`)

// parseAgentVersion reads the leading major.minor out of a guest agent version
// string, which carries distro suffixes such as "7.2.0-1.el9".
func parseAgentVersion(s string) agentVersion {
	match := agentVersionPattern.FindStringSubmatch(strings.TrimSpace(s))
	if match == nil {
		return agentVersion{}
	}

	major, err := strconv.Atoi(match[1])
	if err != nil {
		return agentVersion{}
	}
	minor, err := strconv.Atoi(match[2])
	if err != nil {
		return agentVersion{}
	}

	return agentVersion{major: major, minor: minor, known: true}
}

func (v agentVersion) atLeast(min agentVersion) bool {
	if !v.known {
		return false
	}
	if v.major != min.major {
		return v.major > min.major
	}
	return v.minor >= min.minor
}

func osFamily(id string) string {
	if id == "mswindows" {
		return windowsFamily
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
		case catGuestGetDevices:
			req.GuestGetDevices = enabled
		}
	}
	return req
}
