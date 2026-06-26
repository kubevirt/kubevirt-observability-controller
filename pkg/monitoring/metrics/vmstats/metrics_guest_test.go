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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"
)

var _ = Describe("Guest Metrics", func() {
	var report *VMIReport

	BeforeEach(func() {
		vmi := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "vm1", Namespace: "ns1"},
			Status:     k6tv1.VirtualMachineInstanceStatus{NodeName: "node1"},
		}
		report = NewVMIReport(vmi, &VMStats{})
	})

	It("should return empty when no guest agent data", func() {
		Expect(collectGuestMetrics(report)).To(BeEmpty())
	})

	It("should parse GuestGetOsInfo", func() {
		report.Stats.GuestGetOsInfo = `{"id":"fedora","name":"Fedora Linux",` +
			`"version":"38","kernel-release":"6.2.0","machine":"x86_64"}`

		results := collectGuestMetrics(report)

		var found bool
		for _, r := range results {
			if r.Metric.GetOpts().Name == "kubevirt_vmi_guest_os_info" {
				found = true
				Expect(r.ConstLabels).To(HaveKeyWithValue("os_name", "Fedora Linux"))
				Expect(r.ConstLabels).To(HaveKeyWithValue("os_id", "fedora"))
				Expect(r.ConstLabels).To(HaveKeyWithValue("kernel_release", "6.2.0"))
				Expect(r.Value).To(Equal(1.0))
			}
		}
		Expect(found).To(BeTrue())
	})

	It("should parse GuestGetHostName", func() {
		report.Stats.GuestGetHostName = `{"host-name":"myhost"}`
		results := collectGuestMetrics(report)

		var found bool
		for _, r := range results {
			if r.Metric.GetOpts().Name == "kubevirt_vmi_guest_hostname" {
				found = true
				Expect(r.ConstLabels).To(HaveKeyWithValue("hostname", "myhost"))
			}
		}
		Expect(found).To(BeTrue())
	})

	It("should parse GuestGetUsers and count them", func() {
		report.Stats.GuestGetUsers = `[{"user":"root"},{"user":"testuser"}]`
		results := collectGuestMetrics(report)

		var found bool
		for _, r := range results {
			if r.Metric.GetOpts().Name == "kubevirt_vmi_guest_user_count" {
				found = true
				Expect(r.Value).To(Equal(2.0))
			}
		}
		Expect(found).To(BeTrue())
	})

	It("should skip malformed JSON gracefully", func() {
		report.Stats.GuestGetOsInfo = `{invalid json`
		results := collectGuestMetrics(report)

		for _, r := range results {
			Expect(r.Metric.GetOpts().Name).ToNot(Equal("kubevirt_vmi_guest_os_info"))
		}
	})
})
