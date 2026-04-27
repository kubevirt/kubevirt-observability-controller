package metrics

import (
	"maps"
	"slices"
	"strings"

	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"

	k6tv1 "kubevirt.io/api/core/v1"
	instancetypeapi "kubevirt.io/api/instancetype"
)

var (
	VMStatsCollector = operatormetrics.Collector{
		Metrics: append(timestampMetrics,
			vmResourceRequests, vmResourceLimits, vmInfo,
			vmDiskAllocatedSize, vmCreationTimestamp, vmVnicInfo, vmLabels,
		),
		CollectCallback: vmStatsCollectorCallback,
	}

	vmLabelsSlice = []string{"name", "namespace"}

	timestampMetrics = []operatormetrics.Metric{
		startingTimestamp,
		runningTimestamp,
		migratingTimestamp,
		nonRunningTimestamp,
		errorTimestamp,
	}

	startingTimestamp = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vm_starting_status_last_transition_timestamp_seconds",
			Help: "Virtual Machine last transition timestamp to starting status.",
		},
		vmLabelsSlice,
	)

	runningTimestamp = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vm_running_status_last_transition_timestamp_seconds",
			Help: "Virtual Machine last transition timestamp to running status.",
		},
		vmLabelsSlice,
	)

	migratingTimestamp = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vm_migrating_status_last_transition_timestamp_seconds",
			Help: "Virtual Machine last transition timestamp to migrating status.",
		},
		vmLabelsSlice,
	)

	nonRunningTimestamp = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vm_non_running_status_last_transition_timestamp_seconds",
			Help: "Virtual Machine last transition timestamp to paused/stopped status.",
		},
		vmLabelsSlice,
	)

	errorTimestamp = operatormetrics.NewCounterVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vm_error_status_last_transition_timestamp_seconds",
			Help: "Virtual Machine last transition timestamp to error status.",
		},
		vmLabelsSlice,
	)

	vmResourceRequests = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vm_resource_requests",
			Help: "Resources requested by Virtual Machine.",
		},
		[]string{"name", "namespace", "resource", "unit", "source"},
	)

	vmResourceLimits = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vm_resource_limits",
			Help: "Resource limits set for a Virtual Machine.",
		},
		[]string{"name", "namespace", "resource", "unit"},
	)

	vmInfo = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vm_info",
			Help: "Information about Virtual Machines.",
		},
		[]string{
			"name", "namespace",
			"os", "workload", "flavor",
			"machine_type",
			"instance_type", "preference",
			"status", "status_group",
		},
	)

	vmDiskAllocatedSize = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vm_disk_allocated_size_bytes",
			Help: "Allocated disk size of a Virtual Machine in bytes.",
		},
		[]string{"name", "namespace", "persistentvolumeclaim", "volume_mode", "device"},
	)

	vmCreationTimestamp = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vm_create_date_timestamp_seconds",
			Help: "Virtual Machine creation timestamp.",
		},
		vmLabelsSlice,
	)

	vmVnicInfo = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vm_vnic_info",
			Help: "Details of Virtual Machine vNIC interfaces.",
		},
		[]string{
			"name", "namespace", "vnic_name",
			"binding_type", "network", "binding_name", "model",
		},
	)

	vmLabels = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vm_labels",
			Help: "The metric exposes the VM labels as Prometheus labels.",
		},
		vmLabelsSlice,
	)
)

func vmStatsCollectorCallback() []operatormetrics.CollectorResult {
	s := getStores()
	if s == nil || s.VM == nil {
		return []operatormetrics.CollectorResult{}
	}

	cachedObjs := s.VM.List()
	if len(cachedObjs) == 0 {
		return []operatormetrics.CollectorResult{}
	}

	vms := make([]*k6tv1.VirtualMachine, len(cachedObjs))
	for i, obj := range cachedObjs {
		vms[i] = obj.(*k6tv1.VirtualMachine)
	}

	return slices.Concat(
		CollectDiskAllocatedSize(vms),
		CollectVMsInfo(vms),
		CollectResourceRequestsAndLimits(vms),
		ReportVMStats(vms),
		collectVMCreationTimestamp(vms),
		CollectVmsVnicInfo(vms),
	)
}

func CollectVMsInfo(vms []*k6tv1.VirtualMachine) []operatormetrics.CollectorResult {
	results := make([]operatormetrics.CollectorResult, 0, len(vms))

	for _, vm := range vms {
		os, workload, flavor, machineType := None, None, None, None
		if vm.Spec.Template != nil {
			os, workload, flavor = GetSystemInfoFromAnnotations(
				vm.Spec.Template.ObjectMeta.Annotations,
			)
			if vm.Spec.Template.Spec.Domain.Machine != nil {
				machineType = vm.Spec.Template.Spec.Domain.Machine.Type
			}
		}

		instanceType := getVMInstancetype(vm)
		preference := getVMPreference(vm)

		results = append(results, operatormetrics.CollectorResult{
			Metric: vmInfo,
			Labels: []string{
				vm.Name, vm.Namespace,
				os, workload, flavor, machineType,
				instanceType, preference,
				strings.ToLower(string(vm.Status.PrintableStatus)),
				GetVMStatusGroup(vm.Status.PrintableStatus),
			},
			Value: 1.0,
		})
	}

	return results
}

func getVMInstancetype(vm *k6tv1.VirtualMachine) string {
	it := vm.Spec.Instancetype
	if it == nil {
		return None
	}

	s := getStores()
	if strings.EqualFold(it.Kind, instancetypeapi.SingularResourceName) {
		key := types.NamespacedName{Namespace: vm.Namespace, Name: it.Name}
		return FetchResourceName(key.String(), s.Instancetype)
	}

	if strings.EqualFold(it.Kind, instancetypeapi.ClusterSingularResourceName) {
		return FetchResourceName(it.Name, s.ClusterInstancetype)
	}

	return None
}

func getVMPreference(vm *k6tv1.VirtualMachine) string {
	pref := vm.Spec.Preference
	if pref == nil {
		return None
	}

	s := getStores()
	if strings.EqualFold(pref.Kind, instancetypeapi.SingularPreferenceResourceName) {
		key := types.NamespacedName{Namespace: vm.Namespace, Name: pref.Name}
		return FetchResourceName(key.String(), s.Preference)
	}

	if strings.EqualFold(
		pref.Kind, instancetypeapi.ClusterSingularPreferenceResourceName,
	) {
		return FetchResourceName(pref.Name, s.ClusterPreference)
	}

	return None
}

func CollectResourceRequestsAndLimits(
	vms []*k6tv1.VirtualMachine,
) []operatormetrics.CollectorResult {
	results := make([]operatormetrics.CollectorResult, 0, len(vms))

	for _, vm := range vms {
		results = append(results, collectVMResourceRequestsAndLimits(vm)...)
	}

	return results
}

func collectVMResourceRequestsAndLimits(
	vm *k6tv1.VirtualMachine,
) []operatormetrics.CollectorResult {
	return slices.Concat(
		collectMemoryRequests(vm),
		collectMemoryLimits(vm),
		collectCPURequestsFromDomainCPU(vm),
		collectCPURequestsFromResources(vm),
		collectCPULimits(vm),
		collectAllocatedCPU(vm),
		collectAllocatedMemory(vm),
	)
}

func collectMemoryRequests(vm *k6tv1.VirtualMachine) []operatormetrics.CollectorResult {
	var cr []operatormetrics.CollectorResult
	if vm.Spec.Template == nil {
		return cr
	}

	memRequested := vm.Spec.Template.Spec.Domain.Resources.Requests.Memory()
	if !memRequested.IsZero() {
		cr = append(cr, operatormetrics.CollectorResult{
			Metric: vmResourceRequests,
			Value:  float64(memRequested.Value()),
			Labels: []string{vm.Name, vm.Namespace, "memory", "bytes", "domain"},
		})
	}

	if vm.Spec.Template.Spec.Domain.Memory == nil {
		return cr
	}

	guestMem := vm.Spec.Template.Spec.Domain.Memory.Guest
	if guestMem != nil && !guestMem.IsZero() {
		cr = append(cr, operatormetrics.CollectorResult{
			Metric: vmResourceRequests,
			Value:  float64(guestMem.Value()),
			Labels: []string{vm.Name, vm.Namespace, "memory", "bytes", "guest"},
		})
	}

	hugepages := vm.Spec.Template.Spec.Domain.Memory.Hugepages
	if hugepages != nil {
		quantity, err := resource.ParseQuantity(hugepages.PageSize)
		if err == nil {
			cr = append(cr, operatormetrics.CollectorResult{
				Metric: vmResourceRequests,
				Value:  float64(quantity.Value()),
				Labels: []string{
					vm.Name, vm.Namespace, "memory", "bytes", "hugepages",
				},
			})
		}
	}

	return cr
}

func collectMemoryLimits(vm *k6tv1.VirtualMachine) []operatormetrics.CollectorResult {
	if vm.Spec.Template == nil {
		return nil
	}

	memLimit := vm.Spec.Template.Spec.Domain.Resources.Limits.Memory()
	if memLimit.IsZero() {
		return nil
	}

	return []operatormetrics.CollectorResult{{
		Metric: vmResourceLimits,
		Value:  float64(memLimit.Value()),
		Labels: []string{vm.Name, vm.Namespace, "memory", "bytes"},
	}}
}

func collectAllocatedMemory(
	vm *k6tv1.VirtualMachine,
) []operatormetrics.CollectorResult {
	if vm.Spec.Template == nil {
		return nil
	}

	vmi := &k6tv1.VirtualMachineInstance{Spec: vm.Spec.Template.Spec}
	allocatedMem := GetVirtualMemory(vmi)
	if allocatedMem != nil && !allocatedMem.IsZero() {
		return []operatormetrics.CollectorResult{{
			Metric: vmResourceRequests,
			Value:  float64(allocatedMem.Value()),
			Labels: []string{
				vm.Name, vm.Namespace, "memory", "bytes", "guest_effective",
			},
		}}
	}
	return nil
}

func collectCPURequestsFromDomainCPU(
	vm *k6tv1.VirtualMachine,
) []operatormetrics.CollectorResult {
	var cr []operatormetrics.CollectorResult
	if vm.Spec.Template == nil || vm.Spec.Template.Spec.Domain.CPU == nil {
		return cr
	}

	cpu := vm.Spec.Template.Spec.Domain.CPU
	if cpu.Cores != 0 {
		cr = append(cr, operatormetrics.CollectorResult{
			Metric: vmResourceRequests,
			Value:  float64(cpu.Cores),
			Labels: []string{vm.Name, vm.Namespace, "cpu", "cores", "domain"},
		})
	}
	if cpu.Threads != 0 {
		cr = append(cr, operatormetrics.CollectorResult{
			Metric: vmResourceRequests,
			Value:  float64(cpu.Threads),
			Labels: []string{vm.Name, vm.Namespace, "cpu", "threads", "domain"},
		})
	}
	if cpu.Sockets != 0 {
		cr = append(cr, operatormetrics.CollectorResult{
			Metric: vmResourceRequests,
			Value:  float64(cpu.Sockets),
			Labels: []string{vm.Name, vm.Namespace, "cpu", "sockets", "domain"},
		})
	}
	return cr
}

func collectCPURequestsFromResources(
	vm *k6tv1.VirtualMachine,
) []operatormetrics.CollectorResult {
	if vm.Spec.Template == nil {
		return []operatormetrics.CollectorResult{}
	}

	cpuReq := vm.Spec.Template.Spec.Domain.Resources.Requests.Cpu()
	if cpuReq == nil || cpuReq.IsZero() {
		if vm.Spec.Template.Spec.Domain.CPU == nil {
			return []operatormetrics.CollectorResult{
				{
					Metric: vmResourceRequests, Value: 1.0,
					Labels: []string{
						vm.Name, vm.Namespace, "cpu", "cores", "default",
					},
				},
				{
					Metric: vmResourceRequests, Value: 1.0,
					Labels: []string{
						vm.Name, vm.Namespace, "cpu", "threads", "default",
					},
				},
				{
					Metric: vmResourceRequests, Value: 1.0,
					Labels: []string{
						vm.Name, vm.Namespace, "cpu", "sockets", "default",
					},
				},
			}
		}

		return []operatormetrics.CollectorResult{}
	}

	return []operatormetrics.CollectorResult{{
		Metric: vmResourceRequests,
		Value:  float64(cpuReq.ScaledValue(resource.Milli)) / 1000,
		Labels: []string{vm.Name, vm.Namespace, "cpu", "cores", "requests"},
	}}
}

func collectCPULimits(
	vm *k6tv1.VirtualMachine,
) []operatormetrics.CollectorResult {
	if vm.Spec.Template == nil {
		return nil
	}

	cpuLimits := vm.Spec.Template.Spec.Domain.Resources.Limits.Cpu()
	if cpuLimits == nil || cpuLimits.IsZero() {
		return nil
	}

	return []operatormetrics.CollectorResult{{
		Metric: vmResourceLimits,
		Value:  float64(cpuLimits.ScaledValue(resource.Milli)) / 1000,
		Labels: []string{vm.Name, vm.Namespace, "cpu", "cores"},
	}}
}

func collectAllocatedCPU(
	vm *k6tv1.VirtualMachine,
) []operatormetrics.CollectorResult {
	if vm.Spec.Template == nil {
		return nil
	}

	if vm.Spec.Template.Spec.Domain.CPU != nil {
		vcpus := GetNumberOfVCPUs(vm.Spec.Template.Spec.Domain.CPU)
		return []operatormetrics.CollectorResult{{
			Metric: vmResourceRequests,
			Value:  float64(vcpus),
			Labels: []string{
				vm.Name, vm.Namespace, "cpu", "cores", "guest_effective",
			},
		}}
	}

	return []operatormetrics.CollectorResult{{
		Metric: vmResourceRequests,
		Value:  1.0,
		Labels: []string{
			vm.Name, vm.Namespace, "cpu", "cores", "guest_effective",
		},
	}}
}

func ReportVMStats(
	vms []*k6tv1.VirtualMachine,
) []operatormetrics.CollectorResult {
	cr := make([]operatormetrics.CollectorResult, 0, len(vms)*(1+len(timestampMetrics)))
	for _, vm := range vms {
		cr = append(cr, ReportVMLabels(vm)...)
		cr = append(cr, reportVMTimestamps(vm)...)
	}
	return cr
}

func ReportVMLabels(vm *k6tv1.VirtualMachine) []operatormetrics.CollectorResult {
	mergedLabels := make(map[string]string)
	maps.Copy(mergedLabels, vm.Labels)
	if vm.Spec.Template != nil {
		maps.Copy(mergedLabels, vm.Spec.Template.ObjectMeta.Labels)
	}

	if len(mergedLabels) == 0 {
		return nil
	}

	constLabels := make(map[string]string)
	for key, value := range mergedLabels {
		sanitized := SanitizeLabelName(key)
		constLabels["label_"+sanitized] = value
	}

	if len(constLabels) == 0 {
		return nil
	}

	return []operatormetrics.CollectorResult{{
		Metric:      vmLabels,
		Labels:      []string{vm.Name, vm.Namespace},
		ConstLabels: constLabels,
		Value:       1.0,
	}}
}

func reportVMTimestamps(
	vm *k6tv1.VirtualMachine,
) []operatormetrics.CollectorResult {
	status := vm.Status.PrintableStatus
	currentMetric := getTimestampMetric(status)
	lastTransition := GetLastConditionTransitionTime(vm)

	cr := make([]operatormetrics.CollectorResult, 0, len(timestampMetrics))
	for _, metric := range timestampMetrics {
		value := float64(0)
		if metric == currentMetric {
			value = float64(lastTransition)
		}
		cr = append(cr, operatormetrics.CollectorResult{
			Metric: metric,
			Labels: []string{vm.Name, vm.Namespace},
			Value:  value,
		})
	}
	return cr
}

func getTimestampMetric(
	status k6tv1.VirtualMachinePrintableStatus,
) *operatormetrics.CounterVec {
	switch {
	case ContainsStatus(status, StartingStatuses):
		return startingTimestamp
	case ContainsStatus(status, RunningStatuses):
		return runningTimestamp
	case ContainsStatus(status, MigratingStatuses):
		return migratingTimestamp
	case ContainsStatus(status, NonRunningStatuses):
		return nonRunningTimestamp
	default:
		return errorTimestamp
	}
}

func CollectDiskAllocatedSize(
	vms []*k6tv1.VirtualMachine,
) []operatormetrics.CollectorResult {
	var cr []operatormetrics.CollectorResult
	for _, vm := range vms {
		if vm.Spec.Template != nil {
			cr = append(cr, collectDiskMetricsFromPVC(vm)...)
		}
	}
	return cr
}

func collectDiskMetricsFromPVC(
	vm *k6tv1.VirtualMachine,
) []operatormetrics.CollectorResult {
	s := getStores()
	if s == nil || s.PVC == nil {
		return nil
	}

	cr := make(
		[]operatormetrics.CollectorResult, 0, len(vm.Spec.Template.Spec.Volumes),
	)
	for _, vol := range vm.Spec.Template.Spec.Volumes {
		pvcName, diskName, isDataVolume := getPVCAndDiskName(vol)
		if pvcName == "" {
			continue
		}

		key := NamespacedKey(vm.Namespace, pvcName)
		obj, exists, err := s.PVC.GetByKey(key)
		if err != nil || !exists {
			continue
		}

		pvc, ok := obj.(*k8sv1.PersistentVolumeClaim)
		if !ok {
			continue
		}

		cr = append(cr, getDiskSizeResult(vm, pvc, diskName, isDataVolume))
	}

	return cr
}

func getPVCAndDiskName(
	vol k6tv1.Volume,
) (pvcName, diskName string, isDataVolume bool) {
	if vol.PersistentVolumeClaim != nil {
		return vol.PersistentVolumeClaim.ClaimName, vol.Name, false
	}
	if vol.DataVolume != nil {
		return vol.DataVolume.Name, vol.Name, true
	}
	return "", "", false
}

func getDiskSizeResult(
	vm *k6tv1.VirtualMachine, pvc *k8sv1.PersistentVolumeClaim,
	diskName string, isDataVolume bool,
) operatormetrics.CollectorResult {
	var pvcSize *resource.Quantity

	if isDataVolume {
		pvcSize = getSizeFromDataVolumeTemplates(vm, pvc.Name)
	}

	if pvcSize == nil {
		pvcSize = pvc.Spec.Resources.Requests.Storage()
	}

	volumeMode := ""
	if pvc.Spec.VolumeMode != nil {
		volumeMode = string(*pvc.Spec.VolumeMode)
	}

	return operatormetrics.CollectorResult{
		Metric: vmDiskAllocatedSize,
		Value:  float64(pvcSize.Value()),
		Labels: []string{vm.Name, vm.Namespace, pvc.Name, volumeMode, diskName},
	}
}

func getSizeFromDataVolumeTemplates(
	vm *k6tv1.VirtualMachine, dvName string,
) *resource.Quantity {
	for _, dvTemplate := range vm.Spec.DataVolumeTemplates {
		if dvTemplate.Name == dvName {
			if dvTemplate.Spec.PVC != nil {
				return dvTemplate.Spec.PVC.Resources.Requests.Storage()
			} else if dvTemplate.Spec.Storage != nil {
				return dvTemplate.Spec.Storage.Resources.Requests.Storage()
			}
		}
	}
	return nil
}

func collectVMCreationTimestamp(
	vms []*k6tv1.VirtualMachine,
) []operatormetrics.CollectorResult {
	cr := make([]operatormetrics.CollectorResult, 0, len(vms))
	for _, vm := range vms {
		if !vm.CreationTimestamp.IsZero() {
			cr = append(cr, operatormetrics.CollectorResult{
				Metric: vmCreationTimestamp,
				Labels: []string{vm.Name, vm.Namespace},
				Value:  float64(vm.CreationTimestamp.Unix()),
			})
		}
	}
	return cr
}

func CollectVmsVnicInfo(
	vms []*k6tv1.VirtualMachine,
) []operatormetrics.CollectorResult {
	var results []operatormetrics.CollectorResult

	for _, vm := range vms {
		if vm.Spec.Template == nil ||
			vm.Spec.Template.Spec.Domain.Devices.Interfaces == nil {
			continue
		}

		interfaces := vm.Spec.Template.Spec.Domain.Devices.Interfaces
		networks := vm.Spec.Template.Spec.Networks

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
				Metric: vmVnicInfo,
				Labels: []string{
					vm.Name, vm.Namespace, iface.Name,
					bindingType, networkName, bindingName, model,
				},
				Value: 1.0,
			})
		}
	}

	return results
}
