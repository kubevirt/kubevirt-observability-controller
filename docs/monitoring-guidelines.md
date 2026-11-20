# Monitoring Guidelines

Conventions for the recording rules and alerts reconciled by the
kubevirt-observability-controller.

Alert and recording rule expressions are kept in sync with
[kubevirt/kubevirt](https://github.com/kubevirt/kubevirt)
(`pkg/monitoring/rules/alerts/vms.go`). When syncing, port expressions
verbatim, including the `vm` label wrapping described below.

`VMNonRecoverableOSPanic` is the one intentional exception to verbatim
syncing. kubevirt applies its new-counter fallback after aggregating by
VM; the controller decides it per exporter series, because the panic
counter is process-local in virt-handler and a restarting or migrating
handler exposes a fresh series (see the comment on the alert). When
syncing, keep the controller's per-exporter form; converging the two
expressions is upstream work in kubevirt/kubevirt.

## Alert Rules

### VM-Scoped Alerts: `vm` Typed Resource Label

VM-scoped alerts (alerts in the VM workload groups in
`pkg/monitoring/rules/alerts/vms.go` that target a specific VM)
MUST include a `vm` typed resource label identifying the
VirtualMachine.

**Why:**
The `vm` label is a typed resource label. It enables the
monitoring-plugin to navigate from an alert to the VM resource page,
and lets alert-to-resource correlation resolve the alert to the
VirtualMachine.

**How:**
Wrap the PromQL expression with the `withVMLabel()` helper from
`pkg/monitoring/rules/alerts/vms.go`. It uses `label_replace` to copy
the `name` label into `vm`:

```go
Expr: intstr.FromString(withVMLabel("vmi:kubevirt_vmi_guest_queue_length:sum{namespace!=''} > 10")),
```

For VMI-scoped alerts that identify the workload by a different label,
use `withVMLabelFrom()` to derive `vm` from that label instead:

```go
Expr: intstr.FromString(withVMLabelFrom(
    "sum by (vmi, namespace) (topk by (vmi, namespace, vmim) " +
        "(1, max_over_time(kubevirt_vmi_migration_succeeded[1d]))) >= 12",
    "vmi",
)),
```

The existing `name`/`vmi` labels are preserved for backward
compatibility; do not remove them.

**Testing:**
When adding a new VM-scoped alert, add a promtool unit test case in
`pkg/monitoring/rules/alerts/vms_test.promtool.yml` that asserts `vm`
in `exp_labels` (see the existing VM alert test cases for the
pattern). The fixtures run as part of `make test`.

**Exemptions:**
Infrastructure and aggregate alerts that do not target a specific VM
are exempt:

- `VirtLauncherPodsStuckFailed` (cluster-wide virt-launcher pod count)
- `OrphanedVirtualMachineInstances` (virt-handler infrastructure)
- `OutdatedVirtualMachineInstanceWorkloads` (aggregate count of
  outdated workloads)

### Guest driver build comparison

`OutdatedGuestDeviceDrivers` fires after one hour when the installed driver
build is lower than the highest configured build for the same PCI device ID
and `guestOSVersionID`. For virtio-win versions such as `100.103.104.30200`,
only the final component (`30200`) determines build ordering. The guest Windows
release selects the reference; the kernel, RHEL certification and internal
number components do not participate in the numeric comparison. Keep the full
final component, so revisions such as `30201` are newer than `30200`.

Equal or newer builds do not alert. For example, Balloon build `27100` can be
current for Windows 10 while Windows 11 requires `29400`. If several versions
are configured for one device and Windows release, the highest build is used;
a lower listed build can therefore alert. A device or Windows release with no
comparable reference produces no alert.

The collectors expose installed and configured builds through
`kubevirt_vmi_guest_device_driver_build_number` and
`kubevirt_guest_device_driver_latest_build_number`. They retain the original
version labels and existing version-info and timestamp metrics. Only versions
with four unsigned 16-bit decimal components produce build metrics; unfamiliar
formats are omitted from comparison rather than interpreted as build zero.
The ConfigMap schema remains `v1alpha1`.

The alert now uses
`vmi:kubevirt_vmi_guest_device_driver_build_number:max` instead of the previous
timestamp recording rule. Update explicit recording-rule allowlists or queries
that reference that old name. The `guest-drivers` group selects the new metrics
and rule automatically.
