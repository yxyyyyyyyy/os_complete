package codebasedag

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func ValidateRun(dir string) error {
	data, err := os.ReadFile(filepath.Join(dir, "summary.json"))
	if err != nil {
		return err
	}
	var summary EvidenceSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		return err
	}
	return ValidateEvidenceSummary(summary)
}

func ValidateEvidenceSummary(summary EvidenceSummary) error {
	if summary.SchemaVersion != SchemaVersion {
		return fmt.Errorf("summary schema %q is not %q", summary.SchemaVersion, SchemaVersion)
	}
	if summary.RunID == "" {
		return fmt.Errorf("run ID is required")
	}
	if !summary.AllRequiredPassed {
		return fmt.Errorf("all_required_passed must be true")
	}
	if summary.HumanFunctionalEdits != 0 {
		return fmt.Errorf("human functional edits must be zero")
	}
	if err := summary.SourceManifest.ValidateLargeCodebaseWithOptions(ManifestOptions{
		MinPhysical: summary.MinPhysicalLines,
		MinNonblank: summary.MinNonblankLines,
	}); err != nil {
		return err
	}
	if summary.SourceManifest.SchemaVersion != "" && summary.SourceManifest.SchemaVersion != SchemaVersion {
		return fmt.Errorf("source manifest schema %q is invalid", summary.SourceManifest.SchemaVersion)
	}
	if summary.SourceManifest.TreeHash == "" {
		return fmt.Errorf("source manifest tree hash is required")
	}
	if summary.SourceManifest.TrackedGoFiles <= 0 {
		return fmt.Errorf("tracked Go files must be positive")
	}
	if err := validateSummaryNodes(summary.Nodes); err != nil {
		return err
	}
	if err := validateSummaryCalls(summary.Calls); err != nil {
		return err
	}
	if err := validateSummaryPatches(summary.Patches); err != nil {
		return err
	}
	if err := validateSummaryTests(summary.Tests); err != nil {
		return err
	}
	if len(summary.Artifacts) == 0 {
		return fmt.Errorf("artifact hash index is required")
	}
	for path, hash := range summary.Artifacts {
		if _, err := cleanPolicyPath(path); err != nil {
			return fmt.Errorf("invalid artifact path %q: %w", path, err)
		}
		if len(hash) != 64 {
			return fmt.Errorf("artifact %q hash must be sha256 hex", path)
		}
	}
	if summary.JudgeMode == "strict" {
		if err := validateStrictJudgeEvidence(summary); err != nil {
			return err
		}
	}
	return nil
}

func validateStrictJudgeEvidence(summary EvidenceSummary) error {
	if len(summary.Processes) == 0 {
		return fmt.Errorf("strict judge mode requires process evidence")
	}
	for _, p := range summary.Processes {
		if p.PID <= 0 {
			return fmt.Errorf("process evidence missing real pid")
		}
	}
	if summary.FaultReport == nil {
		return fmt.Errorf("strict judge mode requires fault_report")
	}
	if summary.CommunicationComparison == nil {
		return fmt.Errorf("strict judge mode requires communication_comparison")
	}
	if summary.CVMMetrics == nil {
		return fmt.Errorf("strict judge mode requires cvm_metrics")
	}
	if summary.ResourceAgentPhysicalLines > 0 && summary.ResourceAgentPhysicalLines < MinResourceAgentPhysicalLines {
		return fmt.Errorf("resource-coder owned corpus physical lines %d < %d", summary.ResourceAgentPhysicalLines, MinResourceAgentPhysicalLines)
	}
	if summary.ResourceAgentPhysicalLines == 0 {
		return fmt.Errorf("strict judge mode requires resourceagent_physical_lines evidence")
	}
	return nil
}

func validateSummaryPatches(patches []PatchRecord) error {
	if len(patches) < 3 {
		return fmt.Errorf("at least 3 attributed coder patches required, got %d", len(patches))
	}
	required := map[string]bool{"resource-coder": false, "context-coder": false, "evidence-coder": false}
	for _, patch := range patches {
		if _, ok := required[patch.NodeID]; ok {
			required[patch.NodeID] = true
		}
		if patch.SourceCallID == "" {
			return fmt.Errorf("patch for %q missing source call ID", patch.NodeID)
		}
		if len(patch.SHA256) != 64 {
			return fmt.Errorf("patch for %q missing sha256 attribution", patch.NodeID)
		}
		if len(patch.ChangedFiles) == 0 {
			return fmt.Errorf("patch for %q has no changed files", patch.NodeID)
		}
		for _, path := range patch.ChangedFiles {
			if _, err := cleanPolicyPath(path); err != nil {
				return fmt.Errorf("patch for %q has invalid path %q: %w", patch.NodeID, path, err)
			}
		}
	}
	for node, ok := range required {
		if !ok {
			return fmt.Errorf("required coder patch %q is missing", node)
		}
	}
	return nil
}

func validateSummaryNodes(nodes []NodeRecord) error {
	required := map[string]struct{}{
		"preflight": {}, "planner": {}, "resource-coder": {}, "context-coder": {}, "evidence-coder": {},
		"fault-agent": {}, "integrate": {}, "tester": {}, "reviewer": {}, "finalizer": {},
	}
	seen := make(map[string]NodeStatus, len(nodes))
	for _, node := range nodes {
		if node.SchemaVersion != "" && node.SchemaVersion != SchemaVersion {
			return fmt.Errorf("node %q schema %q is invalid", node.NodeID, node.SchemaVersion)
		}
		if node.Status != NodeSucceeded {
			return fmt.Errorf("node %q status %q is not succeeded", node.NodeID, node.Status)
		}
		if _, ok := seen[node.NodeID]; ok {
			return fmt.Errorf("duplicate node %q", node.NodeID)
		}
		seen[node.NodeID] = node.Status
	}
	for node := range required {
		if _, ok := seen[node]; !ok {
			return fmt.Errorf("required node %q is missing", node)
		}
	}
	return nil
}

func validateSummaryCalls(calls []CallRecord) error {
	if len(calls) < 7 {
		return fmt.Errorf("at least 7 real DeepSeek calls required, got %d", len(calls))
	}
	if len(calls) > DefaultMaxModelCalls {
		return fmt.Errorf("at most %d calls allowed, got %d", DefaultMaxModelCalls, len(calls))
	}
	ledger := NewCallLedger(RequiredDeepSeekModel, DefaultMaxModelCalls)
	for _, call := range calls {
		if call.SchemaVersion != "" && call.SchemaVersion != SchemaVersion {
			return fmt.Errorf("call %q schema %q is invalid", call.CallID, call.SchemaVersion)
		}
		attemptID, err := ledger.Begin(call.NodeID, call.Role)
		if err != nil {
			return err
		}
		if err := ledger.Finish(attemptID, call); err != nil {
			return err
		}
		if call.Status != "succeeded" {
			return fmt.Errorf("call %q status %q is not succeeded", call.CallID, call.Status)
		}
	}
	return nil
}

func validateSummaryTests(tests []TestRecord) error {
	if len(tests) == 0 {
		return fmt.Errorf("test records are required")
	}
	for _, test := range tests {
		if test.SchemaVersion != "" && test.SchemaVersion != SchemaVersion {
			return fmt.Errorf("test %q schema %q is invalid", test.Name, test.SchemaVersion)
		}
		if test.ExitCode != 0 {
			return fmt.Errorf("test %q exit code %d is not passing", test.Name, test.ExitCode)
		}
	}
	return nil
}
