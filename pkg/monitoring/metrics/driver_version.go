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

import (
	"strconv"
	"strings"
)

// DriverBuildNumber returns the final component of a four-component Windows
// driver version. For virtio-win, the first three components describe the target
// Windows kernel, RHEL certification version and an internal number; they do not
// determine build ordering. Compare builds only for the same device and guest
// OS version. Keep the full component (e.g. 30201), including any build revision.
// Unknown version formats are not comparable and must not be treated as build 0.
func DriverBuildNumber(version string) (uint16, bool) {
	parts := strings.Split(strings.TrimSpace(version), ".")
	if len(parts) != 4 {
		return 0, false
	}

	var build uint64
	for _, part := range parts {
		value, err := strconv.ParseUint(part, 10, 16)
		if err != nil {
			return 0, false
		}
		build = value
	}
	return uint16(build), true
}
