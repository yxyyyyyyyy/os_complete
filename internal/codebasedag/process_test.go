package codebasedag

import (
	"context"
	"errors"
	"runtime"
	"testing"
)

func TestInterfaceProcessRuntimeLifecycle(t *testing.T) {
	driver := &fakeProcessDriver{}
	runtime := NewInterfaceProcessRuntime(driver)
	result, err := runtime.StartPrepared(context.Background(), ProcessConfig{
		RunID:  "run",
		NodeID: "planner",
		Worker: WorkerSpec{
			Command: "/bin/aort-code-worker",
			Args:    []string{"--node", "planner"},
			Env:     []string{"AORT_TEST=1"},
		},
		Limits: ResourceLimits{MemoryMax: "67108864", PidsMax: "8", CPUMax: "100000 100000"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.EvidenceMode != "test-process" || result.PID != 1234 || result.CgroupPath == "" {
		t.Fatalf("result = %#v", result)
	}
	if driver.started != 1 || driver.attached != 1 || driver.continued != 1 || driver.waited != 1 || driver.cleaned != 1 {
		t.Fatalf("driver counts = %#v", driver)
	}
}

func TestInterfaceProcessRuntimeCleansUpAfterFailure(t *testing.T) {
	driver := &fakeProcessDriver{attachErr: errors.New("attach failed")}
	runtime := NewInterfaceProcessRuntime(driver)
	_, err := runtime.StartPrepared(context.Background(), ProcessConfig{
		RunID: "run", NodeID: "planner", Worker: WorkerSpec{Command: "worker"},
	})
	if !errors.Is(err, driver.attachErr) {
		t.Fatalf("error = %v", err)
	}
	if driver.cleaned != 1 {
		t.Fatalf("cleanup calls = %d", driver.cleaned)
	}
}

func TestDefaultProcessRuntimePortableBehavior(t *testing.T) {
	runtime := NewDefaultProcessRuntime()
	_, err := runtime.StartPrepared(context.Background(), ProcessConfig{RunID: "run", NodeID: "planner", Worker: WorkerSpec{Command: "worker"}})
	if runtimeGOOS := runtimeGOOS(); runtimeGOOS != "linux" {
		if err == nil || !errors.Is(err, ErrProcessUnsupported) {
			t.Fatalf("non-linux should return unsupported, got %v", err)
		}
	} else if err == nil {
		t.Fatal("linux default runtime should still require a concrete process driver in unit tests")
	}
}

func TestValidateResourceLimits(t *testing.T) {
	valid := ResourceLimits{MemoryMax: "67108864", PidsMax: "8", CPUMax: "100000 100000"}
	if err := ValidateResourceLimits(valid); err != nil {
		t.Fatal(err)
	}
	for _, limits := range []ResourceLimits{
		{MemoryMax: "-1", PidsMax: "8", CPUMax: "100000 100000"},
		{MemoryMax: "67108864", PidsMax: "0", CPUMax: "100000 100000"},
		{MemoryMax: "67108864", PidsMax: "8", CPUMax: "0 100000"},
		{MemoryMax: "67108864", PidsMax: "8", CPUMax: "bad"},
	} {
		if err := ValidateResourceLimits(limits); err == nil {
			t.Fatalf("limits should fail: %#v", limits)
		}
	}
}

type fakeProcessDriver struct {
	started   int
	attached  int
	continued int
	waited    int
	cleaned   int
	attachErr error
}

func (d *fakeProcessDriver) StartStopped(context.Context, WorkerSpec) (PreparedWorker, error) {
	d.started++
	return PreparedWorker{PID: 1234, EvidenceMode: "test-process"}, nil
}

func (d *fakeProcessDriver) Attach(context.Context, PreparedWorker, ResourceLimits) (CapsuleAttachment, error) {
	d.attached++
	if d.attachErr != nil {
		return CapsuleAttachment{}, d.attachErr
	}
	return CapsuleAttachment{CgroupPath: "/sys/fs/cgroup/aort.slice/test", EvidenceMode: "test-cgroup"}, nil
}

func (d *fakeProcessDriver) Continue(context.Context, PreparedWorker) error {
	d.continued++
	return nil
}

func (d *fakeProcessDriver) Wait(context.Context, PreparedWorker) (WorkerResult, error) {
	d.waited++
	return WorkerResult{ExitCode: 0, OutputSHA256: "hash", LLMCallID: "call"}, nil
}

func (d *fakeProcessDriver) Cleanup(context.Context, PreparedWorker, CapsuleAttachment) error {
	d.cleaned++
	return nil
}

func runtimeGOOS() string {
	return runtime.GOOS
}
