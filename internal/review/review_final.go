package review

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type ReviewFinalConfig struct {
	OutDir         string
	ResourceDir    string
	ContextDir     string
	DemoDir        string
	LegacyFinalDir string
}

type ReviewArtifactStatus struct {
	ScenarioID    string `json:"scenario_id"`
	Path          string `json:"path"`
	Status        string `json:"status"`
	EvidenceMode  string `json:"evidence_mode"`
	Present       bool   `json:"present"`
	Valid         bool   `json:"valid"`
	FailureReason string `json:"failure_reason,omitempty"`
}

type LegacyFinalStatus struct {
	Path    string `json:"path"`
	Status  string `json:"status"`
	Present bool   `json:"present"`
}

type ReviewEvidenceIndex struct {
	SchemaVersion     string                          `json:"schema_version"`
	Timestamp         string                          `json:"timestamp"`
	GitCommit         string                          `json:"git_commit"`
	GitDirty          bool                            `json:"git_dirty"`
	Scenarios         map[string]ReviewArtifactStatus `json:"scenarios"`
	LegacyFinal       LegacyFinalStatus               `json:"legacy_final"`
	GeneratedFiles    []string                        `json:"generated_files"`
	MissingFiles      []string                        `json:"missing_files"`
	FailedScenarios   []string                        `json:"failed_scenarios"`
	AllRequiredPassed bool                            `json:"all_required_passed"`
	Limitations       []string                        `json:"limitations"`
}

func WriteReviewFinal(cfg ReviewFinalConfig) (ReviewEvidenceIndex, error) {
	if cfg.OutDir == "" {
		cfg.OutDir = filepath.Join("experiments", "results", "review_final")
	}
	if cfg.ResourceDir == "" {
		cfg.ResourceDir = filepath.Join("experiments", "results", "review_remediation", "resource_isolation")
	}
	if cfg.ContextDir == "" {
		cfg.ContextDir = filepath.Join("experiments", "results", "review_remediation", "context_sharing")
	}
	if cfg.DemoDir == "" {
		cfg.DemoDir = filepath.Join("experiments", "results", "review_remediation", "real_agent_demo")
	}
	if cfg.LegacyFinalDir == "" {
		cfg.LegacyFinalDir = filepath.Join("experiments", "results", "final")
	}
	if err := os.MkdirAll(cfg.OutDir, 0o755); err != nil {
		return ReviewEvidenceIndex{}, err
	}
	indexPath := filepath.Join(cfg.OutDir, "REVIEW_EVIDENCE_INDEX.json")
	summaryPath := filepath.Join(cfg.OutDir, "REVIEW_SUMMARY.md")
	index := ReviewEvidenceIndex{
		SchemaVersion:   SchemaVersion,
		Timestamp:       time.Now().UTC().Format(time.RFC3339Nano),
		GitCommit:       gitCommit(),
		GitDirty:        gitDirty(),
		Scenarios:       make(map[string]ReviewArtifactStatus),
		GeneratedFiles:  []string{indexPath, summaryPath},
		MissingFiles:    []string{},
		FailedScenarios: []string{},
		Limitations: []string{
			"This index references historical final evidence without modifying it.",
			"Degraded evidence remains valid only when its fallback reason and platform boundary are explicit in the source summary.",
		},
	}
	inputs := []struct {
		key      string
		scenario string
		dir      string
	}{
		{"resource_isolation", "resource-isolation", cfg.ResourceDir},
		{"context_sharing", "context-sharing", cfg.ContextDir},
		{"real_agent_demo", "real-agent-demo", cfg.DemoDir},
	}
	for _, input := range inputs {
		path := filepath.Join(input.dir, "summary.json")
		status := inspectReviewSummary(path, input.scenario)
		index.Scenarios[input.key] = status
		if !status.Present {
			index.MissingFiles = append(index.MissingFiles, path)
		}
		if !status.Valid {
			index.FailedScenarios = append(index.FailedScenarios, input.key)
		}
	}
	legacyPath := filepath.Join(cfg.LegacyFinalDir, "FINAL_EVIDENCE_INDEX.json")
	index.LegacyFinal = LegacyFinalStatus{Path: legacyPath, Status: "missing"}
	if _, err := os.Stat(legacyPath); err == nil {
		index.LegacyFinal.Status = "present"
		index.LegacyFinal.Present = true
	}
	sort.Strings(index.MissingFiles)
	sort.Strings(index.FailedScenarios)
	index.AllRequiredPassed = len(index.MissingFiles) == 0 && len(index.FailedScenarios) == 0
	if err := writeJSONFile(indexPath, index); err != nil {
		return index, err
	}
	if err := os.WriteFile(summaryPath, []byte(renderReviewSummary(index)), 0o644); err != nil {
		return index, err
	}
	if !index.AllRequiredPassed {
		return index, fmt.Errorf("review-final failed: missing=%v failed=%v", index.MissingFiles, index.FailedScenarios)
	}
	return index, nil
}

func inspectReviewSummary(path, expectedScenario string) ReviewArtifactStatus {
	status := ReviewArtifactStatus{ScenarioID: expectedScenario, Path: path, Status: "missing"}
	data, err := os.ReadFile(path)
	if err != nil {
		status.FailureReason = err.Error()
		return status
	}
	status.Present = true
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		status.Status = "invalid"
		status.FailureReason = err.Error()
		return status
	}
	actualScenario, _ := decoded["scenario_id"].(string)
	if actualScenario != expectedScenario {
		status.Status = "invalid"
		status.FailureReason = fmt.Sprintf("scenario_id=%q, want %q", actualScenario, expectedScenario)
		return status
	}
	status.EvidenceMode, _ = decoded["evidence_mode"].(string)
	if explicit, ok := decoded["status"].(string); ok && explicit != "" {
		status.Status = explicit
		status.Valid = explicit == "passed"
		if !status.Valid {
			status.FailureReason, _ = decoded["failure_reason"].(string)
		}
		return status
	}
	perRun, ok := decoded["per_run"].([]any)
	if !ok || len(perRun) == 0 {
		status.Status = "invalid"
		status.FailureReason = "per_run is missing or empty"
		return status
	}
	allPassed := true
	for _, item := range perRun {
		run, ok := item.(map[string]any)
		if !ok {
			allPassed = false
			break
		}
		passed, ok := run["success"].(bool)
		if !ok || !passed {
			allPassed = false
			break
		}
	}
	if allPassed {
		status.Status = "passed"
		status.Valid = true
	} else {
		status.Status = "failed"
		status.FailureReason = "one or more measured runs failed"
	}
	return status
}

func renderReviewSummary(index ReviewEvidenceIndex) string {
	var b strings.Builder
	b.WriteString("# AORT-R Review Evidence Summary\n\n")
	fmt.Fprintf(&b, "- schema_version: `%s`\n- git_commit: `%s`\n- git_dirty: `%t`\n- all_required_passed: `%t`\n\n", index.SchemaVersion, index.GitCommit, index.GitDirty, index.AllRequiredPassed)
	b.WriteString("| scenario | status | evidence_mode | source |\n|---|---|---|---|\n")
	keys := make([]string, 0, len(index.Scenarios))
	for key := range index.Scenarios {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		item := index.Scenarios[key]
		fmt.Fprintf(&b, "| %s | %s | %s | `%s` |\n", item.ScenarioID, item.Status, item.EvidenceMode, item.Path)
	}
	fmt.Fprintf(&b, "\nLegacy FINAL_EVIDENCE_INDEX: `%s` (%s). It is referenced read-only.\n", index.LegacyFinal.Path, index.LegacyFinal.Status)
	if len(index.MissingFiles) > 0 {
		b.WriteString("\n## Missing files\n\n")
		for _, path := range index.MissingFiles {
			fmt.Fprintf(&b, "- `%s`\n", path)
		}
	}
	return b.String()
}
