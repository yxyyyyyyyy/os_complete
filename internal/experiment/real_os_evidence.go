package experiment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"aort-r/internal/avp"
	"aort-r/internal/capsule"
	"aort-r/internal/evidence"
	"aort-r/internal/resource"
	"aort-r/internal/scheduler"
	syscallgw "aort-r/internal/syscall"
	"aort-r/internal/trace"
	"aort-r/internal/workspace"
)

const (
	defaultRealCgroupRoot = "/sys/fs/cgroup/aort.slice"
	defaultMemoryHogBytes = 64 * 1024 * 1024
	defaultPidsHogCount   = 8
	defaultCPUHogDuration = 2 * time.Second
)

type RealCgroupSmokeConfig struct {
	CgroupRoot string
	OutDir     string
	MemoryMax  string
	PidsMax    string
	CPUMax     string
	Timeout    time.Duration
}

type RealCgroupSmokeResult struct {
	Experiment            string `json:"experiment"`
	EvidenceMode          string `json:"evidence_mode"`
	CgroupPath            string `json:"cgroup_path,omitempty"`
	WorkerPID             int    `json:"worker_pid,omitempty"`
	WorkerPIDAttached     bool   `json:"worker_pid_attached"`
	MemoryMaxWritten      bool   `json:"memory_max_written"`
	PidsMaxWritten        bool   `json:"pids_max_written"`
	MemoryCurrentReadable bool   `json:"memory_current_readable"`
	PidsCurrentReadable   bool   `json:"pids_current_readable"`
	CPUStatReadable       bool   `json:"cpu_stat_readable"`
	FreezeSuccess         bool   `json:"freeze_success"`
	UnfreezeSuccess       bool   `json:"unfreeze_success"`
	CgroupKillSuccess     bool   `json:"cgroup_kill_success"`
	DestroySuccess        bool   `json:"destroy_success"`
	FailureReason         string `json:"failure_reason,omitempty"`
}

type RealPressureSmokeConfig struct {
	Runs                int
	OutDir              string
	CgroupRoot          string
	RequireReal         bool
	MemoryHogBytes      int64
	PidsHogCount        int
	CPUHogDuration      time.Duration
	HogCommand          string
	HogCommandExtraArgs []string
}

type RealPressureSmokeResult struct {
	Experiment                            string `json:"experiment"`
	EvidenceMode                          string `json:"evidence_mode"`
	Runs                                  int    `json:"runs"`
	MemoryHogDetected                     bool   `json:"memory_hog_detected"`
	PidsHogDetected                       bool   `json:"pids_hog_detected"`
	CPUPressureDetected                   bool   `json:"cpu_pressure_detected"`
	ResourceSamplerMode                   string `json:"resource_sampler_mode"`
	ResourceAwareAvoidedHighPressureAgent bool   `json:"resource_aware_avoided_high_pressure_agent"`
	SelectedHighPressureAgentCount        int    `json:"selected_high_pressure_agent_count"`
	CascadeFailure                        bool   `json:"cascade_failure"`
	CleanupSuccess                        bool   `json:"cleanup_success"`
	MemoryHogLimitBytes                   int64  `json:"memory_hog_limit_bytes"`
	PidsHogLimit                          int    `json:"pids_hog_limit"`
	CPUHogTimeoutMS                       int64  `json:"cpu_hog_timeout_ms"`
	FailureReason                         string `json:"failure_reason,omitempty"`
}

type ToolWorkspaceEvidence struct {
	ToolExecUsesWorkspace bool   `json:"tool_exec_uses_workspace"`
	WorkspaceMode         string `json:"workspace_mode"`
	CWDIsMerged           bool   `json:"cwd_is_merged"`
	CommitOnSuccess       bool   `json:"commit_on_success"`
	RollbackOnFailure     bool   `json:"rollback_on_failure"`
	RepoDirUntouched      bool   `json:"repo_dir_untouched"`
	PathEscapeBlocked     bool   `json:"path_escape_blocked"`
	SymlinkEscapeBlocked  bool   `json:"symlink_escape_blocked"`
	EvidenceMode          string `json:"evidence_mode"`
	MergedIsMountpoint    bool   `json:"merged_is_mountpoint"`
	FallbackReason        string `json:"fallback_reason,omitempty"`
	Error                 string `json:"error,omitempty"`
}

type RealAllInputs struct {
	EnvOpenEuler                    bool
	EnvCgroup2FS                    bool
	EnvOverlayFSMount               bool
	CgroupSmoke                     RealCgroupSmokeResult
	PressureSmoke                   RealPressureSmokeResult
	WorkspaceProbeRealOverlay       bool
	WorkspaceRMFault                workspace.RMFaultEvidence
	ToolWorkspace                   ToolWorkspaceEvidence
	E2PressureFaultCascadeFailure   bool
	SoftwareRealRuntimeSuccess      bool
	SoftwareRealRuntimeEvidenceMode string
}

type RealAllIndex struct {
	Experiment                         string   `json:"experiment"`
	EvidenceMode                       string   `json:"evidence_mode"`
	RealCgroupV2                       bool     `json:"real_cgroup_v2"`
	RealResourceSampler                bool     `json:"real_resource_sampler"`
	RealOverlayFS                      bool     `json:"real_overlayfs"`
	RealWorkspaceToolExec              bool     `json:"real_workspace_tool_exec"`
	CgroupKillSuccess                  bool     `json:"cgroup_kill_success"`
	ResourceAwareSchedulerRealPressure bool     `json:"resource_aware_scheduler_real_pressure"`
	CascadeFailure                     bool     `json:"cascade_failure"`
	AllPassed                          bool     `json:"all_passed"`
	FailureReasons                     []string `json:"failure_reasons,omitempty"`
}

func RunRealCgroupSmoke(cfg RealCgroupSmokeConfig) (RealCgroupSmokeResult, error) {
	if cfg.CgroupRoot == "" {
		cfg.CgroupRoot = defaultRealCgroupRoot
	}
	if cfg.OutDir == "" {
		cfg.OutDir = filepath.Join("experiments", "results", "real_cgroup_smoke")
	}
	if cfg.MemoryMax == "" {
		cfg.MemoryMax = strconv.FormatInt(defaultMemoryHogBytes, 10)
	}
	if cfg.PidsMax == "" {
		cfg.PidsMax = "8"
	}
	if cfg.CPUMax == "" {
		cfg.CPUMax = "100000 100000"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	result := RealCgroupSmokeResult{
		Experiment:   "real_cgroup_smoke",
		EvidenceMode: string(evidence.ModeMissing),
	}
	outPath := filepath.Join(cfg.OutDir, "real_cgroup_smoke.json")
	fail := func(reason string) (RealCgroupSmokeResult, error) {
		result.FailureReason = reason
		_ = WriteJSON(outPath, result)
		return result, errors.New(reason)
	}
	if err := ensureRealCgroupRoot(cfg.CgroupRoot); err != nil {
		return fail(err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "sleep", "30")
	if err := cmd.Start(); err != nil {
		return fail("worker process start failed: " + err.Error())
	}
	waitCh := make(chan error, 1)
	go func() { waitCh <- cmd.Wait() }()
	workerDone := false
	defer func() {
		if !workerDone && cmd.Process != nil {
			_ = cmd.Process.Kill()
			<-waitCh
		}
	}()

	manager := capsule.NewManager(capsule.Config{
		Root:          cfg.CgroupRoot,
		AllowDegraded: false,
		MemoryMax:     cfg.MemoryMax,
		PidsMax:       cfg.PidsMax,
		CPUMax:        cfg.CPUMax,
	})
	rt, err := manager.Prepare("real-cgroup-smoke-worker", cmd.Process.Pid)
	if err != nil {
		return fail("real cgroup prepare failed: " + err.Error())
	}
	result.CgroupPath = rt.CgroupPath
	result.WorkerPID = cmd.Process.Pid
	result.WorkerPIDAttached = fileContains(filepath.Join(rt.CgroupPath, "cgroup.procs"), strconv.Itoa(cmd.Process.Pid))
	result.MemoryMaxWritten = fileContains(filepath.Join(rt.CgroupPath, "memory.max"), strings.TrimSuffix(cfg.MemoryMax, "\n"))
	result.PidsMaxWritten = fileContains(filepath.Join(rt.CgroupPath, "pids.max"), strings.TrimSuffix(cfg.PidsMax, "\n"))
	result.MemoryCurrentReadable = readableFile(filepath.Join(rt.CgroupPath, "memory.current"))
	result.PidsCurrentReadable = readableFile(filepath.Join(rt.CgroupPath, "pids.current"))
	result.CPUStatReadable = readableFile(filepath.Join(rt.CgroupPath, "cpu.stat"))

	if !result.WorkerPIDAttached {
		return fail("worker PID was not attached to cgroup")
	}
	if !result.MemoryMaxWritten || !result.PidsMaxWritten || !result.MemoryCurrentReadable || !result.PidsCurrentReadable || !result.CPUStatReadable {
		return fail("required cgroup v2 files were not writable/readable")
	}
	if err := manager.Freeze("real-cgroup-smoke-worker"); err != nil {
		return fail("cgroup freeze failed: " + err.Error())
	}
	result.FreezeSuccess = true
	if err := manager.Unfreeze("real-cgroup-smoke-worker"); err != nil {
		return fail("cgroup unfreeze failed: " + err.Error())
	}
	result.UnfreezeSuccess = true
	killResult, err := manager.Kill("real-cgroup-smoke-worker")
	if err != nil {
		return fail("cgroup.kill failed: " + err.Error())
	}
	result.CgroupKillSuccess = killResult.KillMethod == capsule.KillMethodCgroupKill && killResult.EvidenceMode == evidence.ModeRealCgroupV2
	if !result.CgroupKillSuccess {
		reason := killResult.FallbackReason
		if reason == "" {
			reason = "cgroup.kill unsupported"
		}
		return fail(reason)
	}
	select {
	case <-waitCh:
		workerDone = true
	case <-time.After(2 * time.Second):
		return fail("worker did not exit after cgroup.kill")
	}
	if err := manager.Destroy("real-cgroup-smoke-worker"); err != nil {
		return fail("destroy cgroup failed: " + err.Error())
	}
	result.DestroySuccess = true
	result.EvidenceMode = string(evidence.ModeRealCgroupV2)
	if err := WriteJSON(outPath, result); err != nil {
		return result, err
	}
	return result, nil
}

func RunRealPressureSmoke(cfg RealPressureSmokeConfig) (RealPressureSmokeResult, error) {
	if cfg.Runs <= 0 {
		cfg.Runs = 3
	}
	if cfg.OutDir == "" {
		cfg.OutDir = filepath.Join("experiments", "results", "real_pressure_smoke")
	}
	if cfg.CgroupRoot == "" {
		cfg.CgroupRoot = defaultRealCgroupRoot
	}
	if cfg.MemoryHogBytes <= 0 {
		cfg.MemoryHogBytes = defaultMemoryHogBytes
	}
	if cfg.PidsHogCount <= 0 {
		cfg.PidsHogCount = defaultPidsHogCount
	}
	if cfg.CPUHogDuration <= 0 {
		cfg.CPUHogDuration = defaultCPUHogDuration
	}
	result := RealPressureSmokeResult{
		Experiment:          "real_pressure_smoke",
		Runs:                cfg.Runs,
		EvidenceMode:        string(evidence.ModeMissing),
		ResourceSamplerMode: string(evidence.ModeMissing),
		CleanupSuccess:      true,
		MemoryHogLimitBytes: cfg.MemoryHogBytes,
		PidsHogLimit:        cfg.PidsHogCount,
		CPUHogTimeoutMS:     cfg.CPUHogDuration.Milliseconds(),
	}
	outPath := filepath.Join(cfg.OutDir, "real_pressure_smoke.json")
	fail := func(reason string) (RealPressureSmokeResult, error) {
		result.FailureReason = reason
		_ = WriteJSON(outPath, result)
		return result, errors.New(reason)
	}
	if err := ensureRealCgroupRoot(cfg.CgroupRoot); err != nil {
		if cfg.RequireReal {
			return fail(err.Error())
		}
		result.FailureReason = err.Error()
		_ = WriteJSON(outPath, result)
		return result, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.CPUHogDuration+5*time.Second)
	defer cancel()
	manager := capsule.NewManager(capsule.Config{
		Root:          cfg.CgroupRoot,
		AllowDegraded: false,
		MemoryMax:     strconv.FormatInt(defaultMemoryHogBytes*2, 10),
		PidsMax:       "16",
		CPUMax:        "50000 100000",
	})
	high, err := startPressureHog(ctx, cfg, manager, "high-pressure-agent", true)
	if err != nil {
		return fail("start high pressure hog failed: " + err.Error())
	}
	low, err := startPressureHog(ctx, cfg, manager, "low-pressure-agent", false)
	if err != nil {
		high.cleanup()
		return fail("start low pressure hog failed: " + err.Error())
	}
	cleanupDone := false
	cleanup := func() bool {
		if cleanupDone {
			return result.CleanupSuccess
		}
		cleanupDone = true
		ok := true
		if err := high.cleanup(); err != nil {
			ok = false
		}
		if err := low.cleanup(); err != nil {
			ok = false
		}
		return ok
	}
	defer func() {
		if !cleanupDone {
			result.CleanupSuccess = cleanup()
		}
	}()
	failRunning := func(reason string) (RealPressureSmokeResult, error) {
		result.CleanupSuccess = cleanup()
		return fail(reason)
	}

	time.Sleep(minDuration(900*time.Millisecond, cfg.CPUHogDuration))
	sampler := resource.NewCgroupSampler("")
	highAgent, highPressure, err := sampler.Enrich(avp.AVP{
		AgentID:    "high-pressure-agent",
		TaskID:     "real-pressure-smoke",
		State:      avp.StateReady,
		VRuntime:   5,
		CgroupPath: high.runtime.CgroupPath,
		CreatedAt:  1,
	})
	if err != nil {
		return failRunning("real resource sampler failed for high pressure agent: " + err.Error())
	}
	lowAgent, _, err := sampler.Enrich(avp.AVP{
		AgentID:    "low-pressure-agent",
		TaskID:     "real-pressure-smoke",
		State:      avp.StateReady,
		VRuntime:   6,
		CgroupPath: low.runtime.CgroupPath,
		CreatedAt:  2,
	})
	if err != nil {
		return failRunning("real resource sampler failed for low pressure agent: " + err.Error())
	}
	result.MemoryHogDetected = highAgent.MemoryCurrent >= cfg.MemoryHogBytes/2
	result.PidsHogDetected = int(highAgent.PidsCurrent) >= max(2, cfg.PidsHogCount/2)
	result.CPUPressureDetected = highAgent.CPUStat["usage_usec"] > 0 || highAgent.CPUStat["nr_throttled"] > 0 || highAgent.CPUStat["throttled_usec"] > 0
	result.ResourceSamplerMode = string(highPressure.EvidenceMode)
	result.EvidenceMode = string(highPressure.EvidenceMode)

	selectedHigh := 0
	for run := 0; run < cfg.Runs; run++ {
		s := scheduler.New(scheduler.PolicyTokenCFSPrefixAffinityResourceAware)
		s.SetResourcePressure(scheduler.ResourcePressure{
			PSIPressure:  highPressure.PSIPressure,
			EvidenceMode: evidence.ModeRealCgroupV2,
		})
		selected, _, ok := s.Select(fmt.Sprintf("real-pressure-smoke-%d", run), []avp.AVP{highAgent, lowAgent})
		if !ok {
			return failRunning("resource-aware scheduler did not select an agent")
		}
		if selected.AgentID == highAgent.AgentID {
			selectedHigh++
		}
	}
	result.SelectedHighPressureAgentCount = selectedHigh
	result.ResourceAwareAvoidedHighPressureAgent = selectedHigh == 0
	result.CascadeFailure = selectedHigh > 0
	if cfg.RequireReal {
		switch {
		case result.EvidenceMode != string(evidence.ModeRealCgroupV2):
			return failRunning("resource sampler mode is not real-cgroup-v2")
		case !result.MemoryHogDetected:
			return failRunning("memory hog was not detected from memory.current")
		case !result.PidsHogDetected:
			return failRunning("pids hog was not detected from pids.current")
		case !result.CPUPressureDetected:
			return failRunning("cpu hog was not detected from cpu.stat")
		case !result.ResourceAwareAvoidedHighPressureAgent:
			return failRunning("resource-aware scheduler selected high-pressure agent")
		}
	}
	result.CleanupSuccess = cleanup()
	if cfg.RequireReal && !result.CleanupSuccess {
		return fail("cleanup failed")
	}
	if err := WriteJSON(outPath, result); err != nil {
		return result, err
	}
	return result, nil
}

func RunToolWorkspaceDemo(cfg workspace.Config, outDir string, requireRealOverlay bool) (ToolWorkspaceEvidence, error) {
	if outDir == "" {
		outDir = filepath.Join("experiments", "results")
	}
	manager := workspace.NewManager(cfg)
	ws, err := manager.Create("tool-agent")
	result := ToolWorkspaceEvidence{
		RepoDirUntouched: true,
		EvidenceMode:     string(evidence.ModeMissing),
	}
	outPath := filepath.Join(outDir, "tool_workspace_evidence.json")
	fail := func(reason string) (ToolWorkspaceEvidence, error) {
		result.Error = reason
		if result.FallbackReason == "" {
			result.FallbackReason = reason
		}
		_ = WriteJSON(outPath, result)
		return result, errors.New(reason)
	}
	if err != nil {
		return fail("workspace create failed: " + err.Error())
	}
	defer manager.Destroy("tool-agent")
	result.WorkspaceMode = ws.Mode
	result.EvidenceMode = string(ws.EvidenceMode)
	result.FallbackReason = ws.FallbackReason
	result.MergedIsMountpoint = ws.Mounted
	if requireRealOverlay && (ws.Mode != workspace.ModeOverlayFS || ws.EvidenceMode != evidence.ModeRealOverlayFS || !ws.Mounted) {
		return fail("real-overlayfs required: " + fallbackOr(ws.FallbackReason, "workspace is not mounted overlayfs"))
	}

	beforeRepo := repoFingerprint()
	gateway := syscallgw.NewGateway(syscallgw.Config{WorkspaceRuntime: experimentWorkspaceRuntime{manager: manager}})
	success := gateway.Handle(context.Background(), syscallgw.Request{
		RequestID: "tool-workspace-success",
		TaskID:    "tool-workspace",
		AgentID:   "tool-agent",
		Name:      "tool.exec",
		Args: map[string]any{
			"command": "sh",
			"args":    []any{"-c", "printf 'changed by tool\\n' > src/tool.txt; pwd"},
		},
	})
	cwd, _ := success.Payload["cwd"].(string)
	result.ToolExecUsesWorkspace = success.Status == syscallgw.StatusOK && cwd == ws.MergedDir
	result.CWDIsMerged = cwd == ws.MergedDir
	result.CommitOnSuccess = success.Status == syscallgw.StatusOK && fileExists(filepath.Join(ws.OutputDir, "commit_manifest.json"))

	failure := gateway.Handle(context.Background(), syscallgw.Request{
		RequestID: "tool-workspace-failure",
		TaskID:    "tool-workspace",
		AgentID:   "tool-agent",
		Name:      "tool.exec",
		Args: map[string]any{
			"command": "sh",
			"args":    []any{"-c", "printf 'broken\\n' > src/main.txt; exit 7"},
		},
	})
	restored, _ := os.ReadFile(filepath.Join(ws.MergedDir, "src", "main.txt"))
	result.RollbackOnFailure = failure.Status == syscallgw.StatusError && strings.Contains(string(restored), "base source")

	escape := gateway.Handle(context.Background(), syscallgw.Request{
		RequestID: "tool-workspace-path-escape",
		TaskID:    "tool-workspace",
		AgentID:   "tool-agent",
		Name:      "tool.exec",
		Args: map[string]any{
			"command": "pwd",
			"cwd":     "../escape",
		},
	})
	result.PathEscapeBlocked = escape.Status == syscallgw.StatusDenied

	outside := filepath.Join(os.TempDir(), "aort-tool-workspace-outside")
	_ = os.MkdirAll(outside, 0o755)
	linkPath := filepath.Join(ws.MergedDir, "symlink-escape")
	if err := os.Symlink(outside, linkPath); err == nil {
		symlink := gateway.Handle(context.Background(), syscallgw.Request{
			RequestID: "tool-workspace-symlink-escape",
			TaskID:    "tool-workspace",
			AgentID:   "tool-agent",
			Name:      "tool.exec",
			Args: map[string]any{
				"command": "true",
			},
		})
		result.SymlinkEscapeBlocked = symlink.Status == syscallgw.StatusError && strings.Contains(symlink.Error, "symlink escape blocked")
		_ = os.Remove(linkPath)
		_, _ = manager.Rollback("tool-agent")
	}
	result.RepoDirUntouched = beforeRepo == repoFingerprint()
	if requireRealOverlay {
		switch {
		case !result.ToolExecUsesWorkspace:
			return fail("tool.exec did not use merged workspace cwd")
		case !result.CommitOnSuccess:
			return fail("successful tool did not commit workspace")
		case !result.RollbackOnFailure:
			return fail("failed tool did not rollback workspace")
		case !result.PathEscapeBlocked:
			return fail("path escape was not blocked")
		case !result.SymlinkEscapeBlocked:
			return fail("symlink escape was not blocked")
		}
	}
	if err := WriteJSON(outPath, result); err != nil {
		return result, err
	}
	return result, nil
}

func RunSoftwareRealDemo(runs int, outDir string) (E5EndToEndResult, error) {
	if runs <= 0 {
		runs = 5
	}
	if outDir == "" {
		outDir = filepath.Join("experiments", "results")
	}
	result, traceEvents := RunE5EndToEndBenchmarkWithTrace(runs)
	demoDir := filepath.Join(outDir, "software_real_demo")
	if err := WriteJSON(filepath.Join(demoDir, "result.json"), result); err != nil {
		return result, err
	}
	if err := trace.WriteTrace(filepath.Join(demoDir, "trace.json"), traceEvents); err != nil {
		return result, err
	}
	return result, nil
}

func BuildRealAllIndex(inputs RealAllInputs) RealAllIndex {
	reasons := []string{}
	realCgroup := inputs.EnvOpenEuler &&
		inputs.EnvCgroup2FS &&
		inputs.CgroupSmoke.EvidenceMode == string(evidence.ModeRealCgroupV2) &&
		inputs.CgroupSmoke.WorkerPIDAttached &&
		inputs.CgroupSmoke.CgroupKillSuccess &&
		inputs.CgroupSmoke.DestroySuccess
	if !realCgroup {
		reasons = append(reasons, "real cgroup v2 smoke did not pass")
	}
	realSampler := inputs.PressureSmoke.EvidenceMode == string(evidence.ModeRealCgroupV2) &&
		inputs.PressureSmoke.ResourceSamplerMode == string(evidence.ModeRealCgroupV2) &&
		inputs.PressureSmoke.MemoryHogDetected &&
		inputs.PressureSmoke.PidsHogDetected &&
		inputs.PressureSmoke.CPUPressureDetected &&
		inputs.PressureSmoke.ResourceAwareAvoidedHighPressureAgent &&
		inputs.PressureSmoke.SelectedHighPressureAgentCount == 0 &&
		inputs.PressureSmoke.CleanupSuccess
	if !realSampler {
		reasons = append(reasons, "real resource sampler pressure smoke did not pass")
	}
	realOverlay := inputs.EnvOverlayFSMount &&
		inputs.WorkspaceProbeRealOverlay &&
		inputs.WorkspaceRMFault.EvidenceMode == evidence.ModeRealOverlayFS &&
		inputs.WorkspaceRMFault.MergedIsMountpoint &&
		!inputs.WorkspaceRMFault.CascadeFailure
	if !realOverlay {
		reasons = append(reasons, "real overlayfs workspace evidence did not pass")
	}
	realTool := inputs.ToolWorkspace.EvidenceMode == string(evidence.ModeRealOverlayFS) &&
		inputs.ToolWorkspace.ToolExecUsesWorkspace &&
		inputs.ToolWorkspace.CWDIsMerged &&
		inputs.ToolWorkspace.CommitOnSuccess &&
		inputs.ToolWorkspace.RollbackOnFailure &&
		inputs.ToolWorkspace.RepoDirUntouched &&
		inputs.ToolWorkspace.PathEscapeBlocked &&
		inputs.ToolWorkspace.SymlinkEscapeBlocked
	if !realTool {
		reasons = append(reasons, "real overlayfs tool.exec evidence did not pass")
	}
	cascade := inputs.E2PressureFaultCascadeFailure || inputs.WorkspaceRMFault.CascadeFailure || inputs.PressureSmoke.CascadeFailure
	if cascade {
		reasons = append(reasons, "cascade failure detected")
	}
	softwareReal := inputs.SoftwareRealRuntimeSuccess && inputs.SoftwareRealRuntimeEvidenceMode == string(evidence.ModeRealRuntime)
	if !softwareReal {
		reasons = append(reasons, "software-real runtime evidence did not pass")
	}
	index := RealAllIndex{
		Experiment:                         "real_all",
		EvidenceMode:                       "real-openeuler",
		RealCgroupV2:                       realCgroup,
		RealResourceSampler:                realSampler,
		RealOverlayFS:                      realOverlay,
		RealWorkspaceToolExec:              realTool,
		CgroupKillSuccess:                  inputs.CgroupSmoke.CgroupKillSuccess,
		ResourceAwareSchedulerRealPressure: realSampler,
		CascadeFailure:                     cascade,
	}
	index.AllPassed = realCgroup && realSampler && realOverlay && realTool && !cascade && softwareReal
	if !index.AllPassed {
		index.FailureReasons = reasons
	}
	return index
}

func RunRealAll(runs int, outDir string) (RealAllIndex, error) {
	if runs <= 0 {
		runs = 3
	}
	if outDir == "" {
		outDir = filepath.Join("experiments", "results", "real_all")
	}
	reasons := []string{}
	envPath := filepath.Join(outDir, "real_openeuler_env.json")
	envData := map[string]any{}
	envCmd := exec.Command("bash", filepath.Join(repoRoot(), "scripts", "verify_real_openeuler_env.sh"))
	envCmd.Env = append(os.Environ(), "AORT_REAL_ENV_OUT="+envPath)
	if output, err := envCmd.CombinedOutput(); err != nil {
		reasons = append(reasons, "real env check failed: "+strings.TrimSpace(string(output)))
	}
	if data, err := os.ReadFile(envPath); err == nil {
		_ = json.Unmarshal(data, &envData)
	}
	cgroupSmoke, cgroupErr := RunRealCgroupSmoke(RealCgroupSmokeConfig{OutDir: filepath.Join(outDir, "real_cgroup_smoke")})
	if cgroupErr != nil {
		reasons = append(reasons, cgroupErr.Error())
	}
	pressure, pressureErr := RunRealPressureSmoke(RealPressureSmokeConfig{Runs: runs, OutDir: filepath.Join(outDir, "real_pressure_smoke"), RequireReal: true})
	if pressureErr != nil {
		reasons = append(reasons, pressureErr.Error())
	}
	probeRoot, cleanupProbe, probeRootErr := realAllWorkspaceRoot("workspace-probe")
	if probeRootErr != nil {
		reasons = append(reasons, probeRootErr.Error())
	}
	defer cleanupProbe()
	probe := workspace.ProbeOverlay(probeRoot)
	if err := WriteJSON(filepath.Join(outDir, "workspace_probe.json"), probe); err != nil {
		reasons = append(reasons, err.Error())
	}
	rmRoot, cleanupRM, rmRootErr := realAllWorkspaceRoot("workspace-rmrf")
	if rmRootErr != nil {
		reasons = append(reasons, rmRootErr.Error())
	}
	defer cleanupRM()
	rmFault, rmErr := workspace.RunRMFaultDemo(workspace.Config{Root: rmRoot})
	if rmErr != nil {
		reasons = append(reasons, rmErr.Error())
	}
	if rmFault.EvidenceMode != evidence.ModeRealOverlayFS {
		rmFault.Success = false
		rmFault.Error = "real-overlayfs required: " + fallbackOr(rmFault.FallbackReason, "workspace-rmrf did not use overlayfs")
	}
	if err := WriteJSON(filepath.Join(outDir, "workspace_isolation_evidence.json"), rmFault); err != nil {
		reasons = append(reasons, err.Error())
	}
	toolRoot, cleanupTool, toolRootErr := realAllWorkspaceRoot("tool-workspace")
	if toolRootErr != nil {
		reasons = append(reasons, toolRootErr.Error())
	}
	defer cleanupTool()
	toolWorkspace, toolErr := RunToolWorkspaceDemo(workspace.Config{Root: toolRoot}, outDir, true)
	if toolErr != nil {
		reasons = append(reasons, toolErr.Error())
	}
	e2Pressure, e2Err := RunE2PressureFault(runs, filepath.Join(outDir, "e2_pressure_fault"))
	if e2Err != nil {
		reasons = append(reasons, e2Err.Error())
	}
	software, softwareErr := RunSoftwareRealDemo(runs, outDir)
	if softwareErr != nil {
		reasons = append(reasons, softwareErr.Error())
	}
	index := BuildRealAllIndex(RealAllInputs{
		EnvOpenEuler:                    boolFromMap(envData, "openEuler"),
		EnvCgroup2FS:                    boolFromMap(envData, "cgroup2fs"),
		EnvOverlayFSMount:               boolFromMap(envData, "overlayfs_mount_success"),
		CgroupSmoke:                     cgroupSmoke,
		PressureSmoke:                   pressure,
		WorkspaceProbeRealOverlay:       probe.EvidenceMode == evidence.ModeRealOverlayFS && probe.MountTestSuccess && probe.MergedIsMountpoint,
		WorkspaceRMFault:                rmFault,
		ToolWorkspace:                   toolWorkspace,
		E2PressureFaultCascadeFailure:   e2Pressure.CascadeFailure,
		SoftwareRealRuntimeSuccess:      software.FinalSuccess,
		SoftwareRealRuntimeEvidenceMode: software.EvidenceMode,
	})
	if len(reasons) > 0 {
		index.FailureReasons = append(index.FailureReasons, reasons...)
		index.AllPassed = false
	}
	if err := WriteJSON(filepath.Join(outDir, "REAL_EVIDENCE_INDEX.json"), index); err != nil {
		return index, err
	}
	if !index.AllPassed {
		return index, fmt.Errorf("real-all failed: %s", strings.Join(index.FailureReasons, "; "))
	}
	return index, nil
}

func realAllWorkspaceRoot(label string) (string, func(), error) {
	root, err := os.MkdirTemp("", "aort-real-all-"+label+"-")
	if err != nil {
		return "", func() {}, fmt.Errorf("create real-all workspace root: %w", err)
	}
	return root, func() { _ = os.RemoveAll(root) }, nil
}

type pressureHog struct {
	cmd     *exec.Cmd
	cancel  context.CancelFunc
	done    chan error
	manager *capsule.Manager
	agentID string
	runtime capsule.Runtime
}

func startPressureHog(parent context.Context, cfg RealPressureSmokeConfig, manager *capsule.Manager, agentID string, high bool) (pressureHog, error) {
	ctx, cancel := context.WithCancel(parent)
	args := append([]string(nil), cfg.HogCommandExtraArgs...)
	if high {
		args = append(args, "_hog", "pressure",
			"--memory-bytes", strconv.FormatInt(cfg.MemoryHogBytes, 10),
			"--pids", strconv.Itoa(cfg.PidsHogCount),
			"--duration-ms", strconv.FormatInt(cfg.CPUHogDuration.Milliseconds(), 10),
		)
	} else {
		args = append(args, "_hog", "sleep", "--duration-ms", strconv.FormatInt((cfg.CPUHogDuration+time.Second).Milliseconds(), 10))
	}
	command := cfg.HogCommand
	if command == "" {
		var err error
		command, err = os.Executable()
		if err != nil {
			cancel()
			return pressureHog{}, err
		}
	}
	cmd := exec.CommandContext(ctx, command, args...)
	if err := cmd.Start(); err != nil {
		cancel()
		return pressureHog{}, err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	rt, err := manager.Prepare(agentID, cmd.Process.Pid)
	if err != nil {
		cancel()
		<-done
		return pressureHog{}, err
	}
	return pressureHog{cmd: cmd, cancel: cancel, done: done, manager: manager, agentID: agentID, runtime: rt}, nil
}

func (h pressureHog) cleanup() error {
	h.cancel()
	if h.manager != nil {
		_, _ = h.manager.Kill(h.agentID)
	}
	select {
	case <-h.done:
	case <-time.After(2 * time.Second):
		if h.cmd != nil && h.cmd.Process != nil {
			_ = h.cmd.Process.Kill()
		}
		<-h.done
	}
	if h.manager != nil {
		return h.manager.Destroy(h.agentID)
	}
	return nil
}

type experimentWorkspaceRuntime struct {
	manager *workspace.Manager
}

func (r experimentWorkspaceRuntime) WorkspaceDir(agentID string) (string, error) {
	status, err := r.manager.Status(agentID)
	if err == nil {
		return status.Workspace.MergedDir, nil
	}
	ws, err := r.manager.Create(agentID)
	if err != nil {
		return "", err
	}
	return ws.MergedDir, nil
}

func (r experimentWorkspaceRuntime) Commit(agentID string) error {
	return r.manager.Commit(agentID)
}

func (r experimentWorkspaceRuntime) Rollback(agentID string) error {
	_, err := r.manager.Rollback(agentID)
	return err
}

func (r experimentWorkspaceRuntime) Destroy(agentID string) error {
	return r.manager.Destroy(agentID)
}

func ensureRealCgroupRoot(root string) error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("cgroup fs is not cgroup2fs: requires linux, got %s", runtime.GOOS)
	}
	existing := root
	for {
		if _, err := os.Stat(existing); err == nil {
			break
		}
		parent := filepath.Dir(existing)
		if parent == existing {
			return fmt.Errorf("cgroup fs is not cgroup2fs: %s does not exist", root)
		}
		existing = parent
	}
	output, err := exec.Command("stat", "-fc", "%T", existing).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cgroup fs is not cgroup2fs: stat failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	if fsType := strings.TrimSpace(string(output)); fsType != "cgroup2fs" {
		return fmt.Errorf("cgroup fs is not cgroup2fs: %s", fsType)
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return fmt.Errorf("cgroup path is not writable: %w", err)
	}
	return nil
}

func readableFile(path string) bool {
	_, err := os.ReadFile(path)
	return err == nil
}

func fileContains(path, want string) bool {
	data, err := os.ReadFile(path)
	return err == nil && strings.Contains(string(data), want)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func fallbackOr(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func repoFingerprint() string {
	path := filepath.Join(repoRoot(), "go.mod")
	info, err := os.Stat(path)
	if err != nil {
		return "missing"
	}
	return fmt.Sprintf("%d:%d", info.Size(), info.ModTime().UnixNano())
}

func boolFromMap(values map[string]any, key string) bool {
	value, ok := values[key]
	if !ok {
		return false
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return typed == "true"
	default:
		return false
	}
}

func minDuration(left, right time.Duration) time.Duration {
	if left < right {
		return left
	}
	return right
}
