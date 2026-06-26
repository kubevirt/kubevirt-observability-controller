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
)

var _ = Describe("Unit Converters", func() {
	Describe("nanosecondsToSeconds", func() {
		It("should convert nanoseconds to seconds", func() {
			Expect(nanosecondsToSeconds(1_000_000_000)).To(Equal(1.0))
			Expect(nanosecondsToSeconds(500_000_000)).To(Equal(0.5))
			Expect(nanosecondsToSeconds(0)).To(Equal(0.0))
		})
	})

	Describe("kibibytesToBytes", func() {
		It("should convert kibibytes to bytes", func() {
			Expect(kibibytesToBytes(1)).To(Equal(1024.0))
			Expect(kibibytesToBytes(1024)).To(Equal(1048576.0))
			Expect(kibibytesToBytes(0)).To(Equal(0.0))
		})
	})

	Describe("humanReadableVCPUState", func() {
		It("should return correct state strings", func() {
			Expect(humanReadableVCPUState(VCPUOffline)).To(Equal("offline"))
			Expect(humanReadableVCPUState(VCPURunning)).To(Equal("running"))
			Expect(humanReadableVCPUState(VCPUBlocked)).To(Equal("blocked"))
			Expect(humanReadableVCPUState(99)).To(Equal("unknown"))
		})
	})
})
