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
