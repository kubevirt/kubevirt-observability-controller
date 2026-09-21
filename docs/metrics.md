# KubeVirt Observability Controller Metrics and Recording Rules

| Name | Kind | Type | Description |
|------|------|------|-------------|
| kubevirt_vm_create_date_timestamp_seconds | Metric | Gauge | Virtual Machine creation timestamp. |
| kubevirt_vm_disk_allocated_size_bytes | Metric | Gauge | Allocated disk size of a Virtual Machine in bytes. |
| kubevirt_vm_error_status_last_transition_timestamp_seconds | Metric | Counter | Virtual Machine last transition timestamp to error status. |
| kubevirt_vm_info | Metric | Gauge | Information about Virtual Machines. |
| kubevirt_vm_labels | Metric | Gauge | The metric exposes the VM labels as Prometheus labels. |
| kubevirt_vm_migrating_status_last_transition_timestamp_seconds | Metric | Counter | Virtual Machine last transition timestamp to migrating status. |
| kubevirt_vm_non_running_status_last_transition_timestamp_seconds | Metric | Counter | Virtual Machine last transition timestamp to paused/stopped status. |
| kubevirt_vm_resource_limits | Metric | Gauge | Resource limits set for a Virtual Machine. |
| kubevirt_vm_resource_requests | Metric | Gauge | Resources requested by Virtual Machine. |
| kubevirt_vm_running_status_last_transition_timestamp_seconds | Metric | Counter | Virtual Machine last transition timestamp to running status. |
| kubevirt_vm_starting_status_last_transition_timestamp_seconds | Metric | Counter | Virtual Machine last transition timestamp to starting status. |
| kubevirt_vm_vnic_info | Metric | Gauge | Details of Virtual Machine vNIC interfaces. |
| kubevirt_vmi_contains_ephemeral_hotplug_volume | Metric | Gauge | Reported only for VMIs that contain an ephemeral hotplug volume. |
| kubevirt_vmi_cpu_system_usage_seconds_total | Metric | Counter | Total CPU time spent in system mode. |
| kubevirt_vmi_cpu_usage_seconds_total | Metric | Counter | Total CPU time spent in all modes (sum of both vcpu and hypervisor usage). |
| kubevirt_vmi_cpu_user_usage_seconds_total | Metric | Counter | Total CPU time spent in user mode. |
| kubevirt_vmi_dirty_rate_bytes_per_second | Metric | Gauge | Guest dirty-rate in bytes per second. |
| kubevirt_vmi_guest_device_driver_date_seconds | Metric | Gauge | Release date of the driver of a guest device, in seconds since the epoch, as reported by the guest agent. |
| kubevirt_vmi_guest_disk_total_bytes | Metric | Gauge | Total disk size in bytes as reported by the guest agent. |
| kubevirt_vmi_guest_disk_used_bytes | Metric | Gauge | Used disk size in bytes as reported by the guest agent. |
| kubevirt_vmi_guest_hostname | Metric | Gauge | Guest hostname from the guest agent. |
| kubevirt_vmi_guest_interface_info | Metric | Gauge | Guest network interface information from the guest agent. |
| kubevirt_vmi_guest_load_15m | Metric | Gauge | Guest system load average over 15 minutes. |
| kubevirt_vmi_guest_load_1m | Metric | Gauge | Guest system load average over 1 minute. |
| kubevirt_vmi_guest_load_5m | Metric | Gauge | Guest system load average over 5 minutes. |
| kubevirt_vmi_guest_os_info | Metric | Gauge | Guest OS information from the guest agent. |
| kubevirt_vmi_guest_timezone | Metric | Gauge | Guest timezone from the guest agent. |
| kubevirt_vmi_guest_user_count | Metric | Gauge | Number of logged-in users in the guest. |
| kubevirt_vmi_info | Metric | Gauge | Information about VirtualMachineInstances. |
| kubevirt_vmi_launcher_memory_overhead_bytes | Metric | Gauge | Estimation of the memory amount required for virt-launcher's infrastructure components. |
| kubevirt_vmi_memory_actual_balloon_bytes | Metric | Gauge | Current balloon size in bytes. |
| kubevirt_vmi_memory_available_bytes | Metric | Gauge | Amount of usable memory as seen by the domain. |
| kubevirt_vmi_memory_cached_bytes | Metric | Gauge | The amount of memory that is being used to cache I/O and is available to be reclaimed. |
| kubevirt_vmi_memory_domain_bytes | Metric | Gauge | The amount of memory in bytes allocated to the domain. |
| kubevirt_vmi_memory_pgmajfault_total | Metric | Counter | The number of page faults when disk IO was required. |
| kubevirt_vmi_memory_pgminfault_total | Metric | Counter | The number of other page faults, when disk IO was not required. |
| kubevirt_vmi_memory_resident_bytes | Metric | Gauge | Resident set size of the process running the domain. |
| kubevirt_vmi_memory_swap_in_traffic_bytes | Metric | Gauge | The total amount of data read from swap space of the guest in bytes. |
| kubevirt_vmi_memory_swap_out_traffic_bytes | Metric | Gauge | The total amount of memory written out to swap space of the guest in bytes. |
| kubevirt_vmi_memory_unused_bytes | Metric | Gauge | The amount of memory left completely unused by the system. |
| kubevirt_vmi_memory_usable_bytes | Metric | Gauge | The amount of memory which can be reclaimed by balloon without pushing the guest system to swap. |
| kubevirt_vmi_migration_end_time_seconds | Metric | Gauge | The time at which the migration ended. |
| kubevirt_vmi_migration_failed | Metric | Gauge | Indicates if the VMI migration failed. |
| kubevirt_vmi_migration_start_time_seconds | Metric | Gauge | The time at which the migration started. |
| kubevirt_vmi_migration_succeeded | Metric | Gauge | Indicates if the VMI migration succeeded. |
| kubevirt_vmi_migrations_in_pending_phase | Metric | Gauge | Number of current pending migrations. |
| kubevirt_vmi_migrations_in_running_phase | Metric | Gauge | Number of current running migrations. |
| kubevirt_vmi_migrations_in_scheduling_phase | Metric | Gauge | Number of current scheduling migrations. |
| kubevirt_vmi_migrations_in_unset_phase | Metric | Gauge | Number of current unset migrations. |
| kubevirt_vmi_network_receive_bytes_total | Metric | Counter | Total network traffic received in bytes. |
| kubevirt_vmi_network_receive_errors_total | Metric | Counter | Total network received error packets. |
| kubevirt_vmi_network_receive_packets_dropped_total | Metric | Counter | The total number of rx packets dropped on vNIC interfaces. |
| kubevirt_vmi_network_receive_packets_total | Metric | Counter | Total network traffic received packets. |
| kubevirt_vmi_network_transmit_bytes_total | Metric | Counter | Total network traffic transmitted in bytes. |
| kubevirt_vmi_network_transmit_errors_total | Metric | Counter | Total network transmitted error packets. |
| kubevirt_vmi_network_transmit_packets_dropped_total | Metric | Counter | The total number of tx packets dropped on vNIC interfaces. |
| kubevirt_vmi_network_transmit_packets_total | Metric | Counter | Total network traffic transmitted packets. |
| kubevirt_vmi_non_evictable | Metric | Gauge | Indication for a VirtualMachine that its eviction strategy is set to Live Migration but is not migratable. |
| kubevirt_vmi_status_addresses | Metric | Gauge | The addresses of a VirtualMachineInstance. |
| kubevirt_vmi_storage_flush_requests_total | Metric | Counter | Total storage flush requests. |
| kubevirt_vmi_storage_flush_times_seconds_total | Metric | Counter | Total time spent on cache flushing. |
| kubevirt_vmi_storage_iops_read_total | Metric | Counter | Total number of I/O read operations. |
| kubevirt_vmi_storage_iops_write_total | Metric | Counter | Total number of I/O write operations. |
| kubevirt_vmi_storage_read_times_seconds_total | Metric | Counter | Total time spent on read operations. |
| kubevirt_vmi_storage_read_traffic_bytes_total | Metric | Counter | Total number of bytes read from storage. |
| kubevirt_vmi_storage_write_times_seconds_total | Metric | Counter | Total time spent on write operations. |
| kubevirt_vmi_storage_write_traffic_bytes_total | Metric | Counter | Total number of written bytes. |
| kubevirt_vmi_vcpu_delay_seconds_total | Metric | Counter | Amount of time spent by each vcpu waiting in the queue instead of running. |
| kubevirt_vmi_vcpu_seconds_total | Metric | Counter | Total amount of time spent in each state by each vcpu. |
| kubevirt_vmi_vcpu_wait_seconds_total | Metric | Counter | Amount of time spent by each vcpu while waiting on I/O. |
| kubevirt_vmi_vnic_info | Metric | Gauge | Details of VirtualMachineInstance vNIC interfaces. |
| cluster:kubevirt_api_request_deprecated_total:sum | Recording rule | Counter | The total number of requests to deprecated KubeVirt APIs, by API verb (e.g., LIST, WATCH). |
| cluster:kubevirt_nodes_allocatable:count | Recording rule | Gauge | The number of allocatable nodes in the cluster. |
| cluster:kubevirt_nodes_with_kvm:count | Recording rule | Gauge | The number of nodes in the cluster that have the devices.kubevirt.io/kvm resource available. |
| cluster:kubevirt_non_schedulable_nodes:sum | Recording rule | Gauge | The number of non-schedulable nodes in the cluster. |
| cluster:kubevirt_virt_api_up:sum | Recording rule | Gauge | The number of virt-api pods that are up. |
| cluster:kubevirt_virt_controller_pods_running:count | Recording rule | Gauge | The number of virt-controller pods that are running. |
| cluster:kubevirt_virt_controller_ready:sum | Recording rule | Gauge | The number of virt-controller pods that are ready. |
| cluster:kubevirt_virt_controller_up:sum | Recording rule | Gauge | The number of virt-controller pods that are up. |
| cluster:kubevirt_virt_handler_up:sum | Recording rule | Gauge | The number of virt-handler pods that are up. |
| cluster:kubevirt_virt_operator_leading:sum | Recording rule | Gauge | The number of virt-operator pods that are leading. |
| cluster:kubevirt_virt_operator_pods_running:count | Recording rule | Gauge | The number of virt-operator pods that are running. |
| cluster:kubevirt_virt_operator_ready:sum | Recording rule | Gauge | The number of virt-operator pods that are ready. |
| cluster:kubevirt_virt_operator_up:sum | Recording rule | Gauge | The number of virt-operator pods that are up. |
| container:kubevirt_memory_delta_from_requested_bytes:max | Recording rule | Gauge | The delta between the pod with highest memory working set or rss and its requested memory for each container, virt-controller, virt-handler, virt-api, virt-operator and compute(virt-launcher). |
| kubevirt_allocatable_nodes | Recording rule | Gauge | [Deprecated] Replaced by cluster:kubevirt_nodes_allocatable:count. |
| kubevirt_api_request_deprecated_total | Recording rule | Counter | [Deprecated] Replaced by cluster:kubevirt_api_request_deprecated_total:sum. |
| kubevirt_memory_delta_from_requested_bytes | Recording rule | Gauge | [Deprecated] Replaced by container:kubevirt_memory_delta_from_requested_bytes:max. |
| kubevirt_nodes_with_kvm | Recording rule | Gauge | [Deprecated] Replaced by cluster:kubevirt_nodes_with_kvm:count. |
| kubevirt_number_of_vms | Recording rule | Gauge | [Deprecated] Replaced by namespace:kubevirt_vm:sum. |
| kubevirt_virt_api_up | Recording rule | Gauge | [Deprecated] Replaced by cluster:kubevirt_virt_api_up:sum. |
| kubevirt_virt_controller_ready | Recording rule | Gauge | [Deprecated] Replaced by cluster:kubevirt_virt_controller_ready:sum. |
| kubevirt_virt_controller_up | Recording rule | Gauge | [Deprecated] Replaced by cluster:kubevirt_virt_controller_up:sum. |
| kubevirt_virt_handler_up | Recording rule | Gauge | [Deprecated] Replaced by cluster:kubevirt_virt_handler_up:sum. |
| kubevirt_virt_operator_leading | Recording rule | Gauge | [Deprecated] Replaced by cluster:kubevirt_virt_operator_leading:sum. |
| kubevirt_virt_operator_ready | Recording rule | Gauge | [Deprecated] Replaced by cluster:kubevirt_virt_operator_ready:sum. |
| kubevirt_virt_operator_up | Recording rule | Gauge | [Deprecated] Replaced by cluster:kubevirt_virt_operator_up:sum. |
| kubevirt_vm_container_memory_request_margin_based_on_rss_bytes | Recording rule | Gauge | [Deprecated] Replaced by pod_container:kubevirt_vm_memory_request_margin_based_on_rss_bytes:sum. |
| kubevirt_vm_container_memory_request_margin_based_on_working_set_bytes | Recording rule | Gauge | [Deprecated] Replaced by pod_container:kubevirt_vm_memory_request_margin_based_on_working_set_bytes:sum. |
| kubevirt_vm_created_total | Recording rule | Counter | [Deprecated] The total number of VMs created by namespace, since install. |
| kubevirt_vmi_guest_vcpu_queue | Recording rule | Gauge | [Deprecated] Replaced by vmi:kubevirt_vmi_guest_queue_length:sum. |
| kubevirt_vmi_memory_used_bytes | Recording rule | Gauge | [Deprecated] Replaced by vmi:kubevirt_vmi_memory_used_bytes:sum. |
| kubevirt_vmi_migration_data_total_bytes | Recording rule | Counter | [Deprecated] Replaced by kubevirt_vmi_migration_data_bytes_total. |
| kubevirt_vmi_phase_count | Recording rule | Gauge | [Deprecated] Replaced by node:kubevirt_vmi_phase:sum. |
| kubevirt_vmsnapshot_disks_restored_from_source | Recording rule | Gauge | [Deprecated] Replaced by vm:kubevirt_vmsnapshot_disks_restored:sum. |
| kubevirt_vmsnapshot_disks_restored_from_source_bytes | Recording rule | Gauge | [Deprecated] Replaced by vm:kubevirt_vmsnapshot_restored_bytes:sum. |
| kubevirt_vmsnapshot_persistentvolumeclaim_labels | Recording rule | Gauge | [Deprecated] Replaced by pvc:kubevirt_vmsnapshot_labels:info. |
| namespace:kubevirt_vm:sum | Recording rule | Gauge | The number of VMs in the cluster by namespace. |
| node:kubevirt_vmi_phase:sum | Recording rule | Gauge | Sum of VMIs per phase and node. `phase` can be one of the following: [`Pending`, `Scheduling`, `Scheduled`, `Running`, `Succeeded`, `Failed`, `Unknown`]. |
| pod_container:kubevirt_vm_memory_request_margin_based_on_rss_bytes:sum | Recording rule | Gauge | Difference between requested memory and rss for VM containers (request margin). Can be negative when usage exceeds request. |
| pod_container:kubevirt_vm_memory_request_margin_based_on_working_set_bytes:sum | Recording rule | Gauge | Difference between requested memory and working set for VM containers (request margin). Can be negative when usage exceeds request. |
| pvc:kubevirt_vmsnapshot_labels:info | Recording rule | Gauge | Returns the labels of the persistent volume claims that are used for restoring virtual machines. |
| vm:kubevirt_vmsnapshot_disks_restored:sum | Recording rule | Gauge | Returns the total number of virtual machine disks restored from the source virtual machine. |
| vm:kubevirt_vmsnapshot_restored_bytes:sum | Recording rule | Gauge | Returns the amount of space in bytes restored from the source virtual machine. |
| vmi:kubevirt_vmi_guest_queue_length:sum | Recording rule | Gauge | Guest queue length. |
| vmi:kubevirt_vmi_memory_available_bytes:sum | Recording rule | Gauge | Sum of available memory bytes per VMI (aggregated by name, namespace). |
| vmi:kubevirt_vmi_memory_headroom_ratio:sum | Recording rule | Gauge | Usable memory to available memory ratio per VMI (aggregated by name, namespace). |
| vmi:kubevirt_vmi_memory_used_bytes:sum | Recording rule | Gauge | Amount of `used` memory as seen by the domain. |
| vmi:kubevirt_vmi_pgmajfaults:rate30m | Recording rule | Gauge | Rate of major page faults over 30 minutes per VMI (aggregated by name, namespace). |
| vmi:kubevirt_vmi_pgmajfaults:rate5m | Recording rule | Gauge | Rate of major page faults over 5 minutes per VMI (aggregated by name, namespace). |
| vmi:kubevirt_vmi_swap_traffic_bytes:rate30m | Recording rule | Gauge | Total swap I/O traffic rate over 30 minutes per VMI (swap in + swap out, aggregated by name, namespace). |
| vmi:kubevirt_vmi_swap_traffic_bytes:rate5m | Recording rule | Gauge | Total swap I/O traffic rate over 5 minutes per VMI (swap in + swap out, aggregated by name, namespace). |
| vmi:kubevirt_vmi_vcpu:count | Recording rule | Gauge | The number of the VMI vCPUs. |

## Developing new metrics

All metrics documented here are auto-generated and reflect exactly what is being
exposed. After developing new metrics or changing old ones please regenerate
this document.
