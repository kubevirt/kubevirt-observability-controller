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

package groups

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestGroups(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Groups Suite")
}

// index stands in for the real one: two items in a Default: true group and two
// in a Default: false group.
var index = map[string]Name{
	"kubevirt_vmi_guest_hostname":          GuestAgent,
	"kubevirt_vmi_guest_timezone":          GuestAgent,
	"kubevirt_vmi_cpu_usage_seconds_total": VMICPU,
	"kubevirt_vmi_guest_load_1m":           VMICPU,
}

var _ = Describe("Resolve", func() {
	DescribeTable("accepts",
		func(raw string, expected []string) {
			got, _, err := Resolve(index, raw)
			Expect(err).ToNot(HaveOccurred())

			names := make([]string, 0, len(got))
			for n, on := range got {
				if on {
					names = append(names, n)
				}
			}
			Expect(names).To(ConsistOf(expected))
		},
		Entry("empty selects the default groups", "",
			[]string{"kubevirt_vmi_guest_hostname", "kubevirt_vmi_guest_timezone"}),
		Entry("all selects everything", "all",
			[]string{
				"kubevirt_vmi_guest_hostname", "kubevirt_vmi_guest_timezone",
				"kubevirt_vmi_cpu_usage_seconds_total", "kubevirt_vmi_guest_load_1m",
			}),
		Entry("none selects nothing", "none", []string{}),
		Entry("a bare group is absolute", "vmi-cpu",
			[]string{"kubevirt_vmi_cpu_usage_seconds_total", "kubevirt_vmi_guest_load_1m"}),
		Entry("a bare item is absolute", "kubevirt_vmi_guest_load_1m",
			[]string{"kubevirt_vmi_guest_load_1m"}),
		Entry("mixing a bare group and item is absolute", "vmi-cpu,kubevirt_vmi_guest_hostname",
			[]string{
				"kubevirt_vmi_cpu_usage_seconds_total", "kubevirt_vmi_guest_load_1m",
				"kubevirt_vmi_guest_hostname",
			}),
		Entry("+group adds to the defaults", "+vmi-cpu",
			[]string{
				"kubevirt_vmi_guest_hostname", "kubevirt_vmi_guest_timezone",
				"kubevirt_vmi_cpu_usage_seconds_total", "kubevirt_vmi_guest_load_1m",
			}),
		Entry("-group removes from the defaults", "-guest-agent", []string{}),
		Entry("-item removes a single default", "-kubevirt_vmi_guest_timezone",
			[]string{"kubevirt_vmi_guest_hostname"}),
		Entry("deltas combine", "+vmi-cpu,-kubevirt_vmi_guest_load_1m",
			[]string{
				"kubevirt_vmi_guest_hostname", "kubevirt_vmi_guest_timezone",
				"kubevirt_vmi_cpu_usage_seconds_total",
			}),
		Entry("whitespace is ignored", " +vmi-cpu , -kubevirt_vmi_guest_load_1m ",
			[]string{
				"kubevirt_vmi_guest_hostname", "kubevirt_vmi_guest_timezone",
				"kubevirt_vmi_cpu_usage_seconds_total",
			}),
	)

	DescribeTable("rejects",
		func(raw, because string) {
			_, _, err := Resolve(index, raw)
			Expect(err).To(MatchError(ContainSubstring(because)))
		},
		Entry("an unknown name", "nonsense", "neither a known group nor a known name"),
		Entry("a name from another artifact type", "VirtAPIDown", "neither a known group"),
		Entry("mixed absolute and relative forms", "+vmi-cpu,guest-agent", "cannot mix"),
		Entry("a sign with no name", "+", "no name after its sign"),
		Entry("only signs", "+-", "no name after its sign"),
		Entry("a repeated leading sign", "+-vmi-cpu", "more than one leading sign"),
		Entry("signs the other way round", "-+vmi-cpu", "more than one leading sign"),
		Entry("all with a sign", "+all", "cannot take a + prefix"),
		Entry("none with a sign", "-none", "cannot take a - prefix"),
		Entry("all combined with another entry", "all,vmi-cpu", "cannot be combined"),
		Entry("only separators", ",,,", "no entries"),
	)

	It("suggests the intended name when the case is wrong", func() {
		_, _, err := Resolve(index, "KUBEVIRT_VMI_GUEST_LOAD_1M")
		Expect(err).To(MatchError(ContainSubstring(`did you mean "kubevirt_vmi_guest_load_1m"`)))
	})

	It("does not let a delta resurrect an item removed earlier in the list", func() {
		got, _, err := Resolve(index, "+kubevirt_vmi_guest_load_1m,-kubevirt_vmi_guest_load_1m")
		Expect(err).ToNot(HaveOccurred())
		Expect(got).ToNot(HaveKey("kubevirt_vmi_guest_load_1m"))
	})

	// The three flags share one group vocabulary, so a group with no items of
	// this artifact type has to be accepted. Reporting it separates that from
	// the typo case, which errors.
	It("reports a known group that has no items of this kind", func() {
		got, empty, err := Resolve(index, "inventory")
		Expect(err).ToNot(HaveOccurred())
		Expect(got).To(BeEmpty())
		Expect(empty).To(ConsistOf("inventory"))
	})

	It("reports nothing when every entry matched", func() {
		_, empty, err := Resolve(index, "+vmi-cpu")
		Expect(err).ToNot(HaveOccurred())
		Expect(empty).To(BeEmpty())
	})
})
