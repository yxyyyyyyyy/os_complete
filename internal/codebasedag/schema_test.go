package codebasedag

import (
	"strings"
	"testing"
)

func TestBuildSchemaRepairPromptIncludesErrorAndOriginalContract(t *testing.T) {
	prompt, err := BuildSchemaRepairPrompt(SchemaRepairRequest{
		NodeID:        "planner",
		Role:          KindPlanner,
		DecodeError:   "json: unknown field extra",
		OriginalText:  `{"extra":true}`,
		AllowedFiles:  []string{"internal/review/resource.go"},
		ContextSHA256: "sha256:ctx",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"planner", "json: unknown field extra", `"schema_version"`, "sha256:ctx", "Return exactly one JSON object"} {
		if !strings.Contains(prompt, want) {
			t.Fatalf("repair prompt missing %q:\n%s", want, prompt)
		}
	}
	if strings.Contains(prompt, "DEEPSEEK_API_KEY") {
		t.Fatal("repair prompt should not mention API key names")
	}
}

func TestDecodeRoleOutputDispatchesByRole(t *testing.T) {
	planJSON := `{"schema_version":"codebase-dag/v1","node_id":"planner","tasks":[{"id":"t","owner":"resource-coder","dependencies":[],"files":["a.go"],"acceptance":["go test"]}],"risks":[],"commands":[["go","test","./..."]]}`
	out, err := DecodeRoleOutput(KindPlanner, "planner", []byte(planJSON))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out.(PlanOutput); !ok {
		t.Fatalf("decoded type = %T", out)
	}
	reviewJSON := `{"schema_version":"codebase-dag/v1","node_id":"reviewer","verdict":"pass","blocking_findings":[],"non_blocking_findings":[]}`
	out, err = DecodeRoleOutput(KindReviewer, "reviewer", []byte(reviewJSON))
	if err != nil {
		t.Fatal(err)
	}
	if review, ok := out.(ReviewOutput); !ok || review.Verdict != "pass" {
		t.Fatalf("review output = %#v", out)
	}
}

func TestDecodeRoleOutputRejectsInvalidFinalStatus(t *testing.T) {
	finalJSON := `{"schema_version":"codebase-dag/v1","node_id":"finalizer","status":"maybe","summary":"x","limitations":[]}`
	if _, err := DecodeRoleOutput(KindFinalizer, "finalizer", []byte(finalJSON)); err == nil || !strings.Contains(err.Error(), "status") {
		t.Fatalf("final status error = %v", err)
	}
	if _, err := BuildSchemaRepairPrompt(SchemaRepairRequest{NodeID: "x", Role: NodeKind("bad"), DecodeError: "err"}); err == nil {
		t.Fatal("unknown repair role should fail")
	}
}
