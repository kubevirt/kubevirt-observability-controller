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

package devicedrivers

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DriversCache", func() {
	var cache *DriversCache

	BeforeEach(func() {
		cache = NewDriversCache()
	})

	It("should be empty before anything is stored", func() {
		Expect(cache.List()).To(BeEmpty())
	})

	It("should return the stored entries", func() {
		entries := []DriverEntry{
			{DeviceID: "1041", GuestOSVersionID: "2022", DriverVersion: "100.95.104.26200"},
		}

		cache.Store(entries)

		Expect(cache.List()).To(Equal(entries))
	})

	It("should replace the previous entries on store", func() {
		cache.Store([]DriverEntry{
			{DeviceID: "1041", GuestOSVersionID: "2019", DriverVersion: "100.94.104.25200"},
		})
		cache.Store([]DriverEntry{
			{DeviceID: "1042", GuestOSVersionID: "2022", DriverVersion: "100.95.104.26200"},
		})

		Expect(cache.List()).To(ConsistOf(
			DriverEntry{DeviceID: "1042", GuestOSVersionID: "2022", DriverVersion: "100.95.104.26200"},
		))
	})

	It("should drop all entries on clear", func() {
		cache.Store([]DriverEntry{
			{DeviceID: "1041", GuestOSVersionID: "2019", DriverVersion: "100.94.104.25200"},
		})

		cache.Clear()

		Expect(cache.List()).To(BeEmpty())
	})

	It("should report the config as valid before anything is stored", func() {
		Expect(cache.ConfigInvalid()).To(BeFalse())
	})

	It("should report the config as invalid once marked", func() {
		cache.MarkConfigInvalid()

		Expect(cache.ConfigInvalid()).To(BeTrue())
	})

	It("should keep the entries when the config is marked invalid", func() {
		entries := []DriverEntry{
			{DeviceID: "1041", GuestOSVersionID: "2022", DriverVersion: "100.95.104.26200"},
		}
		cache.Store(entries)

		cache.MarkConfigInvalid()

		Expect(cache.List()).To(Equal(entries))
	})

	It("should report the config as valid again on store", func() {
		cache.MarkConfigInvalid()

		cache.Store([]DriverEntry{
			{DeviceID: "1041", GuestOSVersionID: "2022", DriverVersion: "100.95.104.26200"},
		})

		Expect(cache.ConfigInvalid()).To(BeFalse())
	})

	It("should report the config as valid again on clear", func() {
		cache.MarkConfigInvalid()

		cache.Clear()

		Expect(cache.ConfigInvalid()).To(BeFalse())
	})

	It("should not let callers mutate the cached entries through the returned slice", func() {
		cache.Store([]DriverEntry{
			{DeviceID: "1041", GuestOSVersionID: "2019", DriverVersion: "100.94.104.25200"},
		})

		listed := cache.List()
		listed[0].DriverVersion = "tampered"

		Expect(cache.List()).To(ConsistOf(
			DriverEntry{DeviceID: "1041", GuestOSVersionID: "2019", DriverVersion: "100.94.104.25200"},
		))
	})
})
