package evidence

import "testing"

func TestEvidenceModesAreEnumeratedAndValidated(t *testing.T) {
	want := []Mode{
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

	for _, mode := range want {
		if !IsValid(mode) {
			t.Fatalf("mode %q should be valid", mode)
		}
	}
	if IsValid("degraded-proxy") {
		t.Fatalf("degraded-proxy is not part of the competition evidence mode vocabulary")
	}
}

func TestEvidenceModeSummaryUsesHonestBoundaries(t *testing.T) {
	summary := CompetitionSummary()
	if summary["cgroup_capsule"] != string(ModeRealCgroupV2) && summary["cgroup_capsule"] != string(ModeDegraded) {
		t.Fatalf("unexpected cgroup mode: %#v", summary)
	}
	if summary["worker_process"] != string(ModeRealRuntime) {
		t.Fatalf("worker process should be real-runtime: %#v", summary)
	}
	if summary["cvm"] != string(ModeRealPartial) {
		t.Fatalf("CVM should be real-partial, not full KV cache sharing: %#v", summary)
	}
	if summary["ipc"] != string(ModeRealPartial) {
		t.Fatalf("IPC should be real-partial page-reference IPC: %#v", summary)
	}
	if summary["llm"] != string(ModeMock) {
		t.Fatalf("default LLM should be mock: %#v", summary)
	}
	if summary["ebpf"] != string(ModePlanned) && summary["ebpf"] != string(ModeDegraded) {
		t.Fatalf("eBPF should be planned/degraded without attachment: %#v", summary)
	}
	if summary["ipc_shm"] != string(ModePlanned) && summary["ipc_shm"] != string(ModeRealShmIPC) && summary["ipc_shm"] != string(ModeDegraded) {
		t.Fatalf("shared-memory IPC should be planned/real-shm-ipc/degraded: %#v", summary)
	}
	if summary["replay"] != string(ModeRealRuntime) {
		t.Fatalf("replay should be real-runtime deterministic replay: %#v", summary)
	}
	if summary["overlayfs"] != string(ModeRealOverlayFS) && summary["overlayfs"] != string(ModeDegradedCopy) {
		t.Fatalf("overlayfs should be real-overlayfs or degraded-copy: %#v", summary)
	}
}
