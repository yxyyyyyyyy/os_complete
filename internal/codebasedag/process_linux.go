//go:build linux

package codebasedag

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
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
	// startFn optional hook for tests; nil means real exec.Command.
	startFn func(spec WorkerSpec) (*exec.Cmd, error)
	nodes   map[int]*linuxPrepared
}

type linuxPrepared struct {
	worker     PreparedWorker
	spec       WorkerSpec
	cmd        *exec.Cmd
	cgroupPath string
	limits     ResourceLimits
	started    bool
	finished   bool
	exitCode   int
	outputSHA  string
	callID     string
	errText    string
	metrics    map[string]string
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
	parent := filepath.Dir(root)
	if parent != "" && parent != "." && parent != "/" {
		_ = os.MkdirAll(parent, 0o755)
		_ = os.WriteFile(filepath.Join(parent, "cgroup.subtree_control"), []byte("+cpu +memory +pids\n"), 0o644)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return unsupportedProcessRuntime{reason: fmt.Sprintf("cgroup root unavailable: %v", err)}
	}
	_ = os.WriteFile(filepath.Join(root, "cgroup.subtree_control"), []byte("+cpu +memory +pids\n"), 0o644)
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
	var cmd *exec.Cmd
	var err error
	if d.startFn != nil {
		cmd, err = d.startFn(spec)
	} else {
		cmd, err = startStoppedCommand(spec)
	}
	if err != nil {
		return PreparedWorker{}, err
	}
	if cmd.Process == nil || cmd.Process.Pid <= 0 {
		return PreparedWorker{}, fmt.Errorf("worker started without kernel pid")
	}
	pid := cmd.Process.Pid
	if err := d.killFn(pid, syscall.SIGSTOP); err != nil {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		return PreparedWorker{}, fmt.Errorf("SIGSTOP worker: %w", err)
	}
	worker := PreparedWorker{PID: pid, EvidenceMode: "real-process"}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.nodes[pid] != nil {
		_ = d.killFn(pid, syscall.SIGKILL)
		_, _ = cmd.Process.Wait()
		return PreparedWorker{}, fmt.Errorf("duplicate prepared pid %d", pid)
	}
	d.nodes[pid] = &linuxPrepared{worker: worker, spec: spec, cmd: cmd, metrics: map[string]string{}}
	return worker, nil
}

func startStoppedCommand(spec WorkerSpec) (*exec.Cmd, error) {
	cmd := exec.Command(spec.Command, spec.Args...)
	cmd.Dir = spec.Dir
	if len(spec.Env) > 0 {
		cmd.Env = append(os.Environ(), spec.Env...)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
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
	// Controllers must be enabled on the parent before children expose memory.max/cpu.max/pids.max.
	if err := d.writeFn(filepath.Join(d.root, "cgroup.subtree_control"), "+cpu +memory +pids"); err != nil {
		state.worker.EvidenceMode = "degraded"
		return CapsuleAttachment{EvidenceMode: "degraded"}, fmt.Errorf("cgroup.subtree_control: %w", err)
	}
	path := filepath.Join(d.root, fmt.Sprintf("node-%d", worker.PID))
	if err := d.mkdirFn(path, 0o755); err != nil {
		state.worker.EvidenceMode = "degraded"
		return CapsuleAttachment{EvidenceMode: "degraded"}, fmt.Errorf("cgroup mkdir: %w", err)
	}
	if err := d.writeFn(filepath.Join(path, "memory.max"), limits.MemoryMax); err != nil {
		state.worker.EvidenceMode = "degraded"
		return CapsuleAttachment{CgroupPath: path, EvidenceMode: "degraded"}, fmt.Errorf("memory.max: %w", err)
	}
	if limits.MemoryHigh != "" {
		if err := d.writeFn(filepath.Join(path, "memory.high"), limits.MemoryHigh); err != nil {
			state.worker.EvidenceMode = "degraded"
			return CapsuleAttachment{CgroupPath: path, EvidenceMode: "degraded"}, fmt.Errorf("memory.high: %w", err)
		}
	}
	if err := d.writeFn(filepath.Join(path, "pids.max"), limits.PidsMax); err != nil {
		state.worker.EvidenceMode = "degraded"
		return CapsuleAttachment{CgroupPath: path, EvidenceMode: "degraded"}, fmt.Errorf("pids.max: %w", err)
	}
	if err := d.writeFn(filepath.Join(path, "cpu.max"), limits.CPUMax); err != nil {
		state.worker.EvidenceMode = "degraded"
		return CapsuleAttachment{CgroupPath: path, EvidenceMode: "degraded"}, fmt.Errorf("cpu.max: %w", err)
	}
	if err := d.writeFn(filepath.Join(path, "cgroup.procs"), strconv.Itoa(worker.PID)); err != nil {
		state.worker.EvidenceMode = "degraded"
		return CapsuleAttachment{CgroupPath: path, EvidenceMode: "degraded"}, fmt.Errorf("cgroup.procs: %w", err)
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
	state, ok := d.nodes[worker.PID]
	d.mu.Unlock()
	if !ok {
		return fmt.Errorf("unknown prepared worker pid %d", worker.PID)
	}
	if state.cgroupPath == "" {
		return fmt.Errorf("worker pid %d is not attached", worker.PID)
	}
	if err := d.killFn(worker.PID, syscall.SIGCONT); err != nil {
		return fmt.Errorf("SIGCONT worker: %w", err)
	}
	d.mu.Lock()
	state.started = true
	d.mu.Unlock()
	return nil
}

func (d *linuxCgroupDriver) Wait(ctx context.Context, worker PreparedWorker) (WorkerResult, error) {
	d.mu.Lock()
	state, ok := d.nodes[worker.PID]
	d.mu.Unlock()
	if !ok {
		return WorkerResult{}, fmt.Errorf("unknown prepared worker pid %d", worker.PID)
	}
	if !state.started {
		return WorkerResult{}, fmt.Errorf("worker pid %d was not continued", worker.PID)
	}
	if state.cmd == nil || state.cmd.Process == nil {
		return WorkerResult{}, fmt.Errorf("worker pid %d missing process handle", worker.PID)
	}

	done := make(chan error, 1)
	go func() { done <- state.cmd.Wait() }()
	var waitErr error
	select {
	case <-ctx.Done():
		_ = d.killFn(worker.PID, syscall.SIGKILL)
		if state.cmd.Process != nil {
			_ = syscall.Kill(-worker.PID, syscall.SIGKILL)
		}
		waitErr = <-done
		if waitErr == nil {
			waitErr = ctx.Err()
		}
	case waitErr = <-done:
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	state.finished = true
	state.exitCode = 0
	if state.cmd.ProcessState != nil {
		state.exitCode = state.cmd.ProcessState.ExitCode()
	} else if waitErr != nil {
		state.exitCode = -1
		state.errText = waitErr.Error()
	}
	if state.cgroupPath != "" {
		state.metrics["memory.current"] = d.readMetric(state.cgroupPath, "memory.current")
		state.metrics["memory.peak"] = d.readMetric(state.cgroupPath, "memory.peak")
		state.metrics["memory.events"] = d.readMetric(state.cgroupPath, "memory.events")
		state.metrics["pids.current"] = d.readMetric(state.cgroupPath, "pids.current")
		state.metrics["cpu.stat"] = d.readMetric(state.cgroupPath, "cpu.stat")
		sum := sha256.Sum256([]byte(fmt.Sprintf("%v", state.metrics)))
		state.outputSHA = hex.EncodeToString(sum[:])
	}
	metrics := map[string]string{}
	for k, v := range state.metrics {
		metrics[k] = v
	}
	return WorkerResult{
		ExitCode:     state.exitCode,
		OutputSHA256: state.outputSHA,
		LLMCallID:    state.callID,
		Error:        state.errText,
		Metrics:      metrics,
	}, nil
}

func (d *linuxCgroupDriver) readMetric(dir, name string) string {
	v, err := d.readFn(filepath.Join(dir, name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(v)
}

func (d *linuxCgroupDriver) Cleanup(ctx context.Context, worker PreparedWorker, attachment CapsuleAttachment) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	state, ok := d.nodes[worker.PID]
	if !ok {
		return nil
	}
	var errs []error
	if state.cmd != nil && state.cmd.Process != nil && !state.finished {
		if err := d.killFn(worker.PID, syscall.SIGKILL); err != nil {
			errs = append(errs, err)
		}
		_ = syscall.Kill(-worker.PID, syscall.SIGKILL)
		_, _ = state.cmd.Process.Wait()
	}
	path := attachment.CgroupPath
	if path == "" {
		path = state.cgroupPath
	}
	if path != "" {
		if err := d.writeFn(filepath.Join(path, "cgroup.kill"), "1"); err != nil {
			errs = append(errs, err)
		}
		if err := os.RemoveAll(path); err != nil && !os.IsNotExist(err) {
			errs = append(errs, err)
		}
	}
	delete(d.nodes, worker.PID)
	return errors.Join(errs...)
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
