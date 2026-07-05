package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"aort-r/internal/evidence"
)

func TestWorkspaceDegradedCopyCreateRollbackCommitAndDestroy(t *testing.T) {
	manager := NewManager(Config{Root: t.TempDir(), ForceDegraded: true})

	ws, err := manager.Create("coder")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if ws.Mode != ModeDegradedCopy || ws.EvidenceMode != evidence.ModeDegradedCopy {
		t.Fatalf("workspace mode = %#v", ws)
	}
	if ws.FallbackReason == "" {
		t.Fatalf("degraded-copy workspace should explain fallback: %#v", ws)
	}
	if _, err := os.Stat(filepath.Join(ws.MergedDir, "src", "main.txt")); err != nil {
		t.Fatalf("lower fixture not materialized into merged: %v", err)
	}

	if err := os.WriteFile(filepath.Join(ws.MergedDir, "src", "main.txt"), []byte("corrupted\n"), 0o644); err != nil {
		t.Fatalf("write corruption: %v", err)
	}
	if err := manager.Commit("coder"); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ws.OutputDir, "commit_manifest.json")); err != nil {
		t.Fatalf("commit manifest missing: %v", err)
	}

	result, err := manager.Rollback("coder")
	if err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	if !result.RollbackSuccess || !result.BaseIntact || result.EvidenceMode != evidence.ModeDegradedCopy {
		t.Fatalf("rollback result = %#v", result)
	}
	restored, err := os.ReadFile(filepath.Join(ws.MergedDir, "src", "main.txt"))
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(restored) == "corrupted\n" {
		t.Fatalf("rollback did not restore base file")
	}

	if err := manager.Destroy("coder"); err != nil {
		t.Fatalf("Destroy: %v", err)
	}
	if _, err := os.Stat(ws.RuntimeRoot); !os.IsNotExist(err) {
		t.Fatalf("workspace root should be removed, err=%v", err)
	}
}

func TestWorkspaceSafetyBlocksPathAndSymlinkEscapes(t *testing.T) {
	root := t.TempDir()
	manager := NewManager(Config{Root: root, ForceDegraded: true})
	if err := EnsureUnderRoot(root, filepath.Join(root, "agent", "merged")); err != nil {
		t.Fatalf("valid runtime path rejected: %v", err)
	}
	if err := EnsureUnderRoot(root, filepath.Join(root, "..", "outside")); err == nil {
		t.Fatalf("path escape should be rejected")
	}

	ws, err := manager.Create("coder")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(ws.MergedDir, "outside-link")); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if err := manager.Commit("coder"); err == nil {
		t.Fatalf("commit should reject symlink escape instead of following it")
	}
}

func TestWorkspaceRMFaultDemoIsolatesAgentsAndRestoresTarget(t *testing.T) {
	evidence, err := RunRMFaultDemo(Config{Root: t.TempDir(), ForceDegraded: true})
	if err != nil {
		t.Fatalf("RunRMFaultDemo: %v", err)
	}
	if evidence.FaultType != "workspace_rmrf" || evidence.TargetAgent != "coder" {
		t.Fatalf("bad fault identity: %#v", evidence)
	}
	if evidence.EvidenceMode != evidencepkgModeDegradedCopy() || evidence.Mode != ModeDegradedCopy {
		t.Fatalf("bad mode: %#v", evidence)
	}
	if !evidence.LowerDirUnchanged || !evidence.TargetAgentAffected || evidence.CascadeFailure || !evidence.RollbackSuccess {
		t.Fatalf("isolation failed: %#v", evidence)
	}
	if len(evidence.UnaffectedAgents) != 2 {
		t.Fatalf("expected planner/reviewer unaffected: %#v", evidence)
	}
	if !evidence.SafetyChecks.RuntimeRootOnly || !evidence.SafetyChecks.PathEscapeBlocked || !evidence.SafetyChecks.SymlinkEscapeBlocked {
		t.Fatalf("safety checks incomplete: %#v", evidence.SafetyChecks)
	}
}

func evidencepkgModeDegradedCopy() evidence.Mode {
	return evidence.ModeDegradedCopy
}
