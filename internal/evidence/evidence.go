package evidence

import (
	"os"
	"runtime"
)

type Mode string

const (
	ModeReal          Mode = "real"
	ModeRealCgroupV2  Mode = "real-cgroup-v2"
	ModeRealRuntime   Mode = "real-runtime"
	ModeRealAPI       Mode = "real-api"
	ModeRealPartial   Mode = "real-partial"
	ModeRealOverlayFS Mode = "real-overlayfs"
	ModeRealEBPF      Mode = "real-ebpf"
	ModeRealShmIPC    Mode = "real-shm-ipc"
	ModeDegraded      Mode = "degraded"
	ModeDegradedCopy  Mode = "degraded-copy"
	ModeMock          Mode = "mock"
	ModeSimulation    Mode = "simulation"
	ModePlanned       Mode = "planned"
	ModeMissing       Mode = "missing"
)

var validModes = map[Mode]struct{}{
	ModeReal:          {},
	ModeRealCgroupV2:  {},
	ModeRealRuntime:   {},
	ModeRealAPI:       {},
	ModeRealPartial:   {},
	ModeRealOverlayFS: {},
	ModeRealEBPF:      {},
	ModeRealShmIPC:    {},
	ModeDegraded:      {},
	ModeDegradedCopy:  {},
	ModeMock:          {},
	ModeSimulation:    {},
	ModePlanned:       {},
	ModeMissing:       {},
}

func IsValid(mode Mode) bool {
	_, ok := validModes[mode]
	return ok
}

func AllModes() []Mode {
	return []Mode{
		ModeReal,
		ModeRealCgroupV2,
		ModeRealRuntime,
		ModeRealAPI,
		ModeRealPartial,
		ModeRealOverlayFS,
		ModeRealEBPF,
		ModeRealShmIPC,
		ModeDegraded,
		ModeDegradedCopy,
		ModeMock,
		ModeSimulation,
		ModePlanned,
		ModeMissing,
	}
}

func CompetitionSummary() map[string]string {
	return map[string]string{
		"cgroup_capsule": cgroupMode(),
		"worker_process": string(ModeRealRuntime),
		"cvm":            string(ModeRealPartial),
		"ipc":            string(ModeRealPartial),
		"llm":            string(ModeMock),
		"ebpf":           string(ModeDegraded),
		"ipc_shm":        string(ModeDegraded),
		"replay":         string(ModeRealRuntime),
		"overlayfs":      string(ModeDegradedCopy),
	}
}

func cgroupMode() string {
	if runtime.GOOS != "linux" {
		return string(ModeDegraded)
	}
	if _, err := os.Stat("/sys/fs/cgroup/cgroup.controllers"); err != nil {
		return string(ModeDegraded)
	}
	return string(ModeRealCgroupV2)
}
