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
	"slices"
	"strconv"
	"strings"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	k6tv1 "kubevirt.io/api/core/v1"
)

type category string

const (
	catDomainStats      category = "domainStats"
	catDirtyRate        category = "dirtyRate"
	catGuestGetHostName category = "guestGetHostName"
	catGuestGetTimezone category = "guestGetTimezone"
	catGuestGetUsers    category = "guestGetUsers"
	catGuestGetFsInfo   category = "guestGetFsInfo"
	catGuestGetDevices  category = "guestGetDevices"
)

var allCategories = []category{
	catDomainStats,
	catDirtyRate,
	catGuestGetHostName,
	catGuestGetTimezone,
	catGuestGetUsers,
	catGuestGetFsInfo,
	catGuestGetDevices,
}

var libvirtCategories = []category{
	catDomainStats,
	catDirtyRate,
}

var guestCommonCategories = []category{
	catGuestGetHostName,
	catGuestGetTimezone,
	catGuestGetFsInfo,
}

const windowsFamily = "windows"

var osFamilyCategories = map[string][]category{
	windowsFamily: {catGuestGetUsers},
}

var guestGetDevicesMinAgent = agentVersion{major: 5, minor: 2, known: true}

var categoryMetrics = map[category][]operatormetrics.Metric{
	catDomainStats: slices.Concat(
		cpuMetricsList, memoryMetricsList, vcpuMetricsList,
		networkMetricsList, blockMetricsList, dirtyRateMetricsList,
	),
	catDirtyRate:        dirtyRateMetricsList,
	catGuestGetHostName: {guestHostname},
	catGuestGetTimezone: {guestTimezone},
	catGuestGetUsers:    {guestUserCount},
	catGuestGetFsInfo:   {filesystemCapacityBytes, filesystemUsedBytes},
	catGuestGetDevices:  {guestDeviceDriverTimestamp},
}

type RequestPolicy struct {
	allowed map[category]bool
}

func NewRequestPolicy(allowlist map[string]bool) *RequestPolicy {
	if allowlist == nil {
		return &RequestPolicy{}
	}

	allowed := make(map[category]bool)
	for cat, metrics := range categoryMetrics {
		for _, m := range metrics {
			if allowlist[m.GetOpts().Name] {
				allowed[cat] = true
				break
			}
		}
	}
	return &RequestPolicy{allowed: allowed}
}

func (p *RequestPolicy) allows(c category) bool {
	if p.allowed == nil {
		return true
	}
	return p.allowed[c]
}

func (p *RequestPolicy) Enabled() bool {
	return p.allowed == nil || len(p.allowed) > 0
}

func (p *RequestPolicy) Resolve(vmi *k6tv1.VirtualMachineInstance, agentVer string) *VMStatsRequest {
	set := make(map[category]bool)
	p.addCategories(set, libvirtCategories)

	id := strings.ToLower(strings.TrimSpace(vmi.Status.GuestOSInfo.ID))
	if id == "" {
		// No guest agent data available; nothing guest-related to collect.
		return buildRequest(set)
	}

	family := osFamily(id)
	p.addCategories(set, guestCommonCategories)
	p.addCategories(set, osFamilyCategories[family])

	if family == windowsFamily && p.allows(catGuestGetDevices) {
		switch version := parseAgentVersion(agentVer); {
		case version.atLeast(guestGetDevicesMinAgent):
			set[catGuestGetDevices] = true
		case agentVer == "":
			// Bootstrap: ask for the cheapest guest-agent command so the reply
			// carries GuestAgentVersion, which gates guest-get-devices. The
			// agent version is not in the VMI status, so it takes a poll.
			set[catGuestGetHostName] = true
		}
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

func (p *RequestPolicy) addCategories(set map[category]bool, cats []category) {
	for _, c := range cats {
		if p.allows(c) {
			set[c] = true
		}
	}
}

func buildRequest(set map[category]bool) *VMStatsRequest {
	if len(set) == 0 {
		return nil
	}

	req := &VMStatsRequest{}
	for c := range set {
		switch c {
		case catDomainStats:
			req.DomainStats = enabled
		case catDirtyRate:
			req.DirtyRate = enabled
		case catGuestGetHostName:
			req.GuestGetHostName = enabled
		case catGuestGetTimezone:
			req.GuestGetTimezone = enabled
		case catGuestGetUsers:
			req.GuestGetUsers = enabled
		case catGuestGetFsInfo:
			req.GuestGetFsInfo = enabled
		case catGuestGetDevices:
			req.GuestGetDevices = enabled
		}
	}
	return req
}
