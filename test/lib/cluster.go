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
	"os/exec"
	"strings"
	"time"
)

func Kubectl(args ...string) (string, error) {
	cmd := exec.Command("kubectl", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("kubectl %s failed: %w\n%s", strings.Join(args, " "), err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

func KubectlApply(manifest string) error {
	cmd := exec.Command("kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(manifest)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kubectl apply failed: %w\n%s", err, string(out))
	}
	return nil
}

func KubectlDelete(manifest string) error {
	cmd := exec.Command("kubectl", "delete", "--ignore-not-found", "-f", "-")
	cmd.Stdin = strings.NewReader(manifest)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("kubectl delete failed: %w\n%s", err, string(out))
	}
	return nil
}

func FindKubeVirtNamespace() (string, error) {
	out, err := Kubectl("get", "kubevirts", "-A", "-o", "jsonpath={.items[0].metadata.namespace}")
	if err != nil {
		return "", fmt.Errorf("finding KubeVirt namespace: %w", err)
	}
	if out == "" {
		return "", fmt.Errorf("no KubeVirt CR found on the cluster")
	}
	return out, nil
}

func HasCRD(name string) bool {
	_, err := Kubectl("get", "crd", name)
	return err == nil
}

func CreateNamespace(name string) error {
	_, err := Kubectl("create", "namespace", name)
	return err
}

func DeleteNamespace(name string) error {
	_, err := Kubectl("delete", "namespace", name, "--ignore-not-found")
	return err
}

func CreateTestVMI(namespace string) error {
	manifest := fmt.Sprintf(`apiVersion: kubevirt.io/v1
kind: VirtualMachineInstance
metadata:
  name: e2e-test-vmi
  namespace: %s
spec:
  domain:
    devices:
      disks:
      - disk:
          bus: virtio
        name: containerdisk
      - disk:
          bus: virtio
        name: cloudinitdisk
      interfaces:
      - masquerade: {}
        name: default
      rng: {}
    memory:
      guest: 1024M
    resources: {}
  networks:
  - name: default
    pod: {}
  terminationGracePeriodSeconds: 0
  volumes:
  - containerDisk:
      image: registry:5000/kubevirt/fedora-with-test-tooling-container-disk:devel
    name: containerdisk
  - cloudInitNoCloud:
      userData: |-
        #cloud-config
        password: fedora
        chpasswd: { expire: False }
    name: cloudinitdisk`, namespace)
	return KubectlApply(manifest)
}

func DeleteTestVMI(namespace string) error {
	_, err := Kubectl("delete", "vmi", "e2e-test-vmi", "-n", namespace, "--ignore-not-found")
	return err
}

func WaitForVMIRunning(namespace, name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, err := Kubectl("get", "vmi", name, "-n", namespace,
			"-o", "jsonpath={.status.phase}")
		if err == nil && out == "Running" {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("VMI %s/%s did not reach Running phase within %v",
		namespace, name, timeout)
}
