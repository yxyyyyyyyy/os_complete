package codebasedag

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BuildEvidenceSummary converts a finished live session into the canonical
// EvidenceSummary schema consumed by ValidateRun.
func BuildEvidenceSummary(store *RunStore, runtime Summary, session *LiveSession, calls []CallRecord) (EvidenceSummary, error) {
	if store == nil {
		return EvidenceSummary{}, fmt.Errorf("run store is required")
	}
	if session == nil {
		return EvidenceSummary{}, fmt.Errorf("live session is required")
	}
	nodes := make([]NodeRecord, 0, len(runtime.Nodes))
	for _, n := range runtime.Nodes {
		rec := NodeRecord{
			SchemaVersion: SchemaVersion,
			NodeID:        n.NodeID,
			Kind:          kindForNodeID(n.NodeID),
			Status:        n.Status,
			OutputSHA256:  n.OutputSHA256,
			LLMCallID:     n.LLMCallID,
			FinishedAt:    n.UpdatedAt,
		}
		if n.Status == NodeFailed {
			rec.Error = n.Reason
		}
		nodes = append(nodes, rec)
	}
	artifacts := map[string]string{}
	if data, err := readArtifactIndex(store); err == nil {
		artifacts = data
	}
	summary := EvidenceSummary{
		SchemaVersion:           SchemaVersion,
		RunID:                   runtime.RunID,
		SourceManifest:          session.Manifest,
		Nodes:                   nodes,
		Calls:                   append([]CallRecord(nil), calls...),
		Patches:                 append([]PatchRecord(nil), session.PatchRecords...),
		Tests:                   append([]TestRecord(nil), session.Tests...),
		Processes:               append([]ProcessResult(nil), session.Processes...),
		PageIDs:                 append([]string(nil), session.PageIDs...),
		CommunicationComparison: session.CommCompare,
		FaultReport:             session.FaultReport,
		BaselineVsAORTR:         session.BaselineCmp,
		JudgeMode:               session.JudgeMode,
		Artifacts:               artifacts,
		AllRequiredPassed:       runtime.AllRequiredPassed,
		HumanFunctionalEdits:    0,
		MinPhysicalLines:        session.MinPhysical,
		MinNonblankLines:        session.MinNonblank,
	}
	if session.CVM != nil && session.CVM.Store != nil {
		m := CVMMetricsFromStats(session.CVM.Store.Stats())
		summary.CVMMetrics = &m
	}
	AttachJudgeEvidence(&summary, summary.CVMMetrics, summary.FaultReport, summary.CommunicationComparison, summary.BaselineVsAORTR)
	workDir := session.WorkloadDir
	if session.Worktree != nil && session.Worktree.WorkDir != "" {
		candidate := session.Worktree.WorkDir
		if _, err := os.Stat(filepath.Join(candidate, "internal", "codebasedag", "resourceagent")); err == nil {
			workDir = candidate
		}
	}
	if workDir != "" {
		if phys, _, err := CountResourceAgentPhysicalLines(workDir); err == nil && phys > 0 {
			summary.ResourceAgentPhysicalLines = phys
		}
	}
	if summary.ResourceAgentPhysicalLines == 0 {
		if phys := resourceAgentLinesFromSeedJudge(store); phys > 0 {
			summary.ResourceAgentPhysicalLines = phys
		}
	}
	if summary.ResourceAgentPhysicalLines == 0 {
		summary.ResourceAgentPhysicalLines = resourceAgentLinesFromManifest(summary.SourceManifest)
	}
	if summary.SourceManifest.SchemaVersion == "" {
		summary.SourceManifest.SchemaVersion = SchemaVersion
	}
	return summary, nil
}

func resourceAgentLinesFromSeedJudge(store *RunStore) int {
	if store == nil {
		return 0
	}
	data, err := os.ReadFile(filepath.Join(store.Dir, "outputs", "seed_judge.json"))
	if err != nil {
		return 0
	}
	var payload struct {
		Physical int `json:"resourceagent_physical"`
	}
	if json.Unmarshal(data, &payload) != nil {
		return 0
	}
	return payload.Physical
}

func resourceAgentLinesFromManifest(manifest SourceManifest) int {
	total := 0
	for _, f := range manifest.Files {
		if strings.Contains(f.Path, "/resourceagent/") || strings.HasPrefix(f.Path, "internal/codebasedag/resourceagent/") {
			if strings.Contains(f.Path, "/_broken/") {
				continue
			}
			total += f.PhysicalLines
		}
	}
	return total
}

func kindForNodeID(nodeID string) NodeKind {
	switch {
	case nodeID == "preflight":
		return KindPreflight
	case nodeID == "planner":
		return KindPlanner
	case nodeID == "integrate":
		return KindIntegrate
	case nodeID == "fault-agent":
		return KindCoder
	case nodeID == "tester" || hasPrefix(nodeID, "tester-recheck-"):
		return KindTester
	case nodeID == "reviewer" || hasPrefix(nodeID, "reviewer-recheck-"):
		return KindReviewer
	case nodeID == "finalizer":
		return KindFinalizer
	case hasPrefix(nodeID, "fixer-"):
		return KindFixer
	case hasSuffix(nodeID, "-coder"):
		return KindCoder
	default:
		return KindCoder
	}
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

func readArtifactIndex(store *RunStore) (map[string]string, error) {
	path := filepath.Join(store.Dir, artifactHashIndex)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out map[string]string
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func VerifyArtifactIntegrity(dir string) error {
	indexPath := filepath.Join(dir, artifactHashIndex)
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return err
	}
	var index map[string]string
	if err := json.Unmarshal(data, &index); err != nil {
		return err
	}
	for rel, want := range index {
		gotSum := sha256Hex(mustRead(filepath.Join(dir, filepath.FromSlash(rel))))
		if gotSum != want {
			return fmt.Errorf("artifact %q hash mismatch", rel)
		}
	}
	return nil
}

func mustRead(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return data
}
