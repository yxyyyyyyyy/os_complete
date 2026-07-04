package capsule

import (
	"os"
	"path/filepath"
	"strings"
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
	assertFileContains(t, filepath.Join(runtime.CgroupPath, "memory.max"), "256M")
	assertFileContains(t, filepath.Join(runtime.CgroupPath, "pids.max"), "64")
	assertFileContains(t, filepath.Join(runtime.CgroupPath, "cpu.max"), "100000 100000")
	assertFileContains(t, filepath.Join(runtime.CgroupPath, "cgroup.procs"), "12345")
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

func TestPrepareReturnsDegradedWhenUnavailable(t *testing.T) {
	mgr := NewManager(Config{Root: "/definitely/not/a/cgroup", AllowDegraded: true})
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
