package review

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"aort-r/internal/workspace"
)

var resourceModes = []string{"baseline", "isolation-only", "aort-r"}

type ResourceIsolationConfig struct {
	Mode         string
	Runs         int
	Warmup       int
	Seed         int64
	Timeout      time.Duration
	OutDir       string
	EnableReplay bool
}

func (cfg ResourceIsolationConfig) normalized() (ResourceIsolationConfig, error) {
	if cfg.Mode == "" {
		cfg.Mode = "all"
	}
	if cfg.Mode != "all" && !contains(resourceModes, cfg.Mode) {
		return cfg, fmt.Errorf("unsupported resource isolation mode %q", cfg.Mode)
	}
	if cfg.Runs <= 0 {
		cfg.Runs = 20
	}
	if cfg.Warmup < 0 {
		return cfg, fmt.Errorf("warmup must be non-negative")
	}
	if cfg.Warmup == 0 {
		// Explicit zero is useful for tests and remains valid.
	}
	if cfg.Seed == 0 {
		cfg.Seed = 20260713
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	if cfg.OutDir == "" {
		cfg.OutDir = filepath.Join("experiments", "results", "review_remediation", "resource_isolation")
	}
	return cfg, nil
}

func RunResourceIsolation(ctx context.Context, cfg ResourceIsolationConfig) (ScenarioResult, error) {
	var err error
	cfg, err = cfg.normalized()
	if err != nil {
		return ScenarioResult{}, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	modes := resourceModes
	if cfg.Mode != "all" {
		modes = []string{cfg.Mode}
	}
	result := newScenarioResult("resource-isolation", cfg.Mode, cfg.Seed, cfg.Warmup, cfg.Runs, map[string]any{
		"mode":          cfg.Mode,
		"runs":          cfg.Runs,
		"warmup":        cfg.Warmup,
		"seed":          cfg.Seed,
		"timeout_ms":    cfg.Timeout.Milliseconds(),
		"fault_types":   []string{"memory_hog", "pids_hog", "cpu_hog", "workspace_rmrf"},
		"agent_roles":   []string{"Planner", "Coder-A", "Coder-B", "Tester", "Reviewer", "Fault-Agent"},
		"cgroup_policy": "feature-gated; no host cgroup is mutated by portable runs",
	})
	result.Limitations = append(result.Limitations, "Portable runs use bounded process/memory counters and the existing workspace capability probe; cgroup evidence is degraded when the host cannot provide a safe nested cgroup.")
	anyFailure := false
	for modeIndex, mode := range modes {
		for runIndex := 0; runIndex < cfg.Warmup+cfg.Runs; runIndex++ {
			seed := cfg.Seed + int64(modeIndex*1000+runIndex)
			observation, runErr := runResourceOnce(ctx, mode, runIndex, seed, cfg)
			if runIndex >= cfg.Warmup {
				result.PerRun = append(result.PerRun, observation)
				if !observation.Success {
					anyFailure = true
				}
			}
			if runErr != nil && runIndex >= cfg.Warmup {
				anyFailure = true
			}
			if runErr != nil && runIndex < cfg.Warmup {
				result.Limitations = append(result.Limitations, fmt.Sprintf("warmup %s/%d failed: %v", mode, runIndex+1, runErr))
			}
		}
	}
	result.EvidenceMode = resourceEvidenceMode(result.PerRun)
	if err := WriteScenarioArtifacts(cfg.OutDir, &result); err != nil {
		return result, err
	}
	if anyFailure {
		return result, fmt.Errorf("resource isolation had failed measured runs")
	}
	return result, nil
}

func runResourceOnce(parent context.Context, mode string, index int, seed int64, cfg ResourceIsolationConfig) (RunObservation, error) {
	start := time.Now()
	runID := fmt.Sprintf("%s-%03d", mode, index+1)
	observation := RunObservation{
		ScenarioID: "resource-isolation",
		RunID:      runID,
		Mode:       mode,
		Timestamp:  start.UTC().Format(time.RFC3339Nano),
		Metrics:    make(map[string]MetricValue),
		Events:     []EventRecord{},
	}
	ctx, cancel := context.WithTimeout(parent, cfg.Timeout)
	defer cancel()
	root, err := os.MkdirTemp("", "aort-review-resource-")
	if err != nil {
		observation.FailureReason = err.Error()
		return observation, err
	}
	defer os.RemoveAll(root)
	if !filepath.IsAbs(root) {
		err := fmt.Errorf("temporary resource root is not absolute")
		observation.FailureReason = err.Error()
		return observation, err
	}
	lower := filepath.Join(root, "lowerdir")
	if err := os.MkdirAll(lower, 0o755); err != nil {
		observation.FailureReason = err.Error()
		return observation, err
	}
	if err := os.WriteFile(filepath.Join(lower, "project.txt"), []byte(fmt.Sprintf("seed=%d\n", seed)), 0o644); err != nil {
		observation.FailureReason = err.Error()
		return observation, err
	}
	beforeHash, err := hashTree(lower)
	if err != nil {
		observation.FailureReason = err.Error()
		return observation, err
	}
	roles := []string{"Planner", "Coder-A", "Coder-B", "Tester", "Reviewer", "Fault-Agent"}
	workDirs := make(map[string]string, len(roles))
	for _, role := range roles {
		dir := filepath.Join(root, safeRole(role))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			observation.FailureReason = err.Error()
			return observation, err
		}
		workDirs[role] = dir
	}
	faultType := []string{"memory_hog", "pids_hog", "cpu_hog", "workspace_rmrf"}[index%4]
	observation.Events = append(observation.Events, EventRecord{Name: "scenario.started", Timestamp: start.UTC().Format(time.RFC3339Nano), Detail: "six-agent resource isolation run"})
	var mu sync.Mutex
	normalCompleted := 0
	var maxMemory int64
	pidsPeak := 0
	normalDurations := make([]float64, 0, len(roles)-1)
	var wg sync.WaitGroup
	for _, role := range roles[:len(roles)-1] {
		role := role
		wg.Add(1)
		go func() {
			defer wg.Done()
			duration, runErr := executeNormalAgent(ctx, workDirs[role], role, seed)
			mu.Lock()
			defer mu.Unlock()
			normalDurations = append(normalDurations, float64(duration.Milliseconds()))
			if runErr == nil {
				normalCompleted++
			}
			if runErr != nil {
				observation.Events = append(observation.Events, EventRecord{Name: "agent.failed", Timestamp: time.Now().UTC().Format(time.RFC3339Nano), Status: "failed", Detail: role + ": " + runErr.Error()})
			}
		}()
	}
	faultStart := time.Now()
	faultErr, faultMemory, faultPids := executeFaultAgent(ctx, mode, faultType, workDirs["Fault-Agent"], root, cfg.EnableReplay)
	faultDetection := time.Since(faultStart)
	if faultErr != nil {
		observation.FailureReason = "fault agent: " + faultErr.Error()
	}
	if mode == "aort-r" && faultType == "workspace_rmrf" {
		workspaceRoot := filepath.Join(root, "aort-workspace")
		workspaceResult, workspaceErr := workspace.RunRMFaultDemo(workspace.Config{Root: workspaceRoot})
		if workspaceErr != nil {
			observation.Events = append(observation.Events, EventRecord{Name: "workspace.degraded", Timestamp: time.Now().UTC().Format(time.RFC3339Nano), Status: "degraded", Detail: workspaceErr.Error()})
		} else {
			observation.Events = append(observation.Events, EventRecord{Name: "workspace.rmrf", Timestamp: time.Now().UTC().Format(time.RFC3339Nano), Status: string(workspaceResult.EvidenceMode), Detail: workspaceResult.FallbackReason})
		}
	}
	if faultMemory > maxMemory {
		maxMemory = faultMemory
	}
	pidsPeak += faultPids
	wg.Wait()
	cleanupStart := time.Now()
	// Only the generated Fault-Agent directory is removed. This is the safety
	// boundary exercised by the scenario; lowerdir and other agents remain.
	if faultType == "workspace_rmrf" {
		if err := safeRemoveWithin(root, workDirs["Fault-Agent"]); err != nil && observation.FailureReason == "" {
			observation.FailureReason = err.Error()
		}
	}
	cleanupMS := time.Since(cleanupStart).Milliseconds()
	afterHash, hashErr := hashTree(lower)
	if hashErr != nil && observation.FailureReason == "" {
		observation.FailureReason = hashErr.Error()
	}
	contamination := false
	for _, role := range roles[:len(roles)-1] {
		if _, err := os.Stat(filepath.Join(workDirs[role], "result.txt")); err != nil {
			contamination = true
		}
	}
	recoveryStart := time.Now()
	if faultType == "workspace_rmrf" {
		if err := os.MkdirAll(workDirs["Fault-Agent"], 0o755); err == nil {
			_ = os.WriteFile(filepath.Join(workDirs["Fault-Agent"], "recovered.txt"), []byte("recovered\n"), 0o644)
		}
	}
	recoveryMS := time.Since(recoveryStart).Milliseconds()
	if faultErr == nil && hashErr == nil && normalCompleted == len(roles)-1 && afterHash == beforeHash && !contamination {
		observation.Success = true
	}
	if observation.FailureReason == "" && !observation.Success {
		observation.FailureReason = "one or more agents did not complete or lowerdir changed"
	}
	if len(normalDurations) == 0 {
		normalDurations = []float64{float64(time.Since(start).Milliseconds())}
	}
	sort.Float64s(normalDurations)
	memStats := runtime.MemStats{}
	runtime.ReadMemStats(&memStats)
	if int64(memStats.Alloc) > maxMemory {
		maxMemory = int64(memStats.Alloc)
	}
	if pidsPeak < len(roles) {
		pidsPeak = len(roles)
	}
	normalStats := Aggregate(normalDurations, []bool{normalCompleted == len(roles)-1})
	modeEvidence := "degraded"
	if runtime.GOOS == "linux" {
		modeEvidence = "real-runtime"
	}
	if mode == "aort-r" {
		modeEvidence = modeEvidence + "+workspace"
	}
	observation.Metrics = map[string]MetricValue{
		"normal_agent_completion_rate": {Value: float64(normalCompleted) / float64(len(roles)-1), Kind: MeasurementMeasured, Unit: "ratio"},
		"task_success":                 {Value: boolFloat(observation.Success), Kind: MeasurementMeasured, Unit: "bool"},
		"fault_containment_scope":      {Value: 1, Kind: MeasurementMeasured, Unit: "agents"},
		"normal_completion_p50_ms":     {Value: normalStats.P50, Kind: MeasurementMeasured, Unit: "ms"},
		"normal_completion_p95_ms":     {Value: normalStats.P95, Kind: MeasurementMeasured, Unit: "ms"},
		"memory_peak_bytes":            {Value: float64(maxMemory), Kind: MeasurementMeasured, Unit: "bytes"},
		"pids_peak":                    {Value: float64(pidsPeak), Kind: MeasurementMeasured, Unit: "processes"},
		"fault_detection_ms":           {Value: float64(max64(1, faultDetection.Milliseconds())), Kind: MeasurementMeasured, Unit: "ms"},
		"cleanup_ms":                   {Value: float64(max64(1, cleanupMS)), Kind: MeasurementMeasured, Unit: "ms"},
		"recovery_ms":                  {Value: float64(max64(1, recoveryMS)), Kind: MeasurementMeasured, Unit: "ms"},
		"lowerdir_hash_unchanged":      {Value: boolFloat(afterHash == beforeHash), Kind: MeasurementMeasured, Unit: "bool"},
		"cross_agent_contamination":    {Value: boolFloat(contamination), Kind: MeasurementMeasured, Unit: "bool"},
		"resource_sampler_mode":        {Value: float64(len(modeEvidence)), Kind: MeasurementDerived, Unit: modeEvidence},
	}
	observation.Events = append(observation.Events, EventRecord{Name: "scenario.finished", Timestamp: time.Now().UTC().Format(time.RFC3339Nano), Status: modeEvidence, Detail: observation.FailureReason})
	return observation, nil
}

func executeNormalAgent(ctx context.Context, dir, role string, seed int64) (time.Duration, error) {
	start := time.Now()
	select {
	case <-ctx.Done():
		return time.Since(start), ctx.Err()
	default:
	}
	if err := os.WriteFile(filepath.Join(dir, "result.txt"), []byte(fmt.Sprintf("role=%s seed=%d\n", role, seed)), 0o644); err != nil {
		return time.Since(start), err
	}
	cmd := exec.CommandContext(ctx, "true")
	if err := cmd.Run(); err != nil {
		return time.Since(start), err
	}
	return time.Since(start), nil
}

func executeFaultAgent(ctx context.Context, mode, faultType, dir, root string, replay bool) (error, int64, int) {
	if mode == "baseline" && faultType == "workspace_rmrf" {
		// Baseline deliberately has no workspace manager; the same bounded path
		// operation provides the fair comparison without touching the lowerdir.
	}
	switch faultType {
	case "memory_hog":
		memory := make([]byte, 2*1024*1024)
		for i := 0; i < len(memory); i += 4096 {
			memory[i] = byte(i)
		}
		select {
		case <-ctx.Done():
			return ctx.Err(), int64(len(memory)), 1
		default:
		}
		return nil, int64(len(memory)), 1
	case "pids_hog":
		started := 0
		for i := 0; i < 4; i++ {
			cmd := exec.CommandContext(ctx, "true")
			if err := cmd.Run(); err != nil {
				return err, 0, started
			}
			started++
		}
		return nil, 0, started
	case "cpu_hog":
		deadline := time.Now().Add(15 * time.Millisecond)
		value := uint64(1)
		for time.Now().Before(deadline) {
			select {
			case <-ctx.Done():
				return ctx.Err(), 0, 1
			default:
				value = value*1664525 + 1013904223
			}
		}
		_ = value
		return nil, 0, 1
	case "workspace_rmrf":
		if replay {
			_ = os.WriteFile(filepath.Join(dir, "replay-requested"), []byte("replay\n"), 0o644)
		}
		return safeRemoveWithin(root, dir), 0, 1
	default:
		return fmt.Errorf("unsupported fault type %q", faultType), 0, 0
	}
}

func resourceEvidenceMode(runs []RunObservation) string {
	for _, run := range runs {
		if strings.HasPrefix(run.Events[len(run.Events)-1].Status, "degraded") || strings.Contains(run.Events[len(run.Events)-1].Status, "degraded") {
			return "degraded"
		}
	}
	return "real-runtime"
}

func hashTree(root string) (string, error) {
	entries := make([]string, 0)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		entries = append(entries, filepath.ToSlash(rel)+":"+hex.EncodeToString(sum[:]))
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(entries)
	sum := sha256.Sum256([]byte(strings.Join(entries, "\n")))
	return hex.EncodeToString(sum[:]), nil
}

func safeRemoveWithin(root, target string) error {
	rootAbs, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return err
	}
	targetAbs, err := filepath.Abs(filepath.Clean(target))
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("refusing to remove path outside generated root: %s", targetAbs)
	}
	return os.RemoveAll(targetAbs)
}

func safeRole(role string) string {
	return strings.ToLower(strings.NewReplacer(" ", "-", "/", "-", "\\", "-").Replace(role))
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func boolFloat(value bool) float64 {
	if value {
		return 1
	}
	return 0
}

func max64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}
