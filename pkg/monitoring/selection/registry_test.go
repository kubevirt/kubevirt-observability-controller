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

// Integration tests for the group registry: these need the definition
// packages, which groups itself cannot import.
package selection_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/groups"
	"github.com/kubevirt/kubevirt-observability-controller/pkg/monitoring/selection"
)

func TestSelection(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Selection Suite")
}

const testNamespace = "kubevirt"

func indexes() selection.Indexes {
	idx, err := selection.BuildIndexes(testNamespace)
	Expect(err).ToNot(HaveOccurred())
	return idx
}

func metricIndex() map[string]groups.Name        { return indexes().Metrics }
func recordingRuleIndex() map[string]groups.Name { return indexes().RecordingRules }
func alertIndex() map[string]groups.Name         { return indexes().Alerts }

func selected(index map[string]groups.Name, raw string) []string {
	got, _, err := groups.Resolve(index, raw)
	Expect(err).ToNot(HaveOccurred())

	out := make([]string, 0, len(got))
	for name, on := range got {
		if on {
			out = append(out, name)
		}
	}
	return out
}

var _ = Describe("Group registry", func() {
	// Indexing fails on any item with a missing or unregistered group, so these
	// three cover the completeness requirement: an untagged definition could
	// never be selected by any flag value.
	It("places every metric in a known group", func() {
		Expect(metricIndex()).ToNot(BeEmpty())
	})

	It("places every recording rule in a known group", func() {
		Expect(recordingRuleIndex()).ToNot(BeEmpty())
	})

	It("places every alert in a known group", func() {
		Expect(alertIndex()).ToNot(BeEmpty())
	})

	It("has no group that is empty", func() {
		used := map[groups.Name]bool{}
		for _, idx := range []map[string]groups.Name{metricIndex(), recordingRuleIndex(), alertIndex()} {
			for _, g := range idx {
				used[g] = true
			}
		}

		for _, g := range groups.All {
			Expect(used).To(HaveKey(g.Name),
				"group %q has no metrics, recording rules, or alerts", g.Name)
		}
	})

	// A collision would make a flag entry ambiguous, and Resolve prefers the
	// group, so an item would silently become unselectable by name.
	It("has no group name that collides with an item name", func() {
		for _, idx := range []map[string]groups.Name{metricIndex(), recordingRuleIndex(), alertIndex()} {
			for item := range idx {
				_, isGroup := groups.Lookup(groups.Name(item))
				Expect(isGroup).To(BeFalse(), "%q is both a group and an item name", item)
			}
		}
	})
})

// These lists are the contract behind "zero config duplicates nothing". Adding
// a Default: true group changes them, which is the moment to re-check that
// KubeVirt does not publish the same series.
var _ = Describe("Default selection", func() {
	It("exposes exactly the metrics KubeVirt does not publish", func() {
		Expect(selected(metricIndex(), "")).To(ConsistOf(
			// guest-drivers
			"kubevirt_vmi_guest_device_driver_timestamp_seconds",
			"kubevirt_guest_device_driver_latest_version_info",
			"kubevirt_guest_device_driver_config_invalid",
			// guest-agent
			"kubevirt_vmi_guest_hostname",
			"kubevirt_vmi_guest_timezone",
			"kubevirt_vmi_guest_user_count",
			// guest-os
			"kubevirt_vmi_guest_os_info",
			// vmi-interfaces
			"kubevirt_vmi_guest_interface_info",
			// vmi-hotplug
			"kubevirt_vmi_contains_ephemeral_hotplug_volume",
			// inventory
			"kubevirt_vmclone_info",
			"kubevirt_vmclone_create_date_timestamp_seconds",
			"kubevirt_vmexport_info",
			"kubevirt_vmexport_ttl_expiration_timestamp_seconds",
			"kubevirt_vmpool_info",
			"kubevirt_vmpool_desired_replicas",
			"kubevirt_vmpool_replicas",
			"kubevirt_vmpool_ready_replicas",
			"kubevirt_vmpool_paused",
			"kubevirt_vmpool_replica_failure",
			"kubevirt_vmrestore_info",
			"kubevirt_vmsnapshot_info",
			"kubevirt_vmsnapshot_create_date_timestamp_seconds",
			"kubevirt_vmi_migration_info",
		))
	})

	It("exposes exactly one recording rule", func() {
		Expect(selected(recordingRuleIndex(), "")).To(ConsistOf(
			"vmi:kubevirt_vmi_guest_device_driver_timestamp_seconds:max",
		))
	})

	It("exposes exactly the alerts KubeVirt does not publish", func() {
		Expect(selected(alertIndex(), "")).To(ConsistOf(
			"OutdatedGuestDeviceDrivers",
			"KubeVirtVMIOrdinalPodInterfaceName",
			"VirtualMachineInstanceHasEphemeralHotplugVolume",
		))
	})
})

// ForFlags is what main() calls. These cover the wiring between the three
// flags and the definitions, which was unreachable while it lived in cmd.
var _ = Describe("ForFlags", func() {
	It("resolves the default selection across all three types", func() {
		sel, err := selection.ForFlags(testNamespace, "", "", "")
		Expect(err).ToNot(HaveOccurred())

		Expect(sel.Metrics).To(HaveLen(23))
		Expect(sel.RecordingRules).To(HaveLen(1))
		Expect(sel.Alerts).To(HaveLen(3))
		Expect(sel.Empty).To(BeEmpty())

		for _, g := range []groups.Name{
			groups.GuestDrivers, groups.GuestAgent, groups.GuestOS,
			groups.Inventory, groups.VMIInterfaces, groups.VMIHotplug,
		} {
			Expect(sel.EnabledGroups).To(HaveKeyWithValue(g, true))
		}
		Expect(sel.EnabledGroups).ToNot(HaveKey(groups.VMIStatus))
	})

	It("applies each flag to its own artifact type", func() {
		sel, err := selection.ForFlags(testNamespace, "all", "none", "virt-api")
		Expect(err).ToNot(HaveOccurred())

		Expect(len(sel.Metrics)).To(BeNumerically(">", 23))
		Expect(sel.RecordingRules).To(BeEmpty())
		Expect(sel.Alerts).To(HaveLen(4))
	})

	It("names the flag that rejected an entry", func() {
		_, err := selection.ForFlags(testNamespace, "", "", "no-such-group")
		Expect(err).To(MatchError(ContainSubstring(selection.AlertsFlag)))
		Expect(err).To(MatchError(ContainSubstring("no-such-group")))
	})

	// inventory has metrics but no alerts, and the three flags share one
	// vocabulary, so this is legal and has to be reported rather than looking
	// like an empty selection.
	It("reports a group with no items of the flag's type", func() {
		sel, err := selection.ForFlags(testNamespace, "", "", "inventory")
		Expect(err).ToNot(HaveOccurred())
		Expect(sel.Alerts).To(BeEmpty())
		Expect(sel.Empty).To(HaveKeyWithValue(selection.AlertsFlag, ConsistOf("inventory")))
	})
})
