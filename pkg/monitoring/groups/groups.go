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

// Package groups defines the named bundles of metrics, recording rules, and
// alerts that the controller exposes, and resolves the --*-allowlist flags
// against them.
package groups

// Key is the ExtraFields entry that carries a metric or recording rule's group.
const Key = "group"

// Name identifies a group. Names follow <scope>-<topic>, where scope is one of
// virt, cluster, vm, vmi, or guest. A name with no scope prefix is a
// cross-cutting group.
type Name string

const (
	VirtAPI        Name = "virt-api"
	VirtController Name = "virt-controller"
	VirtHandler    Name = "virt-handler"
	VirtOperator   Name = "virt-operator"
	ClusterNodes   Name = "cluster-nodes"

	VMStatus     Name = "vm-status"
	VMSnapshot   Name = "vm-snapshot"
	VMIStatus    Name = "vmi-status"
	VMIMigration Name = "vmi-migration"

	VMICPU       Name = "vmi-cpu"
	VMIMemory    Name = "vmi-memory"
	VMINetwork   Name = "vmi-network"
	VMIStorage   Name = "vmi-storage"
	VMIVCPU      Name = "vmi-vcpu"
	VMIDirtyRate Name = "vmi-dirty-rate"

	VMIInterfaces Name = "vmi-interfaces"
	VMIHotplug    Name = "vmi-hotplug"

	GuestAgent      Name = "guest-agent"
	GuestFilesystem Name = "guest-filesystem"
	GuestOS         Name = "guest-os"
	GuestDrivers    Name = "guest-drivers"

	Inventory  Name = "inventory"
	Deprecated Name = "deprecated"
)

// Group is a bundle of metrics, recording rules, and alerts that belong
// together. Membership is declared at each definition site, not here: metrics
// and recording rules carry it in MetricOpts.ExtraFields[Key], alerts in the
// map keys of alerts.All.
type Group struct {
	Name        Name
	Description string

	// Default reports whether the group is enabled when no flag selects it.
	//
	// Set it to true if, and only if, KubeVirt does not itself publish the
	// group's contents. Running alongside KubeVirt must not double-publish a
	// series, so a group that mirrors upstream defaults to off. Check a new
	// group's contents against the KubeVirt release this tracks before
	// setting it.
	Default bool
}

// All is every group the controller knows about.
var All = []Group{
	{
		Name:        VirtAPI,
		Description: "virt-api availability, REST errors, and deprecated API usage",
	},
	{
		Name:        VirtController,
		Description: "virt-controller availability and REST errors",
	},
	{
		Name:        VirtHandler,
		Description: "virt-handler rollout health and REST errors",
	},
	{
		Name:        VirtOperator,
		Description: "virt-operator availability, leadership, and memory delta",
	},
	{
		Name:        ClusterNodes,
		Description: "Node counts for KVM capacity and schedulability",
	},

	{
		Name:        VMStatus,
		Description: "VirtualMachine status, resources, and lifecycle timestamps",
	},
	{
		Name:        VMSnapshot,
		Description: "VolumeSnapshot restore sizes and PVC labels",
	},
	{
		Name:        VMIStatus,
		Description: "VirtualMachineInstance status, vNICs, and launcher overhead",
	},
	{
		Name:        VMIMigration,
		Description: "Migration phase counts, outcomes, and timestamps",
	},

	{
		Name:        VMICPU,
		Description: "Guest CPU time and load averages",
	},
	{
		Name:        VMIMemory,
		Description: "Guest memory usage, swap traffic, and page faults",
	},
	{
		Name:        VMINetwork,
		Description: "Guest network interface traffic, errors, and drops",
	},
	{
		Name:        VMIStorage,
		Description: "Guest block device IOPS, traffic, and latency",
	},
	{
		Name:        VMIVCPU,
		Description: "Per-vCPU time, wait, and delay",
	},
	{
		Name:        VMIDirtyRate,
		Description: "Guest memory dirty rate",
	},

	{
		Name:        VMIInterfaces,
		Description: "Guest network interfaces and ordinal pod interface naming",
		Default:     true,
	},
	{
		Name:        VMIHotplug,
		Description: "Ephemeral hotplug volumes pending migration to persistent ones",
		Default:     true,
	},

	{
		Name:        GuestAgent,
		Description: "Guest hostname, timezone, and logged-in user count",
		Default:     true,
	},
	{
		Name:        GuestFilesystem,
		Description: "Guest filesystem capacity and usage",
	},
	{
		Name:        GuestOS,
		Description: "Guest operating system identity",
		Default:     true,
	},
	{
		Name:        GuestDrivers,
		Description: "Guest device driver versions and staleness",
		Default:     true,
	},

	{
		Name:        Inventory,
		Description: "Snapshot, restore, export, clone, pool, and migration inventory",
		Default:     true,
	},
	{
		Name:        Deprecated,
		Description: "Superseded recording rules kept for compatibility",
	},
}

var byName = func() map[Name]Group {
	m := make(map[Name]Group, len(All))
	for _, g := range All {
		m[g.Name] = g
	}
	return m
}()

// Lookup returns the group with the given name.
func Lookup(n Name) (Group, bool) {
	g, ok := byName[n]
	return g, ok
}

// Tag returns the ExtraFields map that places a metric or recording rule in a
// group. It panics on an unregistered name so a typo fails at init, not at
// flag-parsing time.
func Tag(n Name) map[string]string {
	if _, ok := byName[n]; !ok {
		panic("groups: unknown group " + string(n))
	}
	return map[string]string{Key: string(n)}
}

// Defaults returns the groups enabled when no flag selects otherwise.
func Defaults() []Group {
	var out []Group
	for _, g := range All {
		if g.Default {
			out = append(out, g)
		}
	}
	return out
}
