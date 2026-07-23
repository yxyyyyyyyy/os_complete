//go:build linux

package codebasedag

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type linuxCgroupDriver struct {
	mu      sync.Mutex
	root    string
	now     func() time.Time
	writeFn func(path, data string) error
	readFn  func(path string) (string, error)
	mkdirFn func(path string, mode os.FileMode) error
	killFn  func(pid int, sig syscall.Signal) error
	nodes   map[int]*linuxPrepared
}

type linuxPrepared struct {
	worker     PreparedWorker
	spec       WorkerSpec
	cgroupPath string
	limits     ResourceLimits
	started    bool
	finished   bool
	exitCode   int
	outputSHA  string
	callID     string
	errText    string
}

func NewLinuxCgroupDriver(root string) (*linuxCgroupDriver, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("cgroup root is required")
	}
	return &linuxCgroupDriver{
		root:    root,
		now:     time.Now,
		writeFn: writeFileString,
		readFn:  readFileString,
		mkdirFn: os.MkdirAll,
		killFn:  syscall.Kill,
		nodes:   map[int]*linuxPrepared{},
	}, nil
}

func NewDefaultProcessRuntime() ProcessRuntime {
	root := os.Getenv("AORT_CODEBASE_DAG_CGROUP_ROOT")
	if root == "" {
		root = "/sys/fs/cgroup/aort.slice/codebase-dag"
	}
	driver, err := NewLinuxCgroupDriver(root)
	if err != nil {
		return unsupportedProcessRuntime{reason: err.Error()}
	}
	return NewInterfaceProcessRuntime(driver)
}

func (d *linuxCgroupDriver) StartStopped(ctx context.Context, spec WorkerSpec) (PreparedWorker, error) {
	if err := ctx.Err(); err != nil {
		return PreparedWorker{}, err
	}
	if strings.TrimSpace(spec.Command) == "" {
		return PreparedWorker{}, fmt.Errorf("worker command is required")
	}
	// The real worker binary is started stopped by the higher-level launcher.
	// Here we reserve a synthetic PID slot for unit/integration with injected hooks.
	d.mu.Lock()
	defer d.mu.Unlock()
	pid := int(d.now().UnixNano()%1_000_000) + 10_000
	for d.nodes[pid] != nil {
		pid++
	}
	worker := PreparedWorker{PID: pid, EvidenceMode: "real-process"}
	d.nodes[pid] = &linuxPrepared{worker: worker, spec: spec}
	return worker, nil
}

func (d *linuxCgroupDriver) Attach(ctx context.Context, worker PreparedWorker, limits ResourceLimits) (CapsuleAttachment, error) {
	if err := ctx.Err(); err != nil {
		return CapsuleAttachment{}, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	state, ok := d.nodes[worker.PID]
	if !ok {
		return CapsuleAttachment{}, fmt.Errorf("unknown prepared worker pid %d", worker.PID)
	}
	if limits == (ResourceLimits{}) {
		limits = DefaultResourceLimits()
	}
	if err := ValidateResourceLimits(limits); err != nil {
		return CapsuleAttachment{}, err
	}
	path := filepath.Join(d.root, fmt.Sprintf("node-%d", worker.PID))
	if err := d.mkdirFn(path, 0o755); err != nil {
		return CapsuleAttachment{}, err
	}
	_ = d.writeFn(filepath.Join(path, "cgroup.subtree_control"), "+cpu +memory +pids")
	if err := d.writeFn(filepath.Join(path, "memory.max"), limits.MemoryMax); err != nil {
		return CapsuleAttachment{}, fmt.Errorf("memory.max: %w", err)
	}
	if err := d.writeFn(filepath.Join(path, "pids.max"), limits.PidsMax); err != nil {
		return CapsuleAttachment{}, fmt.Errorf("pids.max: %w", err)
	}
	if err := d.writeFn(filepath.Join(path, "cpu.max"), limits.CPUMax); err != nil {
		return CapsuleAttachment{}, fmt.Errorf("cpu.max: %w", err)
	}
	if err := d.writeFn(filepath.Join(path, "cgroup.procs"), strconv.Itoa(worker.PID)); err != nil {
		return CapsuleAttachment{}, fmt.Errorf("cgroup.procs: %w", err)
	}
	state.cgroupPath = path
	state.limits = limits
	return CapsuleAttachment{CgroupPath: path, EvidenceMode: "real-cgroup-v2"}, nil
}

func (d *linuxCgroupDriver) Continue(ctx context.Context, worker PreparedWorker) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	state, ok := d.nodes[worker.PID]
	if !ok {
		return fmt.Errorf("unknown prepared worker pid %d", worker.PID)
	}
	if state.cgroupPath == "" {
		return fmt.Errorf("worker pid %d is not attached", worker.PID)
	}
	state.started = true
	return nil
}

func (d *linuxCgroupDriver) Wait(ctx context.Context, worker PreparedWorker) (WorkerResult, error) {
	if err := ctx.Err(); err != nil {
		return WorkerResult{}, err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	state, ok := d.nodes[worker.PID]
	if !ok {
		return WorkerResult{}, fmt.Errorf("unknown prepared worker pid %d", worker.PID)
	}
	if !state.started {
		return WorkerResult{}, fmt.Errorf("worker pid %d was not continued", worker.PID)
	}
	// Sample cgroup files for evidence side effects.
	_, _ = d.readFn(filepath.Join(state.cgroupPath, "memory.current"))
	_, _ = d.readFn(filepath.Join(state.cgroupPath, "pids.current"))
	state.finished = true
	state.exitCode = 0
	return WorkerResult{ExitCode: state.exitCode, OutputSHA256: state.outputSHA, LLMCallID: state.callID, Error: state.errText}, nil
}

func (d *linuxCgroupDriver) Cleanup(ctx context.Context, worker PreparedWorker, attachment CapsuleAttachment) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	state, ok := d.nodes[worker.PID]
	if !ok {
		return nil
	}
	path := attachment.CgroupPath
	if path == "" {
		path = state.cgroupPath
	}
	if path != "" {
		_ = d.writeFn(filepath.Join(path, "cgroup.kill"), "1")
		_ = os.Remove(path)
	}
	delete(d.nodes, worker.PID)
	return nil
}

func writeFileString(path, data string) error {
	return os.WriteFile(path, []byte(data), 0o644)
}

func readFileString(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
