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

package vmstats

const (
	VCPUOffline = 0
	VCPURunning = 1
	VCPUBlocked = 2
)

type VMStatsResult struct {
	Stats *VMStats `json:"stats"`
	Error string   `json:"error"`
}

type VMStats struct {
	DomainStats               DomainStats `json:"DomainStats"`
	DirtyRateMbps             *int64      `json:"DirtyRateMbps"`
	GuestAgentVersion         string      `json:"GuestAgentVersion"`
	GuestGetLoad              string      `json:"GuestGetLoad"`
	GuestGetCpuStats          string      `json:"GuestGetCpuStats"`
	GuestGetDiskStats         string      `json:"GuestGetDiskStats"`
	GuestGetTime              string      `json:"GuestGetTime"`
	GuestGetVcpus             string      `json:"GuestGetVcpus"`
	GuestGetMemoryBlockInfo   string      `json:"GuestGetMemoryBlockInfo"`
	GuestGetUsers             string      `json:"GuestGetUsers"`
	GuestGetOsInfo            string      `json:"GuestGetOsInfo"`
	GuestGetDisks             string      `json:"GuestGetDisks"`
	GuestGetHostName          string      `json:"GuestGetHostName"`
	GuestGetTimezone          string      `json:"GuestGetTimezone"`
	GuestNetworkGetRoute      string      `json:"GuestNetworkGetRoute"`
	GuestNetworkGetInterfaces string      `json:"GuestNetworkGetInterfaces"`
	GuestGetMemoryBlocks      string      `json:"GuestGetMemoryBlocks"`
}

type DomainStats struct {
	Name                 string                `json:"Name"`
	UUID                 string                `json:"UUID"`
	Cpu                  *DomainStatsCPU       `json:"Cpu"`
	Memory               *DomainStatsMemory    `json:"Memory"`
	MigrateDomainJobInfo *DomainJobInfo        `json:"MigrateDomainJobInfo"`
	Vcpu                 []DomainStatsVcpu     `json:"Vcpu"`
	Net                  []DomainStatsNet      `json:"Net"`
	Block                []DomainStatsBlock    `json:"Block"`
	CPUMapSet            bool                  `json:"CPUMapSet"`
	CPUMap               [][]bool              `json:"CPUMap"`
	DirtyRate            *DomainStatsDirtyRate `json:"DirtyRate"`
	Load                 *DomainStatsLoad      `json:"Load"`
}

type DomainStatsLoad struct {
	Load1mSet  bool    `json:"Load1mSet"`
	Load1m     float64 `json:"Load1m"`
	Load5mSet  bool    `json:"Load5mSet"`
	Load5m     float64 `json:"Load5m"`
	Load15mSet bool    `json:"Load15mSet"`
	Load15m    float64 `json:"Load15m"`
}

type DomainStatsCPU struct {
	TimeSet   bool   `json:"TimeSet"`
	Time      uint64 `json:"Time"`
	UserSet   bool   `json:"UserSet"`
	User      uint64 `json:"User"`
	SystemSet bool   `json:"SystemSet"`
	System    uint64 `json:"System"`
}

type DomainStatsVcpu struct {
	StateSet bool   `json:"StateSet"`
	State    int    `json:"State"`
	TimeSet  bool   `json:"TimeSet"`
	Time     uint64 `json:"Time"`
	WaitSet  bool   `json:"WaitSet"`
	Wait     uint64 `json:"Wait"`
	DelaySet bool   `json:"DelaySet"`
	Delay    uint64 `json:"Delay"`
}

type DomainStatsNet struct {
	NameSet    bool   `json:"NameSet"`
	Name       string `json:"Name"`
	AliasSet   bool   `json:"AliasSet"`
	Alias      string `json:"Alias"`
	RxBytesSet bool   `json:"RxBytesSet"`
	RxBytes    uint64 `json:"RxBytes"`
	RxPktsSet  bool   `json:"RxPktsSet"`
	RxPkts     uint64 `json:"RxPkts"`
	RxErrsSet  bool   `json:"RxErrsSet"`
	RxErrs     uint64 `json:"RxErrs"`
	RxDropSet  bool   `json:"RxDropSet"`
	RxDrop     uint64 `json:"RxDrop"`
	TxBytesSet bool   `json:"TxBytesSet"`
	TxBytes    uint64 `json:"TxBytes"`
	TxPktsSet  bool   `json:"TxPktsSet"`
	TxPkts     uint64 `json:"TxPkts"`
	TxErrsSet  bool   `json:"TxErrsSet"`
	TxErrs     uint64 `json:"TxErrs"`
	TxDropSet  bool   `json:"TxDropSet"`
	TxDrop     uint64 `json:"TxDrop"`
}

type DomainStatsBlock struct {
	NameSet         bool   `json:"NameSet"`
	Name            string `json:"Name"`
	Alias           string `json:"Alias"`
	BackingIndexSet bool   `json:"BackingIndexSet"`
	BackingIndex    uint   `json:"BackingIndex"`
	PathSet         bool   `json:"PathSet"`
	Path            string `json:"Path"`
	RdReqsSet       bool   `json:"RdReqsSet"`
	RdReqs          uint64 `json:"RdReqs"`
	RdBytesSet      bool   `json:"RdBytesSet"`
	RdBytes         uint64 `json:"RdBytes"`
	RdTimesSet      bool   `json:"RdTimesSet"`
	RdTimes         uint64 `json:"RdTimes"`
	WrReqsSet       bool   `json:"WrReqsSet"`
	WrReqs          uint64 `json:"WrReqs"`
	WrBytesSet      bool   `json:"WrBytesSet"`
	WrBytes         uint64 `json:"WrBytes"`
	WrTimesSet      bool   `json:"WrTimesSet"`
	WrTimes         uint64 `json:"WrTimes"`
	FlReqsSet       bool   `json:"FlReqsSet"`
	FlReqs          uint64 `json:"FlReqs"`
	FlTimesSet      bool   `json:"FlTimesSet"`
	FlTimes         uint64 `json:"FlTimes"`
	ErrorsSet       bool   `json:"ErrorsSet"`
	Errors          uint64 `json:"Errors"`
	AllocationSet   bool   `json:"AllocationSet"`
	Allocation      uint64 `json:"Allocation"`
	CapacitySet     bool   `json:"CapacitySet"`
	Capacity        uint64 `json:"Capacity"`
	PhysicalSet     bool   `json:"PhysicalSet"`
	Physical        uint64 `json:"Physical"`
}

type DomainStatsMemory struct {
	UnusedSet        bool   `json:"UnusedSet"`
	Unused           uint64 `json:"Unused"`
	CachedSet        bool   `json:"CachedSet"`
	Cached           uint64 `json:"Cached"`
	AvailableSet     bool   `json:"AvailableSet"`
	Available        uint64 `json:"Available"`
	ActualBalloonSet bool   `json:"ActualBalloonSet"`
	ActualBalloon    uint64 `json:"ActualBalloon"`
	RSSSet           bool   `json:"RSSSet"`
	RSS              uint64 `json:"RSS"`
	SwapInSet        bool   `json:"SwapInSet"`
	SwapIn           uint64 `json:"SwapIn"`
	SwapOutSet       bool   `json:"SwapOutSet"`
	SwapOut          uint64 `json:"SwapOut"`
	MajorFaultSet    bool   `json:"MajorFaultSet"`
	MajorFault       uint64 `json:"MajorFault"`
	MinorFaultSet    bool   `json:"MinorFaultSet"`
	MinorFault       uint64 `json:"MinorFault"`
	UsableSet        bool   `json:"UsableSet"`
	Usable           uint64 `json:"Usable"`
	TotalSet         bool   `json:"TotalSet"`
	Total            uint64 `json:"Total"`
}

type DomainJobInfo struct {
	DataTotalSet     bool   `json:"DataTotalSet"`
	DataTotal        uint64 `json:"DataTotal"`
	DataProcessedSet bool   `json:"DataProcessedSet"`
	DataProcessed    uint64 `json:"DataProcessed"`
	MemoryBpsSet     bool   `json:"MemoryBpsSet"`
	MemoryBps        uint64 `json:"MemoryBps"`
	DataRemainingSet bool   `json:"DataRemainingSet"`
	DataRemaining    uint64 `json:"DataRemaining"`
	MemDirtyRateSet  bool   `json:"MemDirtyRateSet"`
	MemDirtyRate     uint64 `json:"MemDirtyRate"`
}

type DomainStatsDirtyRate struct {
	CalcStatusSet         bool  `json:"CalcStatusSet"`
	CalcStatus            int   `json:"CalcStatus"`
	CalcStartTimeSet      bool  `json:"CalcStartTimeSet"`
	CalcStartTime         int64 `json:"CalcStartTime"`
	CalcPeriodSet         bool  `json:"CalcPeriodSet"`
	CalcPeriod            int   `json:"CalcPeriod"`
	MegabytesPerSecondSet bool  `json:"MegabytesPerSecondSet"`
	MegabytesPerSecond    int64 `json:"MegabytesPerSecond"`
}
