package experiment

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"aort-r/internal/evidence"
	"aort-r/internal/workspace"
)

type AllExperimentsConfig struct {
	Runs   int
	OutDir string
}

type AllExperimentStep struct {
	Name           string   `json:"name"`
	Command        string   `json:"command"`
	Status         string   `json:"status"`
	GeneratedFiles []string `json:"generated_files"`
	Error          string   `json:"error"`
	FallbackReason string   `json:"fallback_reason,omitempty"`
}

type AllExperimentsSummary struct {
	Experiment        string              `json:"experiment"`
	Runs              int                 `json:"runs"`
	EvidenceMode      string              `json:"evidence_mode"`
	Steps             []AllExperimentStep `json:"steps"`
	Passed            []string            `json:"passed"`
	Failed            []string            `json:"failed"`
	Degraded          []string            `json:"degraded"`
	Skipped           []string            `json:"skipped"`
	Missing           []string            `json:"missing"`
	GeneratedFiles    []string            `json:"generated_files"`
	AllRequiredPassed bool                `json:"all_required_passed"`
}

type FinalEvidenceIndex struct {
	Timestamp                string            `json:"timestamp"`
	Git                      FinalGitInfo      `json:"git"`
	System                   FinalSystemInfo   `json:"system"`
	GitCommit                string            `json:"git_commit"`
	GitBranch                string            `json:"git_branch"`
	GitDirty                 bool              `json:"git_dirty"`
	OSRelease                string            `json:"os_release"`
	Kernel                   string            `json:"kernel"`
	CgroupFSType             string            `json:"cgroup_fs_type"`
	GoVersion                string            `json:"go_version"`
	GenericCompetitionVerify map[string]string `json:"generic_competition_verify"`
	RealOnlyOpenEuler        map[string]string `json:"real_only_openEuler"`
	EvidenceModeSummary      map[string]string `json:"evidence_mode_summary"`
	RealOnlySummary          map[string]bool   `json:"real_only_summary"`
	EBPFObserver             string            `json:"ebpf_observer"`
	IPCShm                   string            `json:"ipc_shm"`
	CVMMemory                string            `json:"cvm_memory"`
	Replay                   string            `json:"replay"`
	DeepSeekRealSmoke        string            `json:"deepseek_real_smoke"`
	AllowedDegraded          map[string]string `json:"allowed_degraded"`
	GeneratedFiles           []string          `json:"generated_files"`
	MissingFiles             []string          `json:"missing_files"`
	KnownLimits              []string          `json:"known_limits"`
}

type FinalGitInfo struct {
	Commit string `json:"commit"`
	Branch string `json:"branch"`
	Dirty  bool   `json:"dirty"`
}

type FinalSystemInfo struct {
	OSRelease    string `json:"os_release"`
	Kernel       string `json:"kernel"`
	CgroupFSType string `json:"cgroup_fs_type"`
	GoVersion    string `json:"go_version"`
}

type allStepSpec struct {
	name     string
	command  []string
	expected []string
	run      func() error
}

func RunAllExperiments(cfg AllExperimentsConfig) (AllExperimentsSummary, error) {
	if cfg.Runs <= 0 {
		cfg.Runs = 1
	}
	if cfg.OutDir == "" {
		cfg.OutDir = filepath.Join("experiments", "results", "all")
	}
	runText := fmt.Sprintf("%d", cfg.Runs)
	specs := []allStepSpec{
		{
			name:    "e1",
			command: []string{"go", "run", "./cmd/aortctl", "experiment", "e1", "--policy", "resource-aware", "--runs", runText, "--out", filepath.Join(cfg.OutDir, "e1")},
			expected: []string{
				filepath.Join(cfg.OutDir, "e1", "e1_resource_aware.json"),
				filepath.Join(cfg.OutDir, "e1", "e1_resource_aware.csv"),
				filepath.Join(cfg.OutDir, "e1", "e1_resource_aware_decisions.json"),
				filepath.Join(cfg.OutDir, "e1", "e1_resource_aware_summary.md"),
			},
			run: func() error {
				_, err := RunE1ResourceAwareBenchmark(cfg.Runs, filepath.Join(cfg.OutDir, "e1"))
				return err
			},
		},
		{
			name:     "e1-pressure",
			command:  []string{"go", "run", "./cmd/aortctl", "experiment", "e1-pressure", "--runs", runText, "--out", filepath.Join(cfg.OutDir, "e1_pressure")},
			expected: []string{filepath.Join(cfg.OutDir, "e1_pressure", "e1_pressure.json")},
			run: func() error {
				_, err := RunE1PressureBenchmark(cfg.Runs, filepath.Join(cfg.OutDir, "e1_pressure"))
				return err
			},
		},
		{
			name: "e2",
			command: []string{"go", "run", "./cmd/aortctl", "experiment", "e2", "--runs", runText,
				"--out", filepath.Join(cfg.OutDir, "e2")},
			expected: []string{filepath.Join(cfg.OutDir, "e2", "e2-real-fault.json"), filepath.Join(cfg.OutDir, "e2", "e2-real-fault.csv")},
			run: func() error {
				results := RunE2RealFaultIsolation(cfg.Runs)
				if err := WriteJSON(filepath.Join(cfg.OutDir, "e2", "e2-real-fault.json"), results); err != nil {
					return err
				}
				return WriteCSV(filepath.Join(cfg.OutDir, "e2", "e2-real-fault.csv"), E2RealCSV(results))
			},
		},
		{
			name:     "e2-pressure-fault",
			command:  []string{"go", "run", "./cmd/aortctl", "experiment", "e2-pressure-fault", "--runs", runText, "--out", filepath.Join(cfg.OutDir, "e2_pressure_fault")},
			expected: []string{filepath.Join(cfg.OutDir, "e2_pressure_fault", "e2_pressure_fault.json")},
			run: func() error {
				_, err := RunE2PressureFault(cfg.Runs, filepath.Join(cfg.OutDir, "e2_pressure_fault"))
				return err
			},
		},
		{
			name:     "software-real",
			command:  []string{"go", "run", "./cmd/aortctl", "demo", "software-real", "--out", filepath.Join(cfg.OutDir, "software_real")},
			expected: []string{filepath.Join(cfg.OutDir, "software_real", "software_real_demo", "result.json")},
			run: func() error {
				_, err := RunSoftwareRealDemo(cfg.Runs, filepath.Join(cfg.OutDir, "software_real"))
				return err
			},
		},
		{
			name:     "workspace probe",
			command:  []string{"go", "run", "./cmd/aortctl", "workspace", "probe", "--out", filepath.Join(cfg.OutDir, "workspace_probe.json")},
			expected: []string{filepath.Join(cfg.OutDir, "workspace_probe.json")},
			run: func() error {
				return WriteJSON(filepath.Join(cfg.OutDir, "workspace_probe.json"), workspace.ProbeOverlay(""))
			},
		},
		{
			name:     "workspace-rmrf",
			command:  []string{"go", "run", "./cmd/aortctl", "demo", "fault", "workspace-rmrf", "--out", filepath.Join(cfg.OutDir, "workspace_rmrf"), "--root", filepath.Join(cfg.OutDir, "workspace_rmrf_runtime")},
			expected: []string{filepath.Join(cfg.OutDir, "workspace_rmrf", "workspace_isolation_evidence.json")},
			run: func() error {
				result, err := workspace.RunRMFaultDemo(workspace.Config{Root: filepath.Join(cfg.OutDir, "workspace_rmrf_runtime")})
				if err != nil {
					return err
				}
				return WriteJSON(filepath.Join(cfg.OutDir, "workspace_rmrf", "workspace_isolation_evidence.json"), result)
			},
		},
		{
			name:     "tool-workspace",
			command:  []string{"go", "run", "./cmd/aortctl", "demo", "tool-workspace", "--out", filepath.Join(cfg.OutDir, "tool_workspace")},
			expected: []string{filepath.Join(cfg.OutDir, "tool_workspace", "tool_workspace_evidence.json")},
			run: func() error {
				_, err := RunToolWorkspaceDemo(workspace.Config{}, filepath.Join(cfg.OutDir, "tool_workspace"), false)
				return err
			},
		},
		{
			name:     "real-cgroup-smoke",
			command:  []string{"go", "run", "./cmd/aortctl", "experiment", "real-cgroup-smoke", "--out", filepath.Join(cfg.OutDir, "real_cgroup_smoke")},
			expected: []string{filepath.Join(cfg.OutDir, "real_cgroup_smoke", "real_cgroup_smoke.json")},
			run: func() error {
				_, err := RunRealCgroupSmoke(RealCgroupSmokeConfig{OutDir: filepath.Join(cfg.OutDir, "real_cgroup_smoke")})
				return err
			},
		},
		{
			name:     "real-pressure-smoke",
			command:  []string{"go", "run", "./cmd/aortctl", "experiment", "real-pressure-smoke", "--runs", runText, "--out", filepath.Join(cfg.OutDir, "real_pressure_smoke")},
			expected: []string{filepath.Join(cfg.OutDir, "real_pressure_smoke", "real_pressure_smoke.json")},
			run: func() error {
				_, err := RunRealPressureSmoke(RealPressureSmokeConfig{Runs: cfg.Runs, OutDir: filepath.Join(cfg.OutDir, "real_pressure_smoke")})
				return err
			},
		},
		{
			name:     "deepseek-real-smoke",
			command:  []string{"go", "run", "./cmd/aortctl", "experiment", "deepseek-real-smoke", "--out", filepath.Join(cfg.OutDir, "deepseek_real")},
			expected: []string{filepath.Join(cfg.OutDir, "deepseek_real", "deepseek_real_smoke.json")},
			run: func() error {
				_, err := RunDeepSeekRealSmoke(DeepSeekRealSmokeConfigFromEnv(filepath.Join(cfg.OutDir, "deepseek_real")))
				return err
			},
		},
		{
			name:     "ebpf-smoke",
			command:  []string{"go", "run", "./cmd/aortctl", "observer", "ebpf-smoke", "--out", filepath.Join(cfg.OutDir, "ebpf_smoke")},
			expected: []string{filepath.Join(cfg.OutDir, "ebpf_smoke", "ebpf_smoke.json")},
			run: func() error {
				_, err := RunEBPFSmoke(filepath.Join(cfg.OutDir, "ebpf_smoke"))
				return err
			},
		},
		{
			name:     "ipc shm-smoke",
			command:  []string{"go", "run", "./cmd/aortctl", "ipc", "shm-smoke", "--out", filepath.Join(cfg.OutDir, "ipc_shm")},
			expected: []string{filepath.Join(cfg.OutDir, "ipc_shm", "ipc_shm_smoke.json")},
			run: func() error {
				_, err := RunIPCShmSmoke(filepath.Join(cfg.OutDir, "ipc_shm"))
				return err
			},
		},
		{
			name:     "cvm memory-smoke",
			command:  []string{"go", "run", "./cmd/aortctl", "cvm", "memory-smoke", "--out", filepath.Join(cfg.OutDir, "cvm_memory")},
			expected: []string{filepath.Join(cfg.OutDir, "cvm_memory", "cvm_memory_smoke.json")},
			run: func() error {
				_, err := RunCVMMemorySmoke(filepath.Join(cfg.OutDir, "cvm_memory"))
				return err
			},
		},
		{
			name:     "real-all",
			command:  []string{"go", "run", "./cmd/aortctl", "experiment", "real-all", "--runs", runText, "--out", filepath.Join(cfg.OutDir, "real_all")},
			expected: []string{filepath.Join(cfg.OutDir, "real_all", "REAL_EVIDENCE_INDEX.json")},
			run: func() error {
				_, err := RunRealAll(cfg.Runs, filepath.Join(cfg.OutDir, "real_all"))
				return err
			},
		},
	}

	summary := AllExperimentsSummary{
		Experiment:     "all",
		Runs:           cfg.Runs,
		EvidenceMode:   "real-runtime",
		Steps:          []AllExperimentStep{},
		Passed:         []string{},
		Failed:         []string{},
		Degraded:       []string{},
		Skipped:        []string{},
		Missing:        []string{},
		GeneratedFiles: []string{},
	}
	for _, spec := range specs {
		step := AllExperimentStep{Name: spec.name, Command: quoteCommand(spec.command)}
		err := spec.run()
		step.GeneratedFiles = existingFiles(spec.expected)
		skipped, skipReason := inspectSkippedEvidence(step.GeneratedFiles)
		degraded, fallback := inspectEvidenceFiles(step.GeneratedFiles)
		switch {
		case skipped:
			step.Status = "skipped"
			step.FallbackReason = skipReason
		case len(step.GeneratedFiles) == 0:
			step.Status = "missing"
			step.Error = "expected output files were not generated"
		case err != nil && isOrdinaryAllDegraded(spec.name, err):
			step.Status = "degraded"
			step.Error = err.Error()
			step.FallbackReason = fallbackOr(fallback, err.Error())
		case err != nil:
			step.Status = "failed"
			step.Error = err.Error()
		case degraded:
			step.Status = "degraded"
			step.FallbackReason = fallbackOr(fallback, "step produced degraded evidence")
		default:
			step.Status = "passed"
		}
		if step.Status == "degraded" && step.FallbackReason == "" {
			step.FallbackReason = fallbackOr(step.Error, "step produced degraded evidence")
		}
		summary.Steps = append(summary.Steps, step)
		summary.GeneratedFiles = append(summary.GeneratedFiles, step.GeneratedFiles...)
		switch step.Status {
		case "passed":
			summary.Passed = append(summary.Passed, step.Name)
		case "failed":
			summary.Failed = append(summary.Failed, step.Name)
		case "degraded":
			summary.Degraded = append(summary.Degraded, step.Name)
		case "skipped":
			summary.Skipped = append(summary.Skipped, step.Name)
		case "missing":
			summary.Missing = append(summary.Missing, step.Name)
		}
	}
	if len(summary.Degraded) > 0 {
		summary.EvidenceMode = "degraded"
	}
	summary.AllRequiredPassed = len(summary.Failed) == 0 && len(summary.Missing) == 0
	summaryPath := filepath.Join(cfg.OutDir, "all_experiments_summary.json")
	summary.GeneratedFiles = append(summary.GeneratedFiles, summaryPath)
	if err := WriteJSON(summaryPath, summary); err != nil {
		return summary, err
	}
	if !summary.AllRequiredPassed {
		return summary, fmt.Errorf("experiment all had failed or missing steps: failed=%v missing=%v", summary.Failed, summary.Missing)
	}
	return summary, nil
}

func WriteFinalEvidence(outDir string) (FinalEvidenceIndex, error) {
	if outDir == "" {
		outDir = filepath.Join("experiments", "results", "final")
	}
	root := repoRoot()
	statuses := readStepStatuses(filepath.Join(root, "experiments", "results", "final", "step_status.tsv"))
	required := finalRequiredFiles(root)
	git := collectGitInfo(root)
	system := collectSystemInfo()

	generic := map[string]string{
		"go_test":             statusFromStepOrFiles(statuses, "go_test", nil, false),
		"smoke":               statusFromStepOrFiles(statuses, "smoke", nil, true),
		"e1_scheduler":        statusFromStepOrFiles(statuses, "e1_scheduler", required["e1_scheduler"], false),
		"e1_pressure":         statusFromStepOrFiles(statuses, "e1_pressure", required["e1_pressure"], false),
		"e2_fault_isolation":  statusFromStepOrFiles(statuses, "e2_fault_isolation", required["e2_fault_isolation"], false),
		"e2_pressure_fault":   statusFromStepOrFiles(statuses, "e2_pressure_fault", required["e2_pressure_fault"], false),
		"software_real_demo":  statusFromStepOrFiles(statuses, "software_real_demo", required["software_real_demo"], false),
		"workspace_probe":     statusFromStepOrFiles(statuses, "workspace_probe", required["workspace_probe"], true),
		"workspace_isolation": statusFromStepOrFiles(statuses, "workspace_isolation", required["workspace_rmrf"], true),
		"ebpf_observer":       statusFromStepOrFiles(statuses, "ebpf_observer", required["ebpf_observer"], true),
		"ipc_shm":             statusFromStepOrFiles(statuses, "ipc_shm", required["ipc_shm"], true),
		"cvm_memory":          statusFromStepOrFiles(statuses, "cvm_memory", required["cvm_memory"], false),
		"replay":              statusFromStepOrFiles(statuses, "replay", required["replay"], false),
	}
	allowedDegraded := allowedDegradedEvidence(root, generic)
	for key := range allowedDegraded {
		if generic[key] == "degraded" {
			generic[key] = "allowed_degraded"
		}
	}
	deepSeekRequired := os.Getenv("AORT_ENABLE_REAL_LLM") == "1"
	realOnly := map[string]string{
		"real_env":            statusFromJSON(filepath.Join(root, "experiments", "results", "real_env", "real_openeuler_env.json"), realEnvPassed),
		"real_cgroup_smoke":   statusFromJSON(filepath.Join(root, "experiments", "results", "real_cgroup_smoke", "real_cgroup_smoke.json"), realCgroupSmokePassed),
		"real_pressure_smoke": statusFromJSON(filepath.Join(root, "experiments", "results", "real_pressure_smoke", "real_pressure_smoke.json"), realPressureSmokePassed),
		"deepseek_real_smoke": statusFromDeepSeekRealSmoke(filepath.Join(root, "experiments", "results", "deepseek_real", "deepseek_real_smoke.json"), deepSeekRequired),
		"workspace_probe":     statusFromJSON(filepath.Join(root, "experiments", "results", "workspace_probe.json"), workspaceProbePassed),
		"workspace_rmrf":      statusFromJSON(filepath.Join(root, "experiments", "results", "workspace_isolation_evidence.json"), workspaceRMFaultPassed),
		"tool_workspace":      statusFromJSON(filepath.Join(root, "experiments", "results", "tool_workspace_evidence.json"), toolWorkspacePassed),
		"real_all":            statusFromJSON(filepath.Join(root, "experiments", "results", "real_all", "REAL_EVIDENCE_INDEX.json"), realAllPassed),
	}
	realEnv, _ := readJSONMap(filepath.Join(root, "experiments", "results", "real_env", "real_openeuler_env.json"))
	realAll, _ := readJSONMap(filepath.Join(root, "experiments", "results", "real_all", "REAL_EVIDENCE_INDEX.json"))
	evidenceModeSummary := buildEvidenceModeSummary(root, realOnly)
	generated, missing := splitExistingFiles(flattenRequired(required, realOnlyFiles(root)))
	knownLimits := buildKnownLimits(root, realOnly, generic)
	realOnlyAllPassed := allRealOnlyRequiredPassed(realOnly, boolField(realAll, "all_passed"), deepSeekRequired)
	index := FinalEvidenceIndex{
		Timestamp:                time.Now().UTC().Format(time.RFC3339Nano),
		Git:                      git,
		System:                   system,
		GitCommit:                git.Commit,
		GitBranch:                git.Branch,
		GitDirty:                 git.Dirty,
		OSRelease:                system.OSRelease,
		Kernel:                   system.Kernel,
		CgroupFSType:             system.CgroupFSType,
		GoVersion:                system.GoVersion,
		GenericCompetitionVerify: generic,
		RealOnlyOpenEuler:        realOnly,
		EvidenceModeSummary:      evidenceModeSummary,
		RealOnlySummary: map[string]bool{
			"openEuler":                boolField(realEnv, "openEuler"),
			"cgroup2fs":                boolField(realEnv, "cgroup2fs"),
			"root":                     boolField(realEnv, "root"),
			"cgroup_kill_supported":    boolField(realEnv, "cgroup_kill_supported"),
			"overlayfs_mount_success":  boolField(realEnv, "overlayfs_mount_success"),
			"real_cgroup_v2":           realOnly["real_cgroup_smoke"] == "passed",
			"real_resource_sampler":    realOnly["real_pressure_smoke"] == "passed",
			"deepseek_real_api":        realOnly["deepseek_real_smoke"] == "passed",
			"real_overlayfs":           realOnly["workspace_probe"] == "passed" && realOnly["workspace_rmrf"] == "passed",
			"real_workspace_tool_exec": realOnly["tool_workspace"] == "passed",
			"all_passed":               realOnlyAllPassed,
		},
		EBPFObserver:      evidenceModeFromPath(filepath.Join(root, "experiments", "results", "ebpf_smoke", "ebpf_smoke.json"), string(evidence.ModeDegraded)),
		IPCShm:            evidenceModeFromPath(filepath.Join(root, "experiments", "results", "ipc_shm", "ipc_shm_smoke.json"), "missing"),
		CVMMemory:         evidenceModeFromPath(filepath.Join(root, "experiments", "results", "cvm_memory", "cvm_memory_smoke.json"), "missing"),
		Replay:            evidenceModeFromPath(filepath.Join(root, "experiments", "results", "replay", "replay_result.json"), "missing"),
		DeepSeekRealSmoke: realOnly["deepseek_real_smoke"],
		AllowedDegraded:   allowedDegraded,
		GeneratedFiles:    generated,
		MissingFiles:      missing,
		KnownLimits:       knownLimits,
	}
	indexPath := filepath.Join(outDir, "FINAL_EVIDENCE_INDEX.json")
	summaryPath := filepath.Join(outDir, "FINAL_SUMMARY.md")
	index.GeneratedFiles = appendUnique(append(index.GeneratedFiles, indexPath, summaryPath)...)
	if err := WriteJSON(indexPath, index); err != nil {
		return index, err
	}
	return index, os.WriteFile(summaryPath, []byte(renderFinalSummary(index)), 0o644)
}

func quoteCommand(parts []string) string {
	return strings.Join(parts, " ")
}

func existingFiles(paths []string) []string {
	files := []string{}
	for _, path := range paths {
		if fileExists(path) {
			files = append(files, path)
		}
	}
	return files
}

func inspectEvidenceFiles(paths []string) (bool, string) {
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var value any
		if json.Unmarshal(data, &value) != nil {
			continue
		}
		if degraded, fallback := inspectEvidenceValue(value); degraded {
			return true, fallback
		}
	}
	return false, ""
}

func inspectSkippedEvidence(paths []string) (bool, string) {
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var value any
		if json.Unmarshal(data, &value) != nil {
			continue
		}
		if skipped, reason := inspectSkippedValue(value); skipped {
			return true, reason
		}
	}
	return false, ""
}

func inspectSkippedValue(value any) (bool, string) {
	switch typed := value.(type) {
	case map[string]any:
		if stringField(typed, "status") == "skipped" {
			return true, fallbackOr(stringField(typed, "failure_reason"), "step skipped")
		}
		for _, child := range typed {
			if skipped, reason := inspectSkippedValue(child); skipped {
				return true, reason
			}
		}
	case []any:
		for _, child := range typed {
			if skipped, reason := inspectSkippedValue(child); skipped {
				return true, reason
			}
		}
	}
	return false, ""
}

func inspectEvidenceValue(value any) (bool, string) {
	switch typed := value.(type) {
	case map[string]any:
		fallback := ""
		if raw, ok := typed["fallback_reason"].(string); ok && raw != "" {
			fallback = raw
		}
		if raw, ok := typed["failure_reason"].(string); ok && raw != "" {
			fallback = fallbackOr(fallback, raw)
		}
		if raw, ok := typed["error"].(string); ok && raw != "" {
			fallback = fallbackOr(fallback, raw)
		}
		mode := ""
		modeDegraded := false
		if raw, ok := typed["evidence_mode"].(string); ok {
			mode = raw
			modeDegraded = degradedEvidenceMode(raw)
		}
		for _, child := range typed {
			if degraded, reason := inspectEvidenceValue(child); degraded {
				if fallback == "" || fallback == mode {
					fallback = reason
				}
				if !modeDegraded {
					return true, fallback
				}
			}
		}
		if modeDegraded {
			return true, fallbackOr(fallback, mode)
		}
		if fallback != "" {
			return true, fallback
		}
	case []any:
		for _, child := range typed {
			if degraded, reason := inspectEvidenceValue(child); degraded {
				return true, reason
			}
		}
	}
	return false, ""
}

func degradedEvidenceMode(mode string) bool {
	return mode == string(evidence.ModeMissing) || strings.Contains(mode, "degraded")
}

func isOrdinaryAllDegraded(name string, err error) bool {
	if strings.HasPrefix(name, "real-") || strings.HasPrefix(name, "workspace") || name == "tool-workspace" {
		return true
	}
	text := err.Error()
	for _, part := range []string{"not openEuler", "cgroup fs is not cgroup2fs", "requires linux", "no root permission", "overlayfs"} {
		if strings.Contains(text, part) {
			return true
		}
	}
	return false
}

func readStepStatuses(path string) map[string]string {
	statuses := map[string]string{}
	data, err := os.ReadFile(path)
	if err != nil {
		return statuses
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) >= 2 && fields[0] != "" {
			statuses[fields[0]] = fields[1]
		}
	}
	return statuses
}

func finalRequiredFiles(root string) map[string][]string {
	results := filepath.Join(root, "experiments", "results")
	return map[string][]string{
		"e1_scheduler":        {filepath.Join(results, "e1", "e1_resource_aware.json"), filepath.Join(results, "e1", "e1_resource_aware.csv"), filepath.Join(results, "e1", "e1_resource_aware_decisions.json"), filepath.Join(results, "e1", "e1_resource_aware_summary.md")},
		"e1_pressure":         {filepath.Join(results, "e1_pressure", "e1_pressure.json")},
		"e2_fault_isolation":  {filepath.Join(results, "e2-real-fault.json"), filepath.Join(results, "e2-real-fault.csv")},
		"e2_pressure_fault":   {filepath.Join(results, "e2_pressure_fault", "e2_pressure_fault.json")},
		"software_real_demo":  {filepath.Join(results, "software_real_demo", "result.json")},
		"workspace_probe":     {filepath.Join(results, "workspace_probe.json")},
		"workspace_rmrf":      {filepath.Join(results, "workspace_isolation_evidence.json")},
		"ebpf_observer":       {filepath.Join(results, "ebpf_smoke", "ebpf_smoke.json")},
		"ipc_shm":             {filepath.Join(results, "ipc_shm", "ipc_shm_smoke.json")},
		"cvm_memory":          {filepath.Join(results, "cvm_memory", "cvm_memory_smoke.json")},
		"deepseek_real_smoke": {filepath.Join(results, "deepseek_real", "deepseek_real_smoke.json")},
		"replay":              {filepath.Join(results, "replay", "replay_result.json")},
	}
}

func realOnlyFiles(root string) []string {
	results := filepath.Join(root, "experiments", "results")
	return []string{
		filepath.Join(results, "real_env", "real_openeuler_env.json"),
		filepath.Join(results, "real_cgroup_smoke", "real_cgroup_smoke.json"),
		filepath.Join(results, "real_pressure_smoke", "real_pressure_smoke.json"),
		filepath.Join(results, "deepseek_real", "deepseek_real_smoke.json"),
		filepath.Join(results, "workspace_probe.json"),
		filepath.Join(results, "workspace_isolation_evidence.json"),
		filepath.Join(results, "tool_workspace_evidence.json"),
		filepath.Join(results, "ebpf_smoke", "ebpf_smoke.json"),
		filepath.Join(results, "ipc_shm", "ipc_shm_smoke.json"),
		filepath.Join(results, "cvm_memory", "cvm_memory_smoke.json"),
		filepath.Join(results, "replay", "replay_result.json"),
		filepath.Join(results, "real_all", "REAL_EVIDENCE_INDEX.json"),
	}
}

func statusFromStepOrFiles(statuses map[string]string, step string, paths []string, allowDegraded bool) string {
	if status := statuses[step]; status != "" {
		if status == "passed" && anyMissing(paths) {
			return "missing"
		}
		return status
	}
	if len(paths) == 0 {
		return "missing"
	}
	if anyMissing(paths) {
		return "missing"
	}
	if allowDegraded {
		if degraded, _ := inspectEvidenceFiles(paths); degraded {
			return "degraded"
		}
	}
	return "passed"
}

func anyMissing(paths []string) bool {
	for _, path := range paths {
		if !fileExists(path) {
			return true
		}
	}
	return false
}

func statusFromJSON(path string, passed func(map[string]any) bool) string {
	data, ok := readJSONMap(path)
	if !ok {
		return "missing"
	}
	if passed(data) {
		return "passed"
	}
	return "failed"
}

func readJSONMap(path string) (map[string]any, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, false
	}
	return value, true
}

func realEnvPassed(data map[string]any) bool {
	for _, key := range []string{"openEuler", "cgroup2fs", "root", "cgroup_writable", "memory_current_readable", "pids_current_readable", "cpu_stat_readable", "cgroup_kill_supported", "overlayfs_mount_success"} {
		if !boolField(data, key) {
			return false
		}
	}
	return stringField(data, "evidence_mode") == "real-openeuler"
}

func realCgroupSmokePassed(data map[string]any) bool {
	return stringField(data, "evidence_mode") == string(evidence.ModeRealCgroupV2) &&
		boolField(data, "worker_pid_attached") &&
		boolField(data, "memory_current_readable") &&
		boolField(data, "pids_current_readable") &&
		boolField(data, "cpu_stat_readable") &&
		boolField(data, "freeze_success") &&
		boolField(data, "unfreeze_success") &&
		boolField(data, "cgroup_kill_success") &&
		boolField(data, "destroy_success")
}

func realPressureSmokePassed(data map[string]any) bool {
	return stringField(data, "evidence_mode") == string(evidence.ModeRealCgroupV2) &&
		stringField(data, "resource_sampler_mode") == string(evidence.ModeRealCgroupV2) &&
		boolField(data, "memory_hog_detected") &&
		boolField(data, "pids_hog_detected") &&
		boolField(data, "cpu_pressure_detected") &&
		boolField(data, "resource_aware_avoided_high_pressure_agent") &&
		numberField(data, "selected_high_pressure_agent_count") == 0 &&
		!boolField(data, "cascade_failure") &&
		boolField(data, "cleanup_success")
}

func workspaceProbePassed(data map[string]any) bool {
	return stringField(data, "evidence_mode") == string(evidence.ModeRealOverlayFS) &&
		boolField(data, "linux") &&
		boolField(data, "overlay_in_proc_filesystems") &&
		boolField(data, "mount_test_success") &&
		boolField(data, "merged_is_mountpoint") &&
		stringField(data, "selected_mode") == "overlayfs"
}

func workspaceRMFaultPassed(data map[string]any) bool {
	return stringField(data, "evidence_mode") == string(evidence.ModeRealOverlayFS) &&
		stringField(data, "mode") == "overlayfs" &&
		boolField(data, "lowerdir_unchanged") &&
		boolField(data, "target_agent_affected") &&
		!boolField(data, "cascade_failure") &&
		boolField(data, "rollback_success") &&
		boolField(data, "merged_is_mountpoint")
}

func toolWorkspacePassed(data map[string]any) bool {
	return stringField(data, "evidence_mode") == string(evidence.ModeRealOverlayFS) &&
		boolField(data, "tool_exec_uses_workspace") &&
		boolField(data, "cwd_is_merged") &&
		boolField(data, "commit_on_success") &&
		boolField(data, "rollback_on_failure") &&
		boolField(data, "repo_dir_untouched") &&
		boolField(data, "path_escape_blocked") &&
		boolField(data, "symlink_escape_blocked")
}

func realAllPassed(data map[string]any) bool {
	return stringField(data, "evidence_mode") == "real-openeuler" && boolField(data, "all_passed")
}

func statusFromDeepSeekRealSmoke(path string, required bool) string {
	data, ok := readJSONMap(path)
	if !ok {
		if required {
			return "missing"
		}
		return "skipped"
	}
	if deepSeekRealSmokePassed(data) {
		return "passed"
	}
	if stringField(data, "status") == "skipped" && !required {
		return "skipped"
	}
	return "failed"
}

func deepSeekRealSmokePassed(data map[string]any) bool {
	return stringField(data, "experiment") == "deepseek_real_smoke" &&
		stringField(data, "status") == "passed" &&
		stringField(data, "evidence_mode") == string(evidence.ModeRealAPI) &&
		stringField(data, "provider") == "deepseek" &&
		!boolField(data, "llm_mock") &&
		boolField(data, "request_success") &&
		boolField(data, "response_non_empty") &&
		numberField(data, "status_code") == 200 &&
		numberField(data, "latency_ms") > 0 &&
		stringField(data, "api_key_source") == "env" &&
		boolField(data, "api_key_present") &&
		boolField(data, "api_key_redacted") &&
		boolField(data, "cleanup_success")
}

func allRealOnlyRequiredPassed(realOnly map[string]string, realAllPassed bool, deepSeekRequired bool) bool {
	if !realAllPassed {
		return false
	}
	required := []string{"real_env", "real_cgroup_smoke", "real_pressure_smoke", "workspace_probe", "workspace_rmrf", "tool_workspace", "real_all"}
	if deepSeekRequired {
		required = append(required, "deepseek_real_smoke")
	}
	for _, key := range required {
		if realOnly[key] != "passed" {
			return false
		}
	}
	return true
}

func buildEvidenceModeSummary(root string, realOnly map[string]string) map[string]string {
	workspaceMode := "missing"
	if probe, ok := readJSONMap(filepath.Join(root, "experiments", "results", "workspace_probe.json")); ok {
		workspaceMode = fallbackOr(stringField(probe, "evidence_mode"), workspaceMode)
	}
	toolMode := "missing"
	if tool, ok := readJSONMap(filepath.Join(root, "experiments", "results", "tool_workspace_evidence.json")); ok {
		toolMode = fallbackOr(stringField(tool, "evidence_mode"), toolMode)
	}
	cgroupMode := "degraded"
	if realOnly["real_cgroup_smoke"] == "passed" {
		cgroupMode = string(evidence.ModeRealCgroupV2)
	}
	samplerMode := "degraded"
	if realOnly["real_pressure_smoke"] == "passed" {
		samplerMode = string(evidence.ModeRealCgroupV2)
	}
	llmMode := string(evidence.ModeMock)
	switch realOnly["deepseek_real_smoke"] {
	case "passed":
		llmMode = string(evidence.ModeRealAPI)
	case "failed":
		llmMode = "failed"
	}
	return map[string]string{
		"cgroup_capsule":      cgroupMode,
		"worker_process":      string(evidence.ModeRealRuntime),
		"resource_sampler":    samplerMode,
		"workspace_overlayfs": workspaceMode,
		"tool_workspace":      toolMode,
		"scheduler":           string(evidence.ModeRealRuntime),
		"cvm":                 evidenceModeFromPath(filepath.Join(root, "experiments", "results", "cvm_memory", "cvm_memory_smoke.json"), "real-partial"),
		"ipc":                 "real-partial + " + evidenceModeFromPath(filepath.Join(root, "experiments", "results", "ipc_shm", "ipc_shm_smoke.json"), "real-shm-ipc optional"),
		"llm":                 llmMode,
		"ebpf":                evidenceModeFromPath(filepath.Join(root, "experiments", "results", "ebpf_smoke", "ebpf_smoke.json"), string(evidence.ModeDegraded)),
		"replay":              evidenceModeFromPath(filepath.Join(root, "experiments", "results", "replay", "replay_result.json"), string(evidence.ModeRealRuntime)),
	}
}

func allowedDegradedEvidence(root string, generic map[string]string) map[string]string {
	allowed := map[string]string{}
	if generic["ebpf_observer"] != "degraded" {
		return allowed
	}
	path := filepath.Join(root, "experiments", "results", "ebpf_smoke", "ebpf_smoke.json")
	data, ok := readJSONMap(path)
	if !ok {
		return allowed
	}
	if stringField(data, "evidence_mode") != string(evidence.ModeDegraded) {
		return allowed
	}
	reason := fallbackOr(stringField(data, "fallback_reason"), "eBPF observer did not attach")
	allowed["ebpf_observer"] = reason
	return allowed
}

func buildKnownLimits(root string, realOnly, generic map[string]string) []string {
	limits := []string{}
	if generic["workspace_probe"] == "degraded" || generic["workspace_isolation"] == "degraded" {
		limits = append(limits, "Portable workspace checks may use degraded-copy fallback when overlayfs is unavailable.")
	}
	limits = append(limits, "Portable E1 benchmark may use degraded pressure fallback; real-pressure-smoke proves real-cgroup-v2 ResourceSampler on openEuler.")
	if realOnly["real_pressure_smoke"] != "passed" {
		limits = append(limits, "Current local final evidence does not prove real-pressure-smoke; run scripts/competition_verify_real.sh on root openEuler.")
	}
	if _, ok := readJSONMap(filepath.Join(root, "experiments", "results", "real_all", "REAL_EVIDENCE_INDEX.json")); !ok {
		limits = append(limits, "real-all evidence is missing.")
	}
	if mode := evidenceModeFromPath(filepath.Join(root, "experiments", "results", "ebpf_smoke", "ebpf_smoke.json"), string(evidence.ModeDegraded)); mode != "real-ebpf" {
		limits = append(limits, "eBPF observer experimental path implemented; current submitted evidence is degraded unless openEuler/Linux smoke reports real-ebpf.")
	}
	return appendUniqueStrings(limits)
}

func evidenceModeFromPath(path, fallback string) string {
	data, ok := readJSONMap(path)
	if !ok {
		return fallback
	}
	return fallbackOr(stringField(data, "evidence_mode"), fallback)
}

func collectGitInfo(root string) FinalGitInfo {
	commit := commandOutput(root, "git", "rev-parse", "HEAD")
	branch := commandOutput(root, "git", "branch", "--show-current")
	dirty := gitDirtyFromPorcelain(commandOutput(root, "git", "status", "--porcelain", "--untracked-files=no", "--", ".", ":(exclude)experiments/results/**"))
	return FinalGitInfo{Commit: fallbackOr(commit, "missing"), Branch: fallbackOr(branch, "missing"), Dirty: dirty}
}

func gitDirtyFromPorcelain(status string) bool {
	for _, line := range strings.Split(status, "\n") {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "??") {
			continue
		}
		if gitDirtyPathIsGeneratedEvidence(line) {
			continue
		}
		return true
	}
	return false
}

func gitDirtyPathIsGeneratedEvidence(line string) bool {
	if len(line) < 4 {
		return false
	}
	path := strings.TrimSpace(line[3:])
	if renamed, _, ok := strings.Cut(path, " -> "); ok {
		path = strings.TrimSpace(renamed)
	}
	return strings.HasPrefix(path, "experiments/results/")
}

func collectSystemInfo() FinalSystemInfo {
	return FinalSystemInfo{
		OSRelease:    fallbackOr(readTextFile("/etc/os-release"), runtime.GOOS),
		Kernel:       fallbackOr(commandOutput("", "uname", "-a"), runtime.GOOS+"/"+runtime.GOARCH),
		CgroupFSType: fallbackOr(commandOutput("", "stat", "-fc", "%T", "/sys/fs/cgroup"), "missing"),
		GoVersion:    runtime.Version(),
	}
}

func commandOutput(dir string, name string, args ...string) string {
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func readTextFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func flattenRequired(groups map[string][]string, extra []string) []string {
	paths := append([]string{}, extra...)
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		paths = append(paths, groups[key]...)
	}
	return appendUnique(paths...)
}

func splitExistingFiles(paths []string) ([]string, []string) {
	generated := []string{}
	missing := []string{}
	for _, path := range appendUnique(paths...) {
		if fileExists(path) {
			generated = append(generated, evidenceDisplayPath(path))
		} else {
			missing = append(missing, evidenceDisplayPath(path))
		}
	}
	return generated, missing
}

func evidenceDisplayPath(path string) string {
	if !filepath.IsAbs(path) {
		return filepath.ToSlash(path)
	}
	rel, err := filepath.Rel(repoRoot(), path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func renderFinalSummary(index FinalEvidenceIndex) string {
	var b strings.Builder
	b.WriteString("# AORT-R Final Evidence Summary\n\n")
	b.WriteString("## Overall conclusion\n")
	if index.RealOnlySummary["all_passed"] {
		b.WriteString("- Real-only openEuler evidence is present and all required real checks passed.\n")
	} else {
		b.WriteString("- Final index was generated from existing evidence; missing or failed real-only checks are listed below.\n")
	}
	b.WriteString(fmt.Sprintf("- Git commit: `%s`\n", index.GitCommit))
	b.WriteString(fmt.Sprintf("- Git branch: `%s`\n", index.GitBranch))
	b.WriteString(fmt.Sprintf("- git_dirty: `%t`\n\n", index.GitDirty))
	writeStatusTable(&b, "generic evidence", index.GenericCompetitionVerify)
	writeStatusTable(&b, "real-only openEuler evidence", index.RealOnlyOpenEuler)
	b.WriteString("## evidence_mode_summary\n")
	writeKeyValueList(&b, index.EvidenceModeSummary)
	b.WriteString("\n## known_limits\n")
	if len(index.KnownLimits) == 0 {
		b.WriteString("- none\n")
	} else {
		for _, item := range index.KnownLimits {
			b.WriteString("- " + item + "\n")
		}
	}
	b.WriteString("\n## allowed_degraded\n")
	if len(index.AllowedDegraded) == 0 {
		b.WriteString("- none\n")
	} else {
		writeKeyValueList(&b, index.AllowedDegraded)
	}
	b.WriteString("\n## Key file paths\n")
	b.WriteString("- `experiments/results/final/FINAL_EVIDENCE_INDEX.json`\n")
	b.WriteString("- `experiments/results/final/FINAL_SUMMARY.md`\n")
	b.WriteString("- `experiments/results/real_all/REAL_EVIDENCE_INDEX.json`\n")
	b.WriteString("- `experiments/results/real_all/REAL_VERIFY_SUMMARY.json`\n")
	b.WriteString("\n## fresh clone verification\n")
	b.WriteString("```bash\n")
	b.WriteString("git clone git@github.com:yxyyyyyyyy/os_complete.git\n")
	b.WriteString("cd os_complete\n")
	b.WriteString("bash scripts/competition_verify_real.sh\n")
	b.WriteString("```\n")
	return b.String()
}

func writeStatusTable(b *strings.Builder, title string, values map[string]string) {
	b.WriteString("## " + title + "\n")
	b.WriteString("| evidence | status |\n")
	b.WriteString("| --- | --- |\n")
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		b.WriteString(fmt.Sprintf("| %s | %s |\n", key, values[key]))
	}
	b.WriteString("\n")
}

func writeKeyValueList(b *strings.Builder, values map[string]string) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		b.WriteString(fmt.Sprintf("- %s: %s\n", key, values[key]))
	}
}

func boolField(data map[string]any, key string) bool {
	value, ok := data[key]
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

func stringField(data map[string]any, key string) string {
	value, ok := data[key]
	if !ok {
		return ""
	}
	if typed, ok := value.(string); ok {
		return typed
	}
	return fmt.Sprint(value)
}

func numberField(data map[string]any, key string) int64 {
	value, ok := data[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case float64:
		return int64(typed)
	case int64:
		return typed
	case int:
		return int64(typed)
	default:
		return 0
	}
}

func appendUnique(values ...string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func appendUniqueStrings(values []string) []string {
	return appendUnique(values...)
}
