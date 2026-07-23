//go:build linux

package codebasedag

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLinuxCgroupDriverLifecycleWithFakeFS(t *testing.T) {
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
	driver.now = func() time.Time { return time.Unix(100, 0).UTC() }

	rt := NewInterfaceProcessRuntime(driver)
	result, err := rt.StartPrepared(context.Background(), ProcessConfig{
		RunID:  "run-linux",
		NodeID: "resource-coder",
		Worker: WorkerSpec{Command: "./bin/aort-code-worker"},
		Limits: DefaultResourceLimits(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.PID <= 0 || result.CgroupPath == "" || result.CgroupEvidenceMode != "real-cgroup-v2" {
		t.Fatalf("result=%#v", result)
	}
	if _, err := os.Stat(filepath.Dir(result.CgroupPath)); err != nil {
		t.Fatal(err)
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
