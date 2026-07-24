package codebasedag

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"aort-r/internal/llm"
)

func TestLiveNodeExecutorIntegrateRequiresPatches(t *testing.T) {
	dir := t.TempDir()
	store, err := NewRunStore(dir, "live-integrate")
	if err != nil {
		t.Fatal(err)
	}
	exec := NewLiveNodeExecutor(nil, ReviewRemediationTicket(), store)
	_, err = exec.ExecuteNode(context.Background(), NodeExecutionRequest{
		RunID:        "live-integrate",
		NodeID:       "integrate",
		Dependencies: []string{"resource-coder", "context-coder", "evidence-coder"},
	})
	if err == nil || !strings.Contains(err.Error(), "at least 3 coder patches") {
		t.Fatalf("expected patch requirement error, got %v", err)
	}
}

func TestLiveNodeExecutorRejectsInvalidCoderJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			_, _ = w.Write([]byte(`{"data":[{"id":"deepseek-v4-flash"}]}`))
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "call-bad", "model": "deepseek-v4-flash",
			"choices": []map[string]any{{"message": map[string]string{"content": `{"schema_version":"codebase-dag/v1","node_id":"resource-coder"}`}}},
			"usage":   map[string]int{"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2},
		})
	}))
	defer server.Close()

	dir := t.TempDir()
	store, err := NewRunStore(dir, "live-bad-json")
	if err != nil {
		t.Fatal(err)
	}
	provider := llm.NewDeepSeekProvider(llm.DeepSeekConfig{APIKey: "k", BaseURL: server.URL, Model: RequiredDeepSeekModel})
	model, err := NewStrictModel(provider, StrictModelOptions{RequiredModel: RequiredDeepSeekModel, MaxCalls: 10})
	if err != nil {
		t.Fatal(err)
	}
	exec := NewLiveNodeExecutor(model, ReviewRemediationTicket(), store)
	_, err = exec.ExecuteNode(context.Background(), NodeExecutionRequest{RunID: "live-bad-json", NodeID: "resource-coder"})
	if err == nil {
		t.Fatal("expected decode/validate failure")
	}
}

func TestRunLiveExecutesGraphAgainstFakeDeepSeek(t *testing.T) {
	repo := initReviewFixtureRepo(t)
	var callSeq atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/models" && r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"data":[{"id":"deepseek-v4-flash"}]}`))
		case r.URL.Path == "/chat/completions":
			var body struct {
				Messages []struct {
					Content string `json:"content"`
				} `json:"messages"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			prompt := ""
			if len(body.Messages) > 0 {
				prompt = body.Messages[0].Content
			}
			text := fakeLiveResponse(prompt)
			id := callSeq.Add(1)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":    fmt.Sprintf("call-%d", id),
				"model": "deepseek-v4-flash",
				"choices": []map[string]any{
					{"message": map[string]string{"content": text}},
				},
				"usage": map[string]int{"prompt_tokens": 3, "completion_tokens": 5, "total_tokens": 8},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	out := t.TempDir()
	result, err := RunLive(context.Background(), LiveRunConfig{
		RunnerConfig: RunnerConfig{
			RunID:       "live-fake",
			WorkloadDir: repo,
			Provider:    RequiredDeepSeekProvider,
			Model:       RequiredDeepSeekModel,
			MaxCalls:    10,
		},
		OutDir:      out,
		Ticket:      "review-remediation",
		MinPhysical: 1,
		MinNonblank: 1,
		APIKey:      "test-key",
		BaseURL:     server.URL,
		RequireKey:  true,
		SkipRace:    true,
		Cleanup:     true,
		TestTimeout: 2 * time.Minute,
	})
	if err != nil {
		t.Fatalf("RunLive: %v", err)
	}
	if !result.AllRequiredPassed {
		t.Fatalf("expected passed, summary=%#v", result.Summary)
	}
	if len(result.Calls) < 7 {
		t.Fatalf("calls=%d", len(result.Calls))
	}
	if err := ValidateRun(result.Dir); err != nil {
		t.Fatalf("ValidateRun: %v", err)
	}
}

func fakeLiveResponse(prompt string) string {
	nodeID := "planner"
	for _, line := range strings.Split(prompt, "\n") {
		if strings.HasPrefix(line, "node_id: ") {
			nodeID = strings.TrimSpace(strings.TrimPrefix(line, "node_id: "))
			break
		}
	}
	switch {
	case nodeID == "planner" || strings.Contains(prompt, "role: planner"):
		return `{"schema_version":"codebase-dag/v1","node_id":"planner","tasks":[{"id":"t1","owner":"resource-coder","dependencies":[],"files":["internal/codebasedag/judge_resource.go"],"acceptance":["go test"]}],"risks":[],"commands":[["go","test","./"]]}`
	case strings.HasPrefix(nodeID, "fixer-"):
		// Markers are already restored by resource-coder; fixer must still emit an applying diff.
		patch := `diff --git a/internal/codebasedag/judge_resource.go b/internal/codebasedag/judge_resource.go
--- a/internal/codebasedag/judge_resource.go
+++ b/internal/codebasedag/judge_resource.go
@@ -1,3 +1,4 @@
 package codebasedag
 
 const ResourceJudgeMarker = "judge-resource-complete"
+// fixer-restore-touch
`
		return coderPatchJSON(nodeID, patch, []string{"internal/codebasedag/judge_resource.go"})
	case nodeID == "resource-coder":
		patch, files, _ := CompleteJudgeMarkersPatch(nodeID)
		return coderPatchJSON(nodeID, patch, files)
	case nodeID == "context-coder":
		patch, files, _ := CompleteJudgeMarkersPatch("context-coder")
		return coderPatchJSON(nodeID, patch, files)
	case nodeID == "evidence-coder":
		patch, files, _ := CompleteJudgeMarkersPatch("evidence-coder")
		return coderPatchJSON(nodeID, patch, files)
	case strings.HasPrefix(nodeID, "tester"):
		return fmt.Sprintf(`{"schema_version":"codebase-dag/v1","node_id":%q,"verdict":"pass","blocking_findings":[],"non_blocking_findings":[]}`, nodeID)
	case strings.HasPrefix(nodeID, "reviewer"):
		return fmt.Sprintf(`{"schema_version":"codebase-dag/v1","node_id":%q,"verdict":"pass","blocking_findings":[],"non_blocking_findings":[]}`, nodeID)
	case nodeID == "finalizer":
		return `{"schema_version":"codebase-dag/v1","node_id":"finalizer","status":"passed","summary":"ok","limitations":[]}`
	default:
		return `{"schema_version":"codebase-dag/v1","node_id":"planner","tasks":[{"id":"t1","owner":"resource-coder","dependencies":[],"files":[],"acceptance":[]}],"risks":[],"commands":[["go","test","./"]]}`
	}
}

func coderPatchJSON(node, patch string, files []string) string {
	encPatch, _ := json.Marshal(patch)
	encFiles, _ := json.Marshal(files)
	return fmt.Sprintf(`{"schema_version":"codebase-dag/v1","node_id":%q,"summary":"restore judge marker","patch":%s,"changed_files":%s,"tests":[["go","test","./"]]}`, node, encPatch, encFiles)
}

func initReviewFixtureRepo(t *testing.T) string {
	t.Helper()
	dir := newTestGitRepo(t)
	for _, args := range [][]string{
		{"git", "init"},
		{"git", "config", "user.email", "test@example.com"},
		{"git", "config", "user.name", "test"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}
	files := map[string]string{
		"go.mod":                                                 "module example.com/reviewfix\n\ngo 1.22\n",
		"internal/codebasedag/judge_resource.go":                 "package codebasedag\n\nconst ResourceJudgeMarker = \"seed-incomplete\"\n",
		"internal/codebasedag/judge_context.go":                  "package codebasedag\n\nconst ContextJudgeMarker = \"seed-incomplete\"\n",
		"internal/codebasedag/judge_evidence.go":                 "package codebasedag\n\nconst EvidenceJudgeMarker = \"seed-incomplete\"\n",
		"internal/codebasedag/judge_smoke_test.go":               "package codebasedag\n\nimport \"testing\"\n\nfunc TestJudgeMarkersPresent(t *testing.T) {\n\tif ResourceJudgeMarker == \"\" || ContextJudgeMarker == \"\" || EvidenceJudgeMarker == \"\" {\n\t\tt.Fatal(\"empty\")\n\t}\n}\n",
		"internal/codebasedag/acceptance/context_real.sh":        "#!/bin/sh\nexit 2\n",
		"internal/codebasedag/acceptance/resource_real.sh":       "#!/bin/sh\nexit 2\n",
		"internal/codebasedag/acceptance/review_final_strict.sh": "#!/bin/sh\nexit 2\n",
	}
	for rel, body := range files {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{
		{"git", "add", "."},
		{"git", "commit", "-m", "init"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}
	return dir
}
