package metrics

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	k8sv1 "k8s.io/api/core/v1"
	toolscache "k8s.io/client-go/tools/cache"
	k6tv1 "kubevirt.io/api/core/v1"
	instancetypev1beta1 "kubevirt.io/api/instancetype/v1beta1"
	ctrlcache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type storeIndexer interface {
	GetStore() toolscache.Store
	GetIndexer() toolscache.Indexer
}

func SetupInformers(ctx context.Context, c ctrlcache.Cache) error {
	vm, err := informerFor(ctx, c, &k6tv1.VirtualMachine{})
	if err != nil {
		return fmt.Errorf("VM: %w", err)
	}

	vmi, err := informerFor(ctx, c, &k6tv1.VirtualMachineInstance{})
	if err != nil {
		return fmt.Errorf("VMI: %w", err)
	}

	pvc, err := informerFor(ctx, c, &k8sv1.PersistentVolumeClaim{})
	if err != nil {
		return fmt.Errorf("PVC: %w", err)
	}

	instancetype, err := informerFor(
		ctx, c, &instancetypev1beta1.VirtualMachineInstancetype{},
	)
	if err != nil {
		return fmt.Errorf("instancetype: %w", err)
	}

	clusterInstancetype, err := informerFor(
		ctx, c, &instancetypev1beta1.VirtualMachineClusterInstancetype{},
	)
	if err != nil {
		return fmt.Errorf("clusterInstancetype: %w", err)
	}

	preference, err := informerFor(
		ctx, c, &instancetypev1beta1.VirtualMachinePreference{},
	)
	if err != nil {
		return fmt.Errorf("preference: %w", err)
	}

	clusterPreference, err := informerFor(
		ctx, c, &instancetypev1beta1.VirtualMachineClusterPreference{},
	)
	if err != nil {
		return fmt.Errorf("clusterPreference: %w", err)
	}

	controllerRevision, err := informerFor(
		ctx, c, &appsv1.ControllerRevision{},
	)
	if err != nil {
		return fmt.Errorf("controllerRevision: %w", err)
	}

	vmim, err := informerFor(
		ctx, c, &k6tv1.VirtualMachineInstanceMigration{},
	)
	if err != nil {
		return fmt.Errorf("VMIM: %w", err)
	}

	if err := vmim.GetIndexer().AddIndexers(toolscache.Indexers{
		ByMigrationUIDIndex: func(obj any) ([]string, error) {
			vmim, ok := obj.(*k6tv1.VirtualMachineInstanceMigration)
			if !ok {
				return nil, nil
			}
			return []string{string(vmim.UID)}, nil
		},
	}); err != nil {
		return fmt.Errorf("VMIM index: %w", err)
	}

	pod, err := informerFor(ctx, c, &k8sv1.Pod{})
	if err != nil {
		return fmt.Errorf("pod: %w", err)
	}

	SetStores(
		&Stores{
			VM:                  vm.GetStore(),
			VMI:                 vmi.GetStore(),
			PVC:                 pvc.GetStore(),
			Instancetype:        instancetype.GetStore(),
			ClusterInstancetype: clusterInstancetype.GetStore(),
			Preference:          preference.GetStore(),
			ClusterPreference:   clusterPreference.GetStore(),
			ControllerRevision:  controllerRevision.GetStore(),
			VirtHandlerPod:      pod.GetStore(),
		},
		&Indexers{
			VMIMigration: vmim.GetIndexer(),
			KVPod:        pod.GetIndexer(),
		},
	)

	return nil
}

func informerFor(
	ctx context.Context, c ctrlcache.Cache, obj client.Object,
) (storeIndexer, error) {
	inf, err := c.GetInformer(ctx, obj)
	if err != nil {
		return nil, err
	}

	si, ok := inf.(storeIndexer)
	if !ok {
		return nil, fmt.Errorf(
			"informer %T does not expose store/indexer", inf,
		)
	}

	return si, nil
}
