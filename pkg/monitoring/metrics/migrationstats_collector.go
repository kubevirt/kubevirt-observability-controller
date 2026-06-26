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
	"github.com/rhobs/operator-observability-toolkit/pkg/operatormetrics"
	k6tv1 "kubevirt.io/api/core/v1"
)

var (
	MigrationStatsCollector = operatormetrics.Collector{
		Metrics: []operatormetrics.Metric{
			PendingMigrations,
			SchedulingMigrations,
			UnsetMigration,
			RunningMigrations,
			SucceededMigration,
			FailedMigration,
		},
		CollectCallback: migrationStatsCollectorCallback,
	}

	PendingMigrations = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migrations_in_pending_phase",
			Help: "Number of current pending migrations.",
		},
	)

	SchedulingMigrations = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migrations_in_scheduling_phase",
			Help: "Number of current scheduling migrations.",
		},
	)

	UnsetMigration = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migrations_in_unset_phase",
			Help: "Number of current unset migrations.",
		},
	)

	RunningMigrations = operatormetrics.NewGauge(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migrations_in_running_phase",
			Help: "Number of current running migrations.",
		},
	)

	SucceededMigration = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migration_succeeded",
			Help: "Indicates if the VMI migration succeeded.",
		},
		[]string{"vmi", "vmim", "namespace"},
	)

	FailedMigration = operatormetrics.NewGaugeVec(
		operatormetrics.MetricOpts{
			Name: "kubevirt_vmi_migration_failed",
			Help: "Indicates if the VMI migration failed.",
		},
		[]string{"vmi", "vmim", "namespace"},
	)
)

func migrationStatsCollectorCallback() []operatormetrics.CollectorResult {
	idx := getIndexers()
	if idx == nil || idx.VMIMigration == nil {
		return []operatormetrics.CollectorResult{}
	}

	cachedObjs := idx.VMIMigration.List()
	vmims := make([]*k6tv1.VirtualMachineInstanceMigration, len(cachedObjs))
	for i, obj := range cachedObjs {
		vmims[i] = obj.(*k6tv1.VirtualMachineInstanceMigration)
	}

	return ReportMigrationStats(vmims)
}

func ReportMigrationStats(
	vmims []*k6tv1.VirtualMachineInstanceMigration,
) []operatormetrics.CollectorResult {
	var cr []operatormetrics.CollectorResult

	pendingCount := 0
	schedulingCount := 0
	unsetCount := 0
	runningCount := 0

	for _, vmim := range vmims {
		switch vmim.Status.Phase {
		case k6tv1.MigrationPending:
			pendingCount++
		case k6tv1.MigrationScheduling:
			schedulingCount++
		case k6tv1.MigrationPhaseUnset:
			unsetCount++
		case k6tv1.MigrationRunning, k6tv1.MigrationScheduled,
			k6tv1.MigrationPreparingTarget, k6tv1.MigrationTargetReady:
			runningCount++
		case k6tv1.MigrationSucceeded:
			cr = append(cr, operatormetrics.CollectorResult{
				Metric: SucceededMigration, Value: 1,
				Labels: []string{vmim.Spec.VMIName, vmim.Name, vmim.Namespace},
			})
		case k6tv1.MigrationFailed:
			cr = append(cr, operatormetrics.CollectorResult{
				Metric: FailedMigration, Value: 1,
				Labels: []string{vmim.Spec.VMIName, vmim.Name, vmim.Namespace},
			})
		}
	}

	return append(cr,
		operatormetrics.CollectorResult{
			Metric: PendingMigrations, Value: float64(pendingCount),
		},
		operatormetrics.CollectorResult{
			Metric: SchedulingMigrations, Value: float64(schedulingCount),
		},
		operatormetrics.CollectorResult{
			Metric: UnsetMigration, Value: float64(unsetCount),
		},
		operatormetrics.CollectorResult{
			Metric: RunningMigrations, Value: float64(runningCount),
		},
	)
}
