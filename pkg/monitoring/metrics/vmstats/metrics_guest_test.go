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
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k6tv1 "kubevirt.io/api/core/v1"
)

func device(driverDate int64, name, version string, deviceID int64) string {
	return fmt.Sprintf(
		`{"driver-date":%d,"driver-name":%q,"driver-version":%q,`+
			`"id":{"device-id":%d,"vendor-id":6900,"type":"pci"}}`,
		driverDate, name, version, deviceID)
}

func devicesPayload(devices ...string) string {
	return `{"return":[` + strings.Join(devices, ",") + `]}`
}

func exposedLabelNames(results []operatormetrics.CollectorResult) [][]string {
	ExpectWithOffset(1, operatormetrics.CleanRegistry()).To(Succeed())

	registry := prometheus.NewRegistry()
	origRegister, origUnregister := operatormetrics.Register, operatormetrics.Unregister
	operatormetrics.Register, operatormetrics.Unregister = registry.Register, registry.Unregister
	defer func() {
		ExpectWithOffset(1, operatormetrics.CleanRegistry()).To(Succeed())
		operatormetrics.Register, operatormetrics.Unregister = origRegister, origUnregister
	}()

	ExpectWithOffset(1, operatormetrics.RegisterCollector(operatormetrics.Collector{
		Metrics:         []operatormetrics.Metric{guestDeviceDriverDate},
		CollectCallback: func() []operatormetrics.CollectorResult { return results },
	})).To(Succeed())

	families, err := registry.Gather()
	ExpectWithOffset(1, err).ToNot(HaveOccurred())

	var out [][]string
	for _, family := range families {
		for _, metric := range family.GetMetric() {
			names := make([]string, 0, len(metric.GetLabel()))
			for _, label := range metric.GetLabel() {
				names = append(names, label.GetName())
			}
			out = append(out, names)
		}
	}
	return out
}

var _ = Describe("Guest Metrics", func() {
	var report *VMIReport

	BeforeEach(func() {
		vmi := &k6tv1.VirtualMachineInstance{
			ObjectMeta: metav1.ObjectMeta{Name: "vm1", Namespace: "ns1"},
			Status: k6tv1.VirtualMachineInstanceStatus{
				NodeName:    "node1",
				GuestOSInfo: k6tv1.VirtualMachineInstanceGuestOSInfo{VersionID: "2022"},
			},
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

	It("should parse GuestGetDevices", func() {
		report.Stats.GuestGetDevices = devicesPayload(
			device(1651363200000000000, "Red Hat VirtIO SCSI controller", "100.85.104.20800", 4162),
		)

		results := collectGuestDevices(report)

		Expect(results).To(HaveLen(1))
		Expect(results[0].Value).To(Equal(1651363200.0))
		labels := results[0].ConstLabels
		Expect(labels).To(HaveKeyWithValue("driver_name", "Red Hat VirtIO SCSI controller"))
		Expect(labels).To(HaveKeyWithValue("driver_version", "100.85.104.20800"))
		Expect(labels).To(HaveKeyWithValue("device_id", "1042"))
		Expect(labels).To(HaveKeyWithValue("guest_os_version_id", "2022"))
		Expect(labels).ToNot(HaveKey("vendor_id"))
		Expect(labels).ToNot(HaveKey("device_type"))
	})

	It("should accept a GuestGetDevices payload that is a bare array", func() {
		report.Stats.GuestGetDevices = "[" +
			device(1651363200000000000, "Red Hat VirtIO SCSI controller", "100.85.104.20800", 4162) + "]"

		results := collectGuestDevices(report)

		Expect(results).To(HaveLen(1))
		Expect(results[0].ConstLabels).To(HaveKeyWithValue("device_id", "1042"))
	})

	It("should not expose labels the guest did not report", func() {
		report.Stats.GuestGetDevices =
			`{"return":[{"driver-date":1736726400000000000,"driver-name":"VirtIO Balloon Driver"}]}`

		results := collectGuestDevices(report)
		Expect(results).To(HaveLen(1))
		Expect(results[0].Value).To(Equal(1736726400.0))

		exposed := exposedLabelNames(results)
		Expect(exposed).To(HaveLen(1))
		Expect(exposed[0]).To(ContainElement("driver_name"))
		Expect(exposed[0]).ToNot(ContainElement("driver_version"))
		Expect(exposed[0]).ToNot(ContainElement("device_id"))
	})

	It("should not emit duplicate series for devices reported twice", func() {
		scsi := device(1771372800000000000, "Red Hat VirtIO SCSI controller", "100.103.104.29700", 4162)
		report.Stats.GuestGetDevices = devicesPayload(
			scsi,
			device(1768953600000000000, "Red Hat VirtIO SCSI pass-through", "100.102.104.29500", 4168),
			scsi,
		)

		results := collectGuestDevices(report)

		Expect(results).To(HaveLen(2), "the repeated SCSI controller should be reported once")

		Expect(results[0].Value).To(Equal(1771372800.0))
		Expect(results[0].ConstLabels).To(HaveKeyWithValue("device_id", "1042"))

		Expect(results[1].Value).To(Equal(1768953600.0))
		Expect(results[1].ConstLabels).To(HaveKeyWithValue("device_id", "1048"))
	})

	It("should keep distinct devices that differ only by device id", func() {
		report.Stats.GuestGetDevices = devicesPayload(
			device(1771372800000000000, "Red Hat VirtIO SCSI controller", "100.103.104.29700", 4100),
			device(1771372800000000000, "Red Hat VirtIO SCSI controller", "100.103.104.29700", 4162),
		)

		Expect(collectGuestDevices(report)).To(HaveLen(2))
	})

	It("should keep device ids that collide only when truncated to uint16", func() {
		report.Stats.GuestGetDevices = devicesPayload(
			device(1749427200000000000, "Red Hat VirtIO Ethernet Adapter", "100.101.104.28200", 4161),
			device(1749427200000000000, "Red Hat VirtIO Ethernet Adapter", "100.101.104.28200", 69697),
		)

		results := collectGuestDevices(report)

		Expect(results).To(HaveLen(2))
		Expect(results[0].ConstLabels).To(HaveKeyWithValue("device_id", "1041"))
		Expect(results[1].ConstLabels).To(HaveKeyWithValue("device_id", "11041"))
	})

	It("should skip devices without a usable driver date", func() {
		report.Stats.GuestGetDevices = `{"return":[
		 {"driver-name":"VirtIO Input Driver","driver-version":"1","id":{"device-id":4178}},
		 {"driver-date":0,"driver-name":"VirtIO Balloon Driver","id":{"device-id":4165}},
		 {"driver-date":-1,"driver-name":"VirtIO Serial Driver","id":{"device-id":4163}}
		]}`

		Expect(collectGuestDevices(report)).To(BeEmpty())
	})

	It("should include GuestGetDevices in the collected guest metrics", func() {
		report.Stats.GuestGetDevices = devicesPayload(
			device(1736726400000000000, "VirtIO Serial Driver", "100.100.104.27100", 4163),
		)

		results := collectGuestMetrics(report)
		names := make([]string, 0, len(results))
		for _, r := range results {
			names = append(names, r.Metric.GetOpts().Name)
		}
		Expect(names).To(ContainElement("kubevirt_vmi_guest_device_driver_date_seconds"))
	})

	It("should skip malformed GuestGetDevices JSON gracefully", func() {
		report.Stats.GuestGetDevices = `{"return":{"not":"an array"}}`
		Expect(collectGuestDevices(report)).To(BeEmpty())

		report.Stats.GuestGetDevices = `[{invalid json`
		Expect(collectGuestDevices(report)).To(BeEmpty())
	})

	It("should skip malformed JSON gracefully", func() {
		report.Stats.GuestGetOsInfo = `{invalid json`
		results := collectGuestMetrics(report)

		for _, r := range results {
			Expect(r.Metric.GetOpts().Name).ToNot(Equal("kubevirt_vmi_guest_os_info"))
		}
	})
})
