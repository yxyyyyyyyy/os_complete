package codebasedag

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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
		SchemaVersion:        SchemaVersion,
		RunID:                runtime.RunID,
		SourceManifest:       session.Manifest,
		Nodes:                nodes,
		Calls:                append([]CallRecord(nil), calls...),
		Patches:              append([]PatchRecord(nil), session.PatchRecords...),
		Tests:                append([]TestRecord(nil), session.Tests...),
		Processes:            append([]ProcessResult(nil), session.Processes...),
		Artifacts:            artifacts,
		AllRequiredPassed:    runtime.AllRequiredPassed,
		HumanFunctionalEdits: 0,
		MinPhysicalLines:     session.MinPhysical,
		MinNonblankLines:     session.MinNonblank,
	}
	if summary.SourceManifest.SchemaVersion == "" {
		summary.SourceManifest.SchemaVersion = SchemaVersion
	}
	return summary, nil
}

func kindForNodeID(nodeID string) NodeKind {
	switch {
	case nodeID == "preflight":
		return KindPreflight
	case nodeID == "planner":
		return KindPlanner
	case nodeID == "integrate":
		return KindIntegrate
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
