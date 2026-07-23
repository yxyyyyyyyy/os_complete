//go:build linux

package codebasedag

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

func TestLinuxCgroupDriverUsesRealPID(t *testing.T) {
	root := t.TempDir()
	driver, err := NewLinuxCgroupDriver(root)
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{}
	driver.writeFn = func(path, data string) error {
		files[path] = data
		return os.WriteFile(path, []byte(data), 0o644)
	}
	driver.readFn = func(path string) (string, error) {
		if v, ok := files[path]; ok {
			return v, nil
		}
		return "0", nil
	}
	driver.mkdirFn = os.MkdirAll
	driver.killFn = syscall.Kill

	rt := NewInterfaceProcessRuntime(driver)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := rt.StartPrepared(ctx, ProcessConfig{
		RunID:  "run-linux",
		NodeID: "resource-coder",
		Worker: WorkerSpec{Command: "sleep", Args: []string{"0.2"}},
		Limits: DefaultResourceLimits(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.PID <= 1 {
		t.Fatalf("expected real kernel pid, got %#v", result)
	}
	if result.ProcessEvidenceMode != "real-process" || result.CgroupEvidenceMode != "real-cgroup-v2" {
		t.Fatalf("evidence modes = %#v", result)
	}
	if result.CgroupPath == "" {
		t.Fatal("missing cgroup path")
	}
}

func TestLinuxCgroupDriverTimeoutKillsProcess(t *testing.T) {
	root := t.TempDir()
	driver, err := NewLinuxCgroupDriver(root)
	if err != nil {
		t.Fatal(err)
	}
	driver.writeFn = func(path, data string) error {
		return os.WriteFile(path, []byte(data), 0o644)
	}
	driver.readFn = func(string) (string, error) { return "0", nil }
	driver.mkdirFn = os.MkdirAll
	driver.killFn = syscall.Kill

	rt := NewInterfaceProcessRuntime(driver)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err = rt.StartPrepared(ctx, ProcessConfig{
		RunID:  "run-timeout",
		NodeID: "tester",
		Worker: WorkerSpec{Command: "sleep", Args: []string{"10"}},
		Limits: DefaultResourceLimits(),
	})
	if err == nil {
		t.Fatal("expected timeout/kill failure")
	}
}

func TestLinuxCgroupDriverValidatesInputs(t *testing.T) {
	driver, err := NewLinuxCgroupDriver(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := driver.StartStopped(context.Background(), WorkerSpec{}); err == nil {
		t.Fatal("empty command")
	}
	if _, err := driver.Attach(context.Background(), PreparedWorker{PID: 1}, DefaultResourceLimits()); err == nil {
		t.Fatal("unknown pid")
	}
}

func TestLinuxCgroupDriverDegradesWhenCgroupWriteFails(t *testing.T) {
	root := t.TempDir()
	driver, err := NewLinuxCgroupDriver(root)
	if err != nil {
		t.Fatal(err)
	}
	driver.writeFn = func(path, data string) error {
		if filepath.Base(path) == "memory.max" {
			return os.ErrPermission
		}
		return os.WriteFile(path, []byte(data), 0o644)
	}
	driver.readFn = func(string) (string, error) { return "0", nil }
	driver.mkdirFn = os.MkdirAll
	driver.killFn = syscall.Kill

	rt := NewInterfaceProcessRuntime(driver)
	result, err := rt.StartPrepared(context.Background(), ProcessConfig{
		RunID:  "run-degraded",
		NodeID: "coder",
		Worker: WorkerSpec{Command: "true"},
		Limits: DefaultResourceLimits(),
	})
	if err == nil {
		t.Fatal("expected attach failure")
	}
	if result.EvidenceMode != "degraded" {
		t.Fatalf("want degraded evidence, got %#v", result)
	}
}

func TestStartStoppedCommandProducesLivePID(t *testing.T) {
	cmd, err := startStoppedCommand(WorkerSpec{Command: "sleep", Args: []string{"1"}})
	if err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	if pid <= 1 {
		t.Fatalf("pid=%d", pid)
	}
	if _, err := os.Stat(filepath.Join("/proc", strconv.Itoa(pid))); err != nil {
		t.Fatalf("pid %d not present in /proc: %v", pid, err)
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
	_, _ = cmd.Process.Wait()
}

func TestLinuxCgroupCleanupRemovesNodeDir(t *testing.T) {
	root := t.TempDir()
	driver, err := NewLinuxCgroupDriver(root)
	if err != nil {
		t.Fatal(err)
	}
	driver.writeFn = func(path, data string) error {
		return os.WriteFile(path, []byte(data), 0o644)
	}
	driver.readFn = func(string) (string, error) { return "0", nil }
	driver.mkdirFn = os.MkdirAll
	driver.killFn = syscall.Kill
	rt := NewInterfaceProcessRuntime(driver)
	result, err := rt.StartPrepared(context.Background(), ProcessConfig{
		RunID: "cleanup-cg", NodeID: "n",
		Worker: WorkerSpec{Command: "true"},
		Limits: DefaultResourceLimits(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.CgroupPath == "" {
		t.Fatal("missing cgroup path")
	}
	if _, err := os.Stat(result.CgroupPath); !os.IsNotExist(err) {
		t.Fatalf("cgroup dir should be cleaned after success, err=%v path=%s", err, result.CgroupPath)
	}
}
