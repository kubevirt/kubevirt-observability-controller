package metrics

import (
	"strconv"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"

	k6tv1 "kubevirt.io/api/core/v1"
)

var (
	VMIStatsCollector = operatormetrics.Collector{
		Metrics: []operatormetrics.Metric{
			vmiInfo,
			vmiEvictionBlocker,
			vmiAddresses,
			vmiMigrationStartTime,
			vmiMigrationEndTime,
			vmiVnicInfo,
			vmiLauncherMemoryOverhead,
			vmiEphemeralHotplugVolume,
		},
		CollectCallback: vmiStatsCollectorCallback,
	}

	vmiInfo = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_info",
			Help: "Information about VirtualMachineInstances.",
		},
		[]string{
			"node", "namespace", "name",
			"phase", "os", "workload", "flavor",
			"instance_type", "preference",
			"guest_os_kernel_release", "guest_os_machine",
			"guest_os_arch", "guest_os_name", "guest_os_version_id",
			"evictable", "outdated",
			"vmi_pod",
		},
	)

	vmiEvictionBlocker = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_non_evictable",
			Help: "Indication for a VirtualMachine that its eviction strategy " +
				"is set to Live Migration but is not migratable.",
		},
		[]string{"node", "namespace", "name"},
	)

	vmiAddresses = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_status_addresses",
			Help: "The addresses of a VirtualMachineInstance.",
		},
		[]string{
			"node", "namespace", "name",
			"vnic_name", "interface_name", "address", "type",
		},
	)

	vmiMigrationStartTime = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migration_start_time_seconds",
			Help: "The time at which the migration started.",
		},
		[]string{"node", "namespace", "name", "migration_name"},
	)

	vmiMigrationEndTime = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migration_end_time_seconds",
			Help: "The time at which the migration ended.",
		},
		[]string{"node", "namespace", "name", "migration_name", "status"},
	)

	vmiVnicInfo = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_vnic_info",
			Help: "Details of VirtualMachineInstance vNIC interfaces.",
		},
		[]string{
			"name", "namespace", "vnic_name",
			"binding_type", "network", "binding_name", "model",
		},
	)

	vmiLauncherMemoryOverhead = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_launcher_memory_overhead_bytes",
			Help: "Estimation of the memory amount required for " +
				"virt-launcher's infrastructure components.",
		},
		[]string{"namespace", "name"},
	)

	vmiEphemeralHotplugVolume = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_contains_ephemeral_hotplug_volume",
			Help: "Reported only for VMIs that contain an ephemeral " +
				"hotplug volume.",
		},
		[]string{"namespace", "name", "volume_name"},
	)
)

func vmiStatsCollectorCallback() []operatormetrics.CollectorResult {
	if stores == nil || stores.VMI == nil {
		return []operatormetrics.CollectorResult{}
	}

	cachedObjs := stores.VMI.List()
	if len(cachedObjs) == 0 {
		return []operatormetrics.CollectorResult{}
	}

	vmis := make([]*k6tv1.VirtualMachineInstance, len(cachedObjs))
	for i, obj := range cachedObjs {
		vmis[i] = obj.(*k6tv1.VirtualMachineInstance)
	}

	return ReportVMIsStats(vmis)
}

func ReportVMIsStats(
	vmis []*k6tv1.VirtualMachineInstance,
) []operatormetrics.CollectorResult {
	crs := make([]operatormetrics.CollectorResult, 0, len(vmis)*8)

	for _, vmi := range vmis {
		crs = append(crs, CollectVMIInfo(vmi), getEvictionBlocker(vmi))
		crs = append(crs, collectVMIInterfacesInfo(vmi)...)
		crs = append(crs, CollectVMIMigrationTime(vmi)...)
		crs = append(crs, CollectVMIsVnicInfo(vmi)...)
		crs = append(crs, collectVMILauncherMemoryOverhead(vmi))
		crs = append(crs, CollectVMIEphemeralHotplug(vmi)...)
	}

	return crs
}

func collectVMILauncherMemoryOverhead(
	vmi *k6tv1.VirtualMachineInstance,
) operatormetrics.CollectorResult {
	var memoryOverheadValue int64
	if vmi.Status.Memory != nil && vmi.Status.Memory.MemoryOverhead != nil {
		memoryOverheadValue = vmi.Status.Memory.MemoryOverhead.Value()
	}

	return operatormetrics.CollectorResult{
		Metric: vmiLauncherMemoryOverhead,
		Labels: []string{vmi.Namespace, vmi.Name},
		Value:  float64(memoryOverheadValue),
	}
}

func CollectVMIInfo(
	vmi *k6tv1.VirtualMachineInstance,
) operatormetrics.CollectorResult {
	os, workload, flavor := GetSystemInfoFromAnnotations(vmi.Annotations)
	instanceType := getVMIInstancetype(vmi)
	preference := getVMIPreference(vmi)
	kernelRelease, machineArch, name, versionID := GetGuestOSInfo(vmi)
	machineType := getVMIMachine(vmi)
	vmiPod := getVMIPod(vmi)

	return operatormetrics.CollectorResult{
		Metric: vmiInfo,
		Labels: []string{
			vmi.Status.NodeName, vmi.Namespace, vmi.Name,
			GetVMIPhase(vmi), os, workload, flavor,
			instanceType, preference,
			kernelRelease, machineType, machineArch, name, versionID,
			strconv.FormatBool(IsVMIEvictable(vmi)),
			strconv.FormatBool(IsVMIOutdated(vmi)),
			vmiPod,
		},
		Value: 1.0,
	}
}

func getVMIMachine(vmi *k6tv1.VirtualMachineInstance) string {
	if vmi.Status.Machine != nil {
		return vmi.Status.Machine.Type
	}
	return ""
}

func getVMIPod(vmi *k6tv1.VirtualMachineInstance) string {
	if indexers == nil || indexers.KVPod == nil {
		return None
	}

	objs, err := indexers.KVPod.ByIndex(
		cache.NamespaceIndex, vmi.Namespace,
	)
	if err != nil {
		return None
	}

	for _, obj := range objs {
		pod, ok := obj.(*k8sv1.Pod)
		if !ok {
			continue
		}

		if pod.Labels["kubevirt.io/created-by"] == string(vmi.UID) &&
			pod.Status.Phase == k8sv1.PodRunning &&
			vmi.Status.NodeName == pod.Spec.NodeName {
			return pod.Name
		}
	}

	return None
}

func getVMIInstancetype(vmi *k6tv1.VirtualMachineInstance) string {
	if name, ok := vmi.Annotations[k6tv1.InstancetypeAnnotation]; ok {
		key := types.NamespacedName{
			Namespace: vmi.Namespace, Name: name,
		}
		return FetchResourceName(key.String(), stores.Instancetype)
	}

	if name, ok := vmi.Annotations[k6tv1.ClusterInstancetypeAnnotation]; ok {
		return FetchResourceName(name, stores.ClusterInstancetype)
	}

	return None
}

func getVMIPreference(vmi *k6tv1.VirtualMachineInstance) string {
	if name, ok := vmi.Annotations[k6tv1.PreferenceAnnotation]; ok {
		key := types.NamespacedName{
			Namespace: vmi.Namespace, Name: name,
		}
		return FetchResourceName(key.String(), stores.Preference)
	}

	if name, ok := vmi.Annotations[k6tv1.ClusterPreferenceAnnotation]; ok {
		return FetchResourceName(name, stores.ClusterPreference)
	}

	return None
}

func getEvictionBlocker(
	vmi *k6tv1.VirtualMachineInstance,
) operatormetrics.CollectorResult {
	nonEvictable := 1.0
	if IsVMIEvictable(vmi) {
		nonEvictable = 0.0
	}

	return operatormetrics.CollectorResult{
		Metric: vmiEvictionBlocker,
		Labels: []string{vmi.Status.NodeName, vmi.Namespace, vmi.Name},
		Value:  nonEvictable,
	}
}

func collectVMIInterfacesInfo(
	vmi *k6tv1.VirtualMachineInstance,
) []operatormetrics.CollectorResult {
	var crs []operatormetrics.CollectorResult

	for _, iface := range vmi.Status.Interfaces {
		if cr := collectVMIInterfaceInfo(vmi, iface); cr != nil {
			crs = append(crs, *cr)
		}
	}

	return crs
}

func collectVMIInterfaceInfo(
	vmi *k6tv1.VirtualMachineInstance,
	iface k6tv1.VirtualMachineInstanceNetworkInterface,
) *operatormetrics.CollectorResult {
	interfaceType := "ExternalInterface"

	if iface.IP == "" {
		if iface.Name == "" && iface.InterfaceName == "" {
			return nil
		}
		interfaceType = "SystemInterface"
	}

	return &operatormetrics.CollectorResult{
		Metric: vmiAddresses,
		Labels: []string{
			vmi.Status.NodeName, vmi.Namespace, vmi.Name,
			iface.Name, iface.InterfaceName, iface.IP, interfaceType,
		},
		Value: 1.0,
	}
}

func CollectVMIMigrationTime(
	vmi *k6tv1.VirtualMachineInstance,
) []operatormetrics.CollectorResult {
	var cr []operatormetrics.CollectorResult

	if vmi.Status.MigrationState == nil {
		return cr
	}

	migrationName := getMigrationNameFromUID(
		vmi.Status.MigrationState.MigrationUID,
	)

	if vmi.Status.MigrationState.StartTimestamp != nil {
		cr = append(cr, operatormetrics.CollectorResult{
			Metric: vmiMigrationStartTime,
			Value: float64(
				vmi.Status.MigrationState.StartTimestamp.Unix(),
			),
			Labels: []string{
				vmi.Status.NodeName, vmi.Namespace, vmi.Name, migrationName,
			},
		})
	}

	if vmi.Status.MigrationState.EndTimestamp != nil {
		cr = append(cr, operatormetrics.CollectorResult{
			Metric: vmiMigrationEndTime,
			Value: float64(
				vmi.Status.MigrationState.EndTimestamp.Unix(),
			),
			Labels: []string{
				vmi.Status.NodeName, vmi.Namespace, vmi.Name,
				migrationName,
				CalculateMigrationStatus(vmi.Status.MigrationState),
			},
		})
	}

	return cr
}

func getMigrationNameFromUID(migrationUID types.UID) string {
	if indexers == nil || indexers.VMIMigration == nil {
		return None
	}

	objs, err := indexers.VMIMigration.ByIndex(
		ByMigrationUIDIndex, string(migrationUID),
	)
	if err != nil || len(objs) == 0 {
		return None
	}

	return objs[0].(*k6tv1.VirtualMachineInstanceMigration).Name
}

func CollectVMIsVnicInfo(
	vmi *k6tv1.VirtualMachineInstance,
) []operatormetrics.CollectorResult {
	interfaces := vmi.Spec.Domain.Devices.Interfaces
	results := make(
		[]operatormetrics.CollectorResult, 0, len(interfaces),
	)
	networks := vmi.Spec.Networks

	for _, iface := range interfaces {
		model := ModelNone
		if iface.Model != "" {
			model = iface.Model
		}
		bindingType, bindingName := GetBinding(iface)
		networkName, matchFound := GetNetworkName(iface.Name, networks)
		if !matchFound {
			continue
		}

		results = append(results, operatormetrics.CollectorResult{
			Metric: vmiVnicInfo,
			Labels: []string{
				vmi.Name, vmi.Namespace, iface.Name,
				bindingType, networkName, bindingName, model,
			},
			Value: 1.0,
		})
	}

	return results
}

func CollectVMIEphemeralHotplug(
	vmi *k6tv1.VirtualMachineInstance,
) []operatormetrics.CollectorResult {
	annotations := vmi.GetAnnotations()
	volumeName, exists := annotations[k6tv1.EphemeralHotplugAnnotation]
	if !exists {
		return nil
	}

	return []operatormetrics.CollectorResult{{
		Metric: vmiEphemeralHotplugVolume,
		Labels: []string{vmi.Namespace, vmi.Name, volumeName},
		Value:  1,
	}}
}
