package review

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteReviewFinalIndexesScenariosWithoutMutatingLegacyFinal(t *testing.T) {
	root := t.TempDir()
	resourceDir := filepath.Join(root, "resource")
	contextDir := filepath.Join(root, "context")
	demoDir := filepath.Join(root, "demo")
	legacyDir := filepath.Join(root, "legacy")
	for _, item := range []struct {
		dir      string
		scenario string
		status   string
	}{
		{resourceDir, "resource-isolation", "passed"},
		{contextDir, "context-sharing", "passed"},
		{demoDir, "real-agent-demo", "passed"},
	} {
		if err := os.MkdirAll(item.dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := writeJSONFile(filepath.Join(item.dir, "summary.json"), map[string]any{"schema_version": SchemaVersion, "scenario_id": item.scenario, "status": item.status, "evidence_mode": "degraded", "per_run": []map[string]any{{"success": true}}}); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	legacyPath := filepath.Join(legacyDir, "FINAL_EVIDENCE_INDEX.json")
	legacy := []byte("{\"legacy\":true}\n")
	if err := os.WriteFile(legacyPath, legacy, 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(root, "review-final")
	index, err := WriteReviewFinal(ReviewFinalConfig{OutDir: out, ResourceDir: resourceDir, ContextDir: contextDir, DemoDir: demoDir, LegacyFinalDir: legacyDir})
	if err != nil {
		t.Fatal(err)
	}
	if !index.AllRequiredPassed || len(index.Scenarios) != 3 || index.LegacyFinal.Status != "present" {
		t.Fatalf("index = %+v", index)
	}
	if data, err := os.ReadFile(legacyPath); err != nil || string(data) != string(legacy) {
		t.Fatalf("legacy final was mutated: %q, %v", data, err)
	}
	for _, name := range []string{"REVIEW_EVIDENCE_INDEX.json", "REVIEW_SUMMARY.md"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
}

func TestWriteReviewFinalWritesFailureIndexWhenScenarioMissing(t *testing.T) {
	out := t.TempDir()
	index, err := WriteReviewFinal(ReviewFinalConfig{OutDir: out, ResourceDir: filepath.Join(out, "missing-resource"), ContextDir: filepath.Join(out, "missing-context"), DemoDir: filepath.Join(out, "missing-demo"), LegacyFinalDir: filepath.Join(out, "missing-legacy")})
	if err == nil || index.AllRequiredPassed || len(index.MissingFiles) != 3 {
		t.Fatalf("missing index = %+v, err=%v", index, err)
	}
	if _, statErr := os.Stat(filepath.Join(out, "REVIEW_EVIDENCE_INDEX.json")); statErr != nil {
		t.Fatalf("failure index missing: %v", statErr)
	}
}
