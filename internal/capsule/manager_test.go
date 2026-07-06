package capsule

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestPrepareWritesCgroupFiles(t *testing.T) {
	root := t.TempDir()
	mgr := NewManager(Config{
		Root:          root,
		ForceReal:     true,
		MemoryMax:     "256M",
		PidsMax:       "64",
		CPUMax:        "100000 100000",
		AllowDegraded: false,
	})

	runtime, err := mgr.Prepare("agent-1", 12345)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if runtime.Mode != ModeReal {
		t.Fatalf("mode = %s", runtime.Mode)
	}
	if runtime.CgroupPath != filepath.Join(root, "agent-1") {
		t.Fatalf("cgroup path = %s", runtime.CgroupPath)
	}
	assertFileContains(t, filepath.Join(runtime.CgroupPath, "memory.max"), "256M")
	assertFileContains(t, filepath.Join(runtime.CgroupPath, "pids.max"), "64")
	assertFileContains(t, filepath.Join(runtime.CgroupPath, "cpu.max"), "100000 100000")
	assertFileContains(t, filepath.Join(runtime.CgroupPath, "cgroup.procs"), "12345")
}

func TestPrepareEnablesParentSubtreeControllers(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "cgroup.controllers"), "cpu memory pids io\n")
	writeFile(t, filepath.Join(root, "cgroup.subtree_control"), "\n")
	mgr := NewManager(Config{
		Root:          root,
		ForceReal:     true,
		AllowDegraded: false,
	})

	if _, err := mgr.Prepare("agent-1", 12345); err != nil {
		t.Fatalf("Prepare: %v", err)
	}

	assertFileContains(t, filepath.Join(root, "cgroup.subtree_control"), "+cpu +memory +pids")
}

func TestStatsReadRealCgroupFiles(t *testing.T) {
	root := t.TempDir()
	mgr := NewManager(Config{Root: root, ForceReal: true, AllowDegraded: false})
	runtime, err := mgr.Prepare("agent-1", 12345)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	writeFile(t, filepath.Join(runtime.CgroupPath, "memory.current"), "18350080\n")
	writeFile(t, filepath.Join(runtime.CgroupPath, "pids.current"), "2\n")
	writeFile(t, filepath.Join(runtime.CgroupPath, "cpu.stat"), "usage_usec 123456\nuser_usec 1000\n")
	writeFile(t, filepath.Join(runtime.CgroupPath, "cgroup.events"), "populated 1\nfrozen 1\n")
	writeFile(t, filepath.Join(runtime.CgroupPath, "cgroup.freeze"), "1\n")

	stats := mgr.Stats("agent-1")
	if stats.Mode != ModeReal || stats.MemoryCurrent != 18350080 || stats.PidsCurrent != 2 {
		t.Fatalf("stats = %#v", stats)
	}
	if stats.CPUStat["usage_usec"] != 123456 || stats.Events["populated"] != 1 || !stats.Frozen {
		t.Fatalf("stats maps/frozen = %#v", stats)
	}
}

func TestFreezeAndUnfreezeWriteCgroupFreeze(t *testing.T) {
	root := t.TempDir()
	mgr := NewManager(Config{Root: root, ForceReal: true, AllowDegraded: false})
	runtime, err := mgr.Prepare("agent-1", 12345)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if err := mgr.Freeze("agent-1"); err != nil {
		t.Fatalf("Freeze: %v", err)
	}
	assertFileContains(t, filepath.Join(runtime.CgroupPath, "cgroup.freeze"), "1")
	if err := mgr.Unfreeze("agent-1"); err != nil {
		t.Fatalf("Unfreeze: %v", err)
	}
	assertFileContains(t, filepath.Join(runtime.CgroupPath, "cgroup.freeze"), "0")
}

func TestKillPrefersCgroupKillForRealCapsule(t *testing.T) {
	root := t.TempDir()
	mgr := NewManager(Config{Root: root, ForceReal: true, AllowDegraded: false})
	runtime, err := mgr.Prepare("agent-1", 12345)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}

	result, err := mgr.Kill("agent-1")
	if err != nil {
		t.Fatalf("Kill: %v", err)
	}
	if result.KillMethod != KillMethodCgroupKill {
		t.Fatalf("kill method = %q, want %q", result.KillMethod, KillMethodCgroupKill)
	}
	if result.EvidenceMode != "real-cgroup-v2" {
		t.Fatalf("evidence mode = %q", result.EvidenceMode)
	}
	assertFileContains(t, filepath.Join(runtime.CgroupPath, "cgroup.kill"), "1")
}

func TestKillFallsBackToPidSignalWhenCgroupKillFails(t *testing.T) {
	root := t.TempDir()
	var signaled []syscall.Signal
	mgr := NewManager(Config{
		Root:          root,
		ForceReal:     true,
		AllowDegraded: false,
		SignalFunc: func(pid int, signal syscall.Signal) error {
			if pid != 12345 {
				t.Fatalf("pid = %d, want 12345", pid)
			}
			signaled = append(signaled, signal)
			return nil
		},
		SignalGracePeriod: 0,
	})
	runtime, err := mgr.Prepare("agent-1", 12345)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if err := os.Mkdir(filepath.Join(runtime.CgroupPath, "cgroup.kill"), 0o755); err != nil {
		t.Fatalf("create cgroup.kill directory: %v", err)
	}

	result, err := mgr.Kill("agent-1")
	if err != nil {
		t.Fatalf("Kill: %v", err)
	}
	if result.KillMethod != KillMethodPidSignalFallback {
		t.Fatalf("kill method = %q, want %q", result.KillMethod, KillMethodPidSignalFallback)
	}
	if result.FallbackReason == "" {
		t.Fatalf("fallback reason should explain cgroup.kill failure: %#v", result)
	}
	if len(signaled) != 2 || signaled[0] != syscall.SIGTERM || signaled[1] != syscall.SIGKILL {
		t.Fatalf("signals = %#v", signaled)
	}
}

func TestPrepareReturnsDegradedWhenUnavailable(t *testing.T) {
	root := filepath.Join(t.TempDir(), "not-a-cgroup-dir")
	writeFile(t, root, "file, not directory\n")
	mgr := NewManager(Config{Root: root, ForceReal: true, AllowDegraded: true})
	runtime, err := mgr.Prepare("agent-1", 12345)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if runtime.Mode != ModeDegraded {
		t.Fatalf("mode = %s", runtime.Mode)
	}
	if runtime.Error == "" {
		t.Fatalf("expected degraded error")
	}
}

func assertFileContains(t *testing.T, path string, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	if strings.TrimSpace(string(data)) != want {
		t.Fatalf("%s = %q want %q", path, strings.TrimSpace(string(data)), want)
	}
}

func writeFile(t *testing.T, path string, value string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}
