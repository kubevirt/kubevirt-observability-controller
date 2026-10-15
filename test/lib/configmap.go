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

package lib

import (
	"fmt"
	"strings"
)

const GuestDriversConfigMapName = "kubevirt-guest-device-drivers-config"

func ApplyGuestDriversConfigMap(namespace, driversYAML string) error {
	var indented strings.Builder
	for line := range strings.SplitSeq(strings.Trim(driversYAML, "\n"), "\n") {
		fmt.Fprintf(&indented, "\n    %s", line)
	}

	manifest := fmt.Sprintf(`apiVersion: v1
kind: ConfigMap
metadata:
  name: %s
  namespace: %s
data:
  drivers.yaml: |%s`, GuestDriversConfigMapName, namespace, indented.String())

	return KubectlApply(manifest)
}

func DeleteGuestDriversConfigMap(namespace string) error {
	_, err := Kubectl("delete", "configmap", GuestDriversConfigMapName,
		"-n", namespace, "--ignore-not-found")
	return err
}
