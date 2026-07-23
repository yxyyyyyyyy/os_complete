package codebasedag

import (
	"strings"
	"testing"
)

func TestPlanPatchApplicationOrdersPatchesAndDetectsCollisions(t *testing.T) {
	records := []PatchRecord{
		{NodeID: "resource-coder", ChangedFiles: []string{"internal/review/resource.go"}, SHA256: strings.Repeat("a", 64)},
		{NodeID: "context-coder", ChangedFiles: []string{"internal/review/context.go"}, SHA256: strings.Repeat("b", 64)},
		{NodeID: "evidence-coder", ChangedFiles: []string{"internal/review/resource.go"}, SHA256: strings.Repeat("c", 64)},
	}
	plan := PlanPatchApplication(records)
	if len(plan.Ordered) != 3 || plan.Ordered[0].NodeID != "context-coder" || plan.Ordered[2].NodeID != "resource-coder" {
		t.Fatalf("ordered = %#v", plan.Ordered)
	}
	if len(plan.Collisions) != 1 || plan.Collisions[0].Path != "internal/review/resource.go" {
		t.Fatalf("collisions = %#v", plan.Collisions)
	}
	if !plan.RequiresFixer {
		t.Fatal("collision should require fixer")
	}
}

func TestPatchBundleManifestRejectsMissingHashesAndUnsafePaths(t *testing.T) {
	_, err := NewPatchBundleManifest([]PatchRecord{{NodeID: "coder", ChangedFiles: []string{"../escape.go"}, SHA256: strings.Repeat("a", 64)}})
	if err == nil || !strings.Contains(err.Error(), "invalid path") {
		t.Fatalf("unsafe path error = %v", err)
	}
	_, err = NewPatchBundleManifest([]PatchRecord{{NodeID: "coder", ChangedFiles: []string{"safe.go"}}})
	if err == nil || !strings.Contains(err.Error(), "hash") {
		t.Fatalf("missing hash error = %v", err)
	}
	manifest, err := NewPatchBundleManifest([]PatchRecord{{NodeID: "coder", ChangedFiles: []string{"safe.go"}, SHA256: strings.Repeat("a", 64)}})
	if err != nil {
		t.Fatal(err)
	}
	if manifest.BundleSHA256 == "" || manifest.Patches[0].NodeID != "coder" {
		t.Fatalf("manifest = %#v", manifest)
	}
}
