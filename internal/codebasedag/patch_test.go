package codebasedag

import (
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
			name: "declared mismatch",
			mutate: func(o CoderOutput) CoderOutput {
				o.ChangedFiles = []string{"internal/review/other.go"}
				return o
			},
			policy: PatchPolicy{NodeID: "resource-coder", AllowedFiles: map[string]struct{}{"internal/review/resource.go": {}}, MaxBytes: 32 << 10},
			want:   "declared",
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
			_, err := ValidatePatch(tc.policy, tc.mutate(output), "api-call-1")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
		})
	}
}
