package metrics

import (
	"regexp"
	"strings"

	k8sv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"

	k6tv1 "kubevirt.io/api/core/v1"
)

const (
	None      = ""
	Other     = "<other>"
	ModelNone = "<none>"

	AnnotationPrefix        = "vm.kubevirt.io/"
	InstancetypeVendorLabel = "instancetype.kubevirt.io/vendor"

	BindingTypeCore   = "core"
	BindingTypePlugin = "plugin"

	ByMigrationUIDIndex = "byMigrationUID"
)

var (
	WhitelistedInstanceTypeVendors = map[string]bool{
		"kubevirt.io": true,
		"redhat.com":  true,
	}

	InvalidLabelCharRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)

	StartingStatuses = []k6tv1.VirtualMachinePrintableStatus{
		k6tv1.VirtualMachineStatusProvisioning,
		k6tv1.VirtualMachineStatusStarting,
		k6tv1.VirtualMachineStatusWaitingForVolumeBinding,
	}

	RunningStatuses = []k6tv1.VirtualMachinePrintableStatus{
		k6tv1.VirtualMachineStatusRunning,
	}

	MigratingStatuses = []k6tv1.VirtualMachinePrintableStatus{
		k6tv1.VirtualMachineStatusMigrating,
	}

	NonRunningStatuses = []k6tv1.VirtualMachinePrintableStatus{
		k6tv1.VirtualMachineStatusStopped,
		k6tv1.VirtualMachineStatusPaused,
		k6tv1.VirtualMachineStatusStopping,
		k6tv1.VirtualMachineStatusTerminating,
	}

	ErrorStatuses = []k6tv1.VirtualMachinePrintableStatus{
		k6tv1.VirtualMachineStatusCrashLoopBackOff,
		k6tv1.VirtualMachineStatusUnknown,
		k6tv1.VirtualMachineStatusUnschedulable,
		k6tv1.VirtualMachineStatusErrImagePull,
		k6tv1.VirtualMachineStatusImagePullBackOff,
		k6tv1.VirtualMachineStatusPvcNotFound,
		k6tv1.VirtualMachineStatusDataVolumeError,
	}
)

func GetNumberOfVCPUs(cpuSpec *k6tv1.CPU) int64 {
	if cpuSpec == nil {
		return 1
	}

	vCPUs := cpuSpec.Cores
	if cpuSpec.Sockets != 0 {
		if vCPUs == 0 {
			vCPUs = cpuSpec.Sockets
		} else {
			vCPUs *= cpuSpec.Sockets
		}
	}
	if cpuSpec.Threads != 0 {
		if vCPUs == 0 {
			vCPUs = cpuSpec.Threads
		} else {
			vCPUs *= cpuSpec.Threads
		}
	}

	if vCPUs == 0 {
		return 1
	}
	return int64(vCPUs)
}

func GetVirtualMemory(vmi *k6tv1.VirtualMachineInstance) *resource.Quantity {
	if vmi.Spec.Domain.Memory != nil && vmi.Spec.Domain.Memory.Guest != nil {
		return vmi.Spec.Domain.Memory.Guest
	}

	reqMemory, isReqMemSet := vmi.Spec.Domain.Resources.Requests[k8sv1.ResourceMemory]

	if v, ok := vmi.Spec.Domain.Resources.Limits[k8sv1.ResourceMemory]; ok && !isReqMemSet {
		return &v
	}

	return &reqMemory
}

func GetSystemInfoFromAnnotations(annotations map[string]string) (os, workload, flavor string) {
	if val, ok := annotations[AnnotationPrefix+"os"]; ok {
		os = val
	}
	if val, ok := annotations[AnnotationPrefix+"workload"]; ok {
		workload = val
	}
	if val, ok := annotations[AnnotationPrefix+"flavor"]; ok {
		flavor = val
	}
	return
}

func GetVMStatusGroup(status k6tv1.VirtualMachinePrintableStatus) string {
	switch {
	case ContainsStatus(status, StartingStatuses):
		return "starting"
	case ContainsStatus(status, RunningStatuses):
		return "running"
	case ContainsStatus(status, MigratingStatuses):
		return "migrating"
	case ContainsStatus(status, NonRunningStatuses):
		return "non_running"
	case ContainsStatus(status, ErrorStatuses):
		return "error"
	}
	return "<unknown>"
}

func ContainsStatus(target k6tv1.VirtualMachinePrintableStatus, elems []k6tv1.VirtualMachinePrintableStatus) bool {
	for _, elem := range elems {
		if elem == target {
			return true
		}
	}
	return false
}

func SanitizeLabelName(name string) string {
	sanitized := InvalidLabelCharRE.ReplaceAllString(name, "_")
	if sanitized == "" || !isValidLabelStart(sanitized[0]) {
		sanitized = "_" + sanitized
	}
	return sanitized
}

func isValidLabelStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func NamespacedKey(namespace, name string) string {
	return namespace + "/" + name
}

func GetBinding(iface k6tv1.Interface) (bindingType, bindingName string) {
	switch {
	case iface.Masquerade != nil:
		return BindingTypeCore, "masquerade"
	case iface.Bridge != nil:
		return BindingTypeCore, "bridge"
	case iface.SRIOV != nil:
		return BindingTypeCore, "sriov"
	case iface.Binding != nil:
		return BindingTypePlugin, iface.Binding.Name
	}
	return "", ""
}

func GetNetworkName(ifaceName string, networks []k6tv1.Network) (string, bool) {
	for _, net := range networks {
		if net.Name == ifaceName {
			if net.Pod != nil {
				return "pod networking", true
			} else if net.Multus != nil {
				return net.Multus.NetworkName, true
			}
		}
	}
	return "", false
}

func GetLastConditionTransitionTime(vm *k6tv1.VirtualMachine) int64 {
	conditions := []k6tv1.VirtualMachineConditionType{
		k6tv1.VirtualMachineReady,
		k6tv1.VirtualMachineFailure,
		k6tv1.VirtualMachinePaused,
	}

	latestTransitionTime := int64(-1)

	for _, c := range vm.Status.Conditions {
		if containsConditionType(c.Type, conditions) && c.LastTransitionTime.Unix() > latestTransitionTime {
			latestTransitionTime = c.LastTransitionTime.Unix()
		}
	}

	return latestTransitionTime
}

func containsConditionType(target k6tv1.VirtualMachineConditionType, elems []k6tv1.VirtualMachineConditionType) bool {
	for _, elem := range elems {
		if elem == target {
			return true
		}
	}
	return false
}

func IsVMIOutdated(vmi *k6tv1.VirtualMachineInstance) bool {
	_, hasOutdatedLabel := vmi.Labels[k6tv1.OutdatedLauncherImageLabel]
	return hasOutdatedLabel
}

func IsVMIEvictable(vmi *k6tv1.VirtualMachineInstance) bool {
	if vmi.Spec.EvictionStrategy == nil {
		return true
	}

	switch *vmi.Spec.EvictionStrategy {
	case k6tv1.EvictionStrategyLiveMigrate:
		return hasConditionWithStatus(vmi, k6tv1.VirtualMachineInstanceIsMigratable, k8sv1.ConditionTrue)
	case k6tv1.EvictionStrategyLiveMigrateIfPossible:
		return hasConditionWithStatus(vmi, k6tv1.VirtualMachineInstanceIsMigratable, k8sv1.ConditionTrue)
	}

	return true
}

func hasConditionWithStatus(
	vmi *k6tv1.VirtualMachineInstance,
	condType k6tv1.VirtualMachineInstanceConditionType,
	status k8sv1.ConditionStatus,
) bool {
	for _, c := range vmi.Status.Conditions {
		if c.Type == condType {
			return c.Status == status
		}
	}
	return false
}

func FetchResourceName(key string, store cache.Store) string {
	if store == nil {
		return Other
	}

	obj, ok, err := store.GetByKey(key)
	if err != nil || !ok {
		return Other
	}

	apiObj, ok := obj.(v1.Object)
	if !ok {
		return Other
	}

	vendorName := apiObj.GetLabels()[InstancetypeVendorLabel]
	if _, isWhitelisted := WhitelistedInstanceTypeVendors[vendorName]; isWhitelisted {
		return apiObj.GetName()
	}

	return Other
}

func GetGuestOSInfo(vmi *k6tv1.VirtualMachineInstance) (kernelRelease, machineArch, name, versionID string) {
	if vmi.Status.GuestOSInfo == (k6tv1.VirtualMachineInstanceGuestOSInfo{}) {
		return
	}

	kernelRelease = vmi.Status.GuestOSInfo.KernelRelease
	machineArch = vmi.Status.GuestOSInfo.Machine
	name = vmi.Status.GuestOSInfo.Name
	versionID = vmi.Status.GuestOSInfo.VersionID
	return
}

func CalculateMigrationStatus(migrationState *k6tv1.VirtualMachineInstanceMigrationState) string {
	if !migrationState.Completed {
		return ""
	}
	if migrationState.Failed {
		return "failed"
	}
	return "succeeded"
}

func GetVMIPhase(vmi *k6tv1.VirtualMachineInstance) string {
	return strings.ToLower(string(vmi.Status.Phase))
}
