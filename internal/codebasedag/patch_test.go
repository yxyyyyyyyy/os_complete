package codebasedag

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const validCoderJSON = `{
  "schema_version": "codebase-dag/v1",
  "node_id": "resource-coder",
  "summary": "tighten resource evidence",
  "patch": "diff --git a/internal/review/resource.go b/internal/review/resource.go\n--- a/internal/review/resource.go\n+++ b/internal/review/resource.go\n@@ -1 +1 @@\n-package review\n+package review\n",
  "changed_files": ["internal/review/resource.go"],
  "tests": [["go", "test", "./internal/review"]]
}`

func TestDecodeCoderOutputAndValidatePatchAttribution(t *testing.T) {
	output, err := DecodeCoderOutput("resource-coder", []byte(validCoderJSON))
	if err != nil {
		t.Fatal(err)
	}
	record, err := ValidatePatch(PatchPolicy{
		NodeID:       "resource-coder",
		AllowedFiles: map[string]struct{}{"internal/review/resource.go": {}},
		MaxBytes:     32 << 10,
	}, output, "api-call-1")
	if err != nil {
		t.Fatal(err)
	}
	if record.NodeID != "resource-coder" || record.SourceCallID != "api-call-1" {
		t.Fatalf("record = %#v", record)
	}
	if record.SHA256 == "" || record.Bytes == 0 || len(record.ChangedFiles) != 1 {
		t.Fatalf("record missing attribution: %#v", record)
	}
}

func TestDecodeCoderOutputRejectsFencesTrailingDataUnknownFieldsAndWrongNode(t *testing.T) {
	cases := []struct {
		name string
		body string
		node string
	}{
		{name: "markdown fence", body: "```json\n" + validCoderJSON + "\n```", node: "resource-coder"},
		{name: "trailing", body: validCoderJSON + "\nextra", node: "resource-coder"},
		{name: "unknown field", body: strings.Replace(validCoderJSON, `"tests":`, `"extra":true,"tests":`, 1), node: "resource-coder"},
		{name: "wrong node", body: validCoderJSON, node: "context-coder"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := DecodeCoderOutput(tc.node, []byte(tc.body)); err == nil {
				t.Fatal("decode should fail")
			}
		})
	}
}

func TestSynthesizeQuotedConstPatchApplies(t *testing.T) {
	dir := t.TempDir()
	rel := "internal/review/live_resource_hook.go"
	path := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("package review\n\nconst LiveResourceHook = \"resource-hook-v1\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	init := exec.Command("git", "init", "--template=")
	init.Dir = dir
	if out, err := init.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"git", "config", "user.email", "t@example.com"},
		{"git", "config", "user.name", "t"},
		{"git", "add", "."},
		{"git", "commit", "-m", "i"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}
	patch, name, err := SynthesizeQuotedConstPatch(dir, rel, "resource-hook-v2")
	if err != nil || name != "LiveResourceHook" {
		t.Fatalf("synth: name=%s err=%v", name, err)
	}
	if err := CheckPatchApplies(context.Background(), "git", dir, patch); err != nil {
		t.Fatal(err)
	}
}

func TestValidatePatchRejectsCorruptHunkBody(t *testing.T) {
	output := CoderOutput{
		SchemaVersion: SchemaVersion,
		NodeID:        "resource-coder",
		Summary:       "truncated",
		Patch:         "diff --git a/internal/review/live_resource_hook.go b/internal/review/live_resource_hook.go\n--- a/internal/review/live_resource_hook.go\n+++ b/internal/review/live_resource_hook.go\n@@ -1,2 +1,2 @@\n package review\nbad-line-without-prefix\n",
		ChangedFiles:  []string{"internal/review/live_resource_hook.go"},
	}
	_, err := ValidatePatch(PatchPolicy{
		NodeID:       "resource-coder",
		AllowedFiles: map[string]struct{}{"internal/review/live_resource_hook.go": {}},
		MaxBytes:     32 << 10,
	}, output, "call")
	if err == nil || !strings.Contains(err.Error(), "corrupt hunk body") {
		t.Fatalf("error = %v", err)
	}
}

func TestValidatePatchNormalizesMiscountedHunkHeader(t *testing.T) {
	// DeepSeek often emits @@ -3,7 +3,7 @@ while the body only has 6 lines.
	output := CoderOutput{
		SchemaVersion: SchemaVersion,
		NodeID:        "evidence-coder",
		Summary:       "restore evidence marker",
		Patch: "diff --git a/internal/codebasedag/judge_evidence.go b/internal/codebasedag/judge_evidence.go\n" +
			"--- a/internal/codebasedag/judge_evidence.go\n" +
			"+++ b/internal/codebasedag/judge_evidence.go\n" +
			"@@ -3,7 +3,7 @@ package codebasedag\n" +
			" import \"aort-r/internal/cvm\"\n" +
			" \n" +
			" // EvidenceJudgeMarker is flipped by live DAG agents from seed-incomplete to complete.\n" +
			"-const EvidenceJudgeMarker = \"seed-incomplete\"\n" +
			"+const EvidenceJudgeMarker = \"judge-evidence-complete\"\n" +
			" \n" +
			" // CVMMetrics is the evidence-facing projection of cvm.Stats.\n",
		ChangedFiles: []string{"internal/codebasedag/judge_evidence.go"},
	}
	rec, err := ValidatePatch(PatchPolicy{
		NodeID:       "evidence-coder",
		AllowedFiles: map[string]struct{}{"internal/codebasedag/judge_evidence.go": {}},
		MaxBytes:     32 << 10,
	}, output, "call")
	if err != nil {
		t.Fatalf("ValidatePatch: %v", err)
	}
	if rec.NodeID != "evidence-coder" || rec.Bytes == 0 {
		t.Fatalf("unexpected record: %+v", rec)
	}
}

func TestValidatePatchRejectsPolicyViolations(t *testing.T) {
	output, err := DecodeCoderOutput("resource-coder", []byte(validCoderJSON))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name   string
		mutate func(CoderOutput) CoderOutput
		policy PatchPolicy
		want   string
	}{
		{
			name:   "outside allowlist",
			mutate: func(o CoderOutput) CoderOutput { return o },
			policy: PatchPolicy{NodeID: "resource-coder", AllowedFiles: map[string]struct{}{"other.go": {}}, MaxBytes: 32 << 10},
			want:   "outside allowlist",
		},
		{
			name: "model declaration ignored in favor of patch files",
			mutate: func(o CoderOutput) CoderOutput {
				o.ChangedFiles = []string{"internal/review/other.go", "internal/review/resource.go"}
				return o
			},
			policy: PatchPolicy{NodeID: "resource-coder", AllowedFiles: map[string]struct{}{"internal/review/resource.go": {}}, MaxBytes: 32 << 10},
			want:   "",
		},
		{
			name: "immutable deletion",
			mutate: func(o CoderOutput) CoderOutput {
				o.Patch = "diff --git a/internal/review/resource.go b/internal/review/resource.go\n--- a/internal/review/resource.go\n+++ /dev/null\n@@ -1 +0,0 @@\n-package review\n"
				return o
			},
			policy: PatchPolicy{
				NodeID:         "resource-coder",
				AllowedFiles:   map[string]struct{}{"internal/review/resource.go": {}},
				ImmutableFiles: map[string]string{"internal/review/resource.go": "test"},
				MaxBytes:       32 << 10,
			},
			want: "immutable",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec, err := ValidatePatch(tc.policy, tc.mutate(output), "api-call-1")
			if tc.want == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if len(rec.ChangedFiles) != 1 || rec.ChangedFiles[0] != "internal/review/resource.go" {
					t.Fatalf("expected patch-derived files, got %#v", rec.ChangedFiles)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}

func TestDecodeCoderOutputAllowsEmptySummaryWithReplacement(t *testing.T) {
	body := `{"schema_version":"codebase-dag/v1","node_id":"context-coder","summary":"","replacement_value":"hook-v2","changed_files":[],"tests":[["go","test","./internal/..."]]}`
	out, err := DecodeCoderOutput("context-coder", []byte(body))
	if err != nil {
		t.Fatalf("DecodeCoderOutput: %v", err)
	}
	if out.ReplacementValue != "hook-v2" || out.Summary != "" || len(out.ChangedFiles) != 0 {
		t.Fatalf("unexpected output: %#v", out)
	}
}

func TestDecodeCoderOutputStillRequiresChangedFilesWithoutReplacement(t *testing.T) {
	body := `{"schema_version":"codebase-dag/v1","node_id":"context-coder","summary":"x","patch":"diff --git a/a b/a\n","changed_files":[],"tests":[]}`
	if _, err := DecodeCoderOutput("context-coder", []byte(body)); err == nil || !strings.Contains(err.Error(), "changed_files") {
		t.Fatalf("error = %v, want changed_files", err)
	}
}
