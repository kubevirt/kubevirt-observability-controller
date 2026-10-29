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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	clonev1 "kubevirt.io/api/clone/v1beta1"
	k6tv1 "kubevirt.io/api/core/v1"
	exportv1 "kubevirt.io/api/export/v1"
	poolv1 "kubevirt.io/api/pool/v1beta1"
	snapshotv1 "kubevirt.io/api/snapshot/v1beta1"
	ctrlmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"
)

var _ = Describe("Metrics Setup", func() {
	BeforeEach(func() {
		Expect(operatormetrics.CleanRegistry()).To(Succeed())
	})

	It("should register all collectors when allowlist is nil", func() {
		err := SetupMetrics(&Stores{}, &Indexers{}, nil)
		Expect(err).ToNot(HaveOccurred())

		metrics := ListMetrics()
		Expect(metrics).ToNot(BeEmpty())
	})

	It("should register no collectors when allowlist is empty (none)", func() {
		err := SetupMetrics(&Stores{}, &Indexers{}, map[string]bool{})
		Expect(err).ToNot(HaveOccurred())

		metrics := ListMetrics()
		Expect(metrics).To(BeEmpty())
	})

	It("should register only allowed metrics", func() {
		allowlist := map[string]bool{
			"kubevirt_vm_info":  true,
			"kubevirt_vmi_info": true,
		}
		err := SetupMetrics(&Stores{}, &Indexers{}, allowlist)
		Expect(err).ToNot(HaveOccurred())

		registered := ListMetrics()
		Expect(registered).To(HaveLen(2))
		for _, m := range registered {
			Expect(allowlist).To(HaveKey(m.GetOpts().Name))
		}
	})

	It("should succeed with unknown metric names in allowlist", func() {
		allowlist := map[string]bool{
			"nonexistent_metric": true,
		}
		err := SetupMetrics(&Stores{}, &Indexers{}, allowlist)
		Expect(err).ToNot(HaveOccurred())

		metrics := ListMetrics()
		Expect(metrics).To(BeEmpty())
	})

	It("should register inventory info metrics", func() {
		err := SetupMetrics(&Stores{}, &Indexers{}, nil)
		Expect(err).ToNot(HaveOccurred())

		names := map[string]bool{}
		for _, m := range ListMetrics() {
			names[m.GetOpts().Name] = true
		}
		Expect(names).To(HaveKey("kubevirt_vmi_migration_info"))
		Expect(names).To(HaveKey("kubevirt_vmsnapshot_info"))
		Expect(names).To(HaveKey("kubevirt_vmsnapshot_create_date_timestamp_seconds"))
		Expect(names).To(HaveKey("kubevirt_vmrestore_info"))
		Expect(names).To(HaveKey("kubevirt_vmexport_info"))
		Expect(names).To(HaveKey("kubevirt_vmexport_ttl_expiration_timestamp_seconds"))
		Expect(names).To(HaveKey("kubevirt_vmclone_info"))
		Expect(names).To(HaveKey("kubevirt_vmclone_create_date_timestamp_seconds"))
		Expect(names).To(HaveKey("kubevirt_vmpool_info"))
		Expect(names).To(HaveKey("kubevirt_vmpool_desired_replicas"))
		Expect(names).To(HaveKey("kubevirt_vmpool_replicas"))
		Expect(names).To(HaveKey("kubevirt_vmpool_ready_replicas"))
		Expect(names).To(HaveKey("kubevirt_vmpool_paused"))
		Expect(names).To(HaveKey("kubevirt_vmpool_replica_failure"))
	})
})

var _ = Describe("Inventory collector store lifecycle", func() {
	It("should follow late initialization and replacement while applying the allowlist", func() {
		oldStores, oldIndexers := getStores(), getIndexers()
		DeferCleanup(func() {
			Expect(operatormetrics.CleanRegistry()).To(Succeed())
			SetStores(oldStores, oldIndexers)
		})
		Expect(operatormetrics.CleanRegistry()).To(Succeed())
		allowlist := map[string]bool{
			"kubevirt_vmi_migration_info": true,
			"kubevirt_vmsnapshot_info":    true,
			"kubevirt_vmrestore_info":     true,
			"kubevirt_vmexport_info":      true,
			"kubevirt_vmclone_info":       true,
			"kubevirt_vmpool_info":        true,
		}
		Expect(SetupMetrics(nil, nil, allowlist)).To(Succeed())

		collectNames := func() map[string]string {
			families, err := ctrlmetrics.Registry.Gather()
			Expect(err).ToNot(HaveOccurred())
			names := map[string]string{}
			for _, family := range families {
				if !strings.HasPrefix(family.GetName(), "kubevirt_") {
					continue
				}
				Expect(allowlist).To(HaveKey(family.GetName()))
				Expect(family.GetMetric()).To(HaveLen(1))
				for _, label := range family.GetMetric()[0].GetLabel() {
					if label.GetName() == "name" {
						names[family.GetName()] = label.GetValue()
					}
				}
			}
			return names
		}
		Expect(collectNames()).To(BeEmpty())

		for _, name := range []string{"first", "replacement"} {
			meta := metav1.ObjectMeta{Name: name, Namespace: "test-ns"}
			newStore := func(object any) cache.Store {
				store := cache.NewStore(cache.MetaNamespaceKeyFunc)
				Expect(store.Add(object)).To(Succeed())
				return store
			}
			migrations := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{})
			Expect(migrations.Add(&k6tv1.VirtualMachineInstanceMigration{ObjectMeta: meta})).To(Succeed())
			SetStores(&Stores{
				VMSnapshot: newStore(&snapshotv1.VirtualMachineSnapshot{ObjectMeta: meta}),
				VMRestore:  newStore(&snapshotv1.VirtualMachineRestore{ObjectMeta: meta}),
				VMExport:   newStore(&exportv1.VirtualMachineExport{ObjectMeta: meta}),
				VMClone:    newStore(&clonev1.VirtualMachineClone{ObjectMeta: meta}),
				VMPool:     newStore(&poolv1.VirtualMachinePool{ObjectMeta: meta}),
			}, &Indexers{VMIMigration: migrations})
			names := collectNames()
			Expect(names).To(HaveLen(len(allowlist)))
			for metric := range allowlist {
				Expect(names).To(HaveKeyWithValue(metric, name))
			}
		}
		SetStores(nil, nil)
		Expect(collectNames()).To(BeEmpty())
	})
})
