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

package metrics

import "testing"

func TestDriverBuildNumber(t *testing.T) {
	for _, tc := range []struct {
		name, version string
		want          uint16
		valid         bool
	}{
		{"normal build", "100.103.104.30200", 30200, true},
		{"different RHEL and internal number", "100.90.105.30200", 30200, true},
		{"legacy Windows", "51.64.104.4900", 4900, true},
		{"preserve sub-build revision", "100.103.104.30201", 30201, true},
		{"trim whitespace", " 100.103.104.30200 ", 30200, true},
		{"zero build", "100.103.104.0", 0, true},
		{"largest Windows component", "100.103.104.65535", 65535, true},
		{"missing version", "", 0, false},
		{"unknown format", "1.4.0", 0, false},
		{"too many components", "100.103.104.30200.1", 0, false},
		{"invalid prefix", "100.foo.104.30200", 0, false},
		{"empty component", "100..104.30200", 0, false},
		{"negative build", "100.103.104.-1", 0, false},
		{"signed build", "100.103.104.+1", 0, false},
		{"fractional build", "100.103.104.30200.5", 0, false},
		{"overflow", "100.103.104.65536", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, valid := DriverBuildNumber(tc.version)
			if got != tc.want || valid != tc.valid {
				t.Fatalf("DriverBuildNumber(%q) = (%d, %t), want (%d, %t)",
					tc.version, got, valid, tc.want, tc.valid)
			}
		})
	}
}
