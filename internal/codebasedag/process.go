package codebasedag

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrProcessUnsupported = errors.New("real codebase DAG process runtime unsupported")

type WorkerSpec struct {
	Command    string
	Args       []string
	Env        []string
	Dir        string
	ExtraFiles []string
}

type ResourceLimits struct {
	MemoryMax string `json:"memory_max"`
	PidsMax   string `json:"pids_max"`
	CPUMax    string `json:"cpu_max"`
}

func DefaultResourceLimits() ResourceLimits {
	return ResourceLimits{MemoryMax: "67108864", PidsMax: "8", CPUMax: "100000 100000"}
}

type ProcessConfig struct {
	RunID  string
	NodeID string
	Worker WorkerSpec
	Limits ResourceLimits
}

type PreparedWorker struct {
	PID          int
	EvidenceMode string
}

type CapsuleAttachment struct {
	CgroupPath   string
	EvidenceMode string
}

type WorkerResult struct {
	ExitCode     int
	OutputSHA256 string
	LLMCallID    string
	Error        string
}

type ProcessResult struct {
	PID                 int    `json:"pid"`
	CgroupPath          string `json:"cgroup_path"`
	ProcessEvidenceMode string `json:"process_evidence_mode"`
	CgroupEvidenceMode  string `json:"cgroup_evidence_mode"`
	EvidenceMode        string `json:"evidence_mode"`
	ExitCode            int    `json:"exit_code"`
	OutputSHA256        string `json:"output_sha256"`
	LLMCallID           string `json:"llm_call_id"`
}

type ProcessRuntime interface {
	StartPrepared(context.Context, ProcessConfig) (ProcessResult, error)
}

type ProcessDriver interface {
	StartStopped(context.Context, WorkerSpec) (PreparedWorker, error)
	Attach(context.Context, PreparedWorker, ResourceLimits) (CapsuleAttachment, error)
	Continue(context.Context, PreparedWorker) error
	Wait(context.Context, PreparedWorker) (WorkerResult, error)
	Cleanup(context.Context, PreparedWorker, CapsuleAttachment) error
}

type InterfaceProcessRuntime struct {
	driver ProcessDriver
}

func NewInterfaceProcessRuntime(driver ProcessDriver) InterfaceProcessRuntime {
	return InterfaceProcessRuntime{driver: driver}
}

func (r InterfaceProcessRuntime) StartPrepared(ctx context.Context, cfg ProcessConfig) (ProcessResult, error) {
	if err := validateProcessConfig(cfg); err != nil {
		return ProcessResult{}, err
	}
	if r.driver == nil {
		return ProcessResult{}, fmt.Errorf("process driver is required")
	}
	worker, err := r.driver.StartStopped(ctx, cfg.Worker)
	if err != nil {
		return ProcessResult{}, err
	}
	var attachment CapsuleAttachment
	cleanup := true
	defer func() {
		if cleanup {
			_ = r.driver.Cleanup(ctx, worker, attachment)
		}
	}()
	attachment, err = r.driver.Attach(ctx, worker, cfg.Limits)
	if err != nil {
		return ProcessResult{}, err
	}
	if err := r.driver.Continue(ctx, worker); err != nil {
		return ProcessResult{}, err
	}
	result, err := r.driver.Wait(ctx, worker)
	if err != nil {
		return ProcessResult{}, err
	}
	if err := r.driver.Cleanup(ctx, worker, attachment); err != nil {
		return ProcessResult{}, err
	}
	cleanup = false
	if result.ExitCode != 0 {
		return ProcessResult{}, fmt.Errorf("worker exited %d: %s", result.ExitCode, result.Error)
	}
	return ProcessResult{
		PID:                 worker.PID,
		CgroupPath:          attachment.CgroupPath,
		ProcessEvidenceMode: worker.EvidenceMode,
		CgroupEvidenceMode:  attachment.EvidenceMode,
		EvidenceMode:        worker.EvidenceMode,
		ExitCode:            result.ExitCode,
		OutputSHA256:        result.OutputSHA256,
		LLMCallID:           result.LLMCallID,
	}, nil
}

func validateProcessConfig(cfg ProcessConfig) error {
	if cfg.RunID == "" {
		return fmt.Errorf("run ID is required")
	}
	if cfg.NodeID == "" {
		return fmt.Errorf("node ID is required")
	}
	if cfg.Worker.Command == "" {
		return fmt.Errorf("worker command is required")
	}
	return ValidateResourceLimits(cfg.Limits)
}

func ValidateResourceLimits(limits ResourceLimits) error {
	if limits == (ResourceLimits{}) {
		limits = DefaultResourceLimits()
	}
	if err := validatePositiveControl("memory.max", limits.MemoryMax); err != nil {
		return err
	}
	if err := validatePositiveControl("pids.max", limits.PidsMax); err != nil {
		return err
	}
	parts := strings.Fields(limits.CPUMax)
	if len(parts) != 2 {
		return fmt.Errorf("cpu.max must contain quota and period")
	}
	for _, part := range parts {
		value, err := strconv.ParseInt(part, 10, 64)
		if err != nil || value <= 0 {
			return fmt.Errorf("cpu.max values must be positive integers")
		}
	}
	return nil
}

func validatePositiveControl(name, value string) error {
	if value == "" {
		return fmt.Errorf("%s is required", name)
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil || parsed <= 0 {
		return fmt.Errorf("%s must be a positive integer", name)
	}
	return nil
}

type unsupportedProcessRuntime struct {
	reason string
}

func (r unsupportedProcessRuntime) StartPrepared(context.Context, ProcessConfig) (ProcessResult, error) {
	if r.reason == "" {
		r.reason = "real worker/cgroup lifecycle requires linux runtime wiring"
	}
	return ProcessResult{}, fmt.Errorf("%w: %s", ErrProcessUnsupported, r.reason)
}
