package codebasedag

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestLiveNodeExecutorIntegrateSkipsLLM(t *testing.T) {
	dir := t.TempDir()
	store, err := NewRunStore(dir, "live-integrate")
	if err != nil {
		t.Fatal(err)
	}
	exec := NewLiveNodeExecutor(nil, ReviewRemediationTicket(), store)
	result, err := exec.ExecuteNode(context.Background(), NodeExecutionRequest{
		RunID:        "live-integrate",
		NodeID:       "integrate",
		Dependencies: []string{"resource-coder", "context-coder", "evidence-coder"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.OutputSHA256 == "" || result.LLMCallID != "" {
		t.Fatalf("result=%#v", result)
	}
	if _, err := os.Stat(filepath.Join(store.Dir, "outputs", "integrate.json")); err != nil {
		t.Fatal(err)
	}
}

func TestRunLiveExecutesGraphAgainstFakeDeepSeek(t *testing.T) {
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
			nodeID := "planner"
			text := `{"schema_version":"codebase-dag/v1","node_id":"planner","tasks":[],"risks":[],"commands":[]}`
			switch {
			case strings.Contains(prompt, "node_id: resource-coder"):
				nodeID = "resource-coder"
				text = strings.ReplaceAll(validCoderJSON, "resource-coder", "resource-coder")
			case strings.Contains(prompt, "node_id: context-coder"):
				nodeID = "context-coder"
				text = strings.ReplaceAll(validCoderJSON, "resource-coder", "context-coder")
				text = strings.ReplaceAll(text, "internal/review/resource.go", "internal/review/context_sharing.go")
			case strings.Contains(prompt, "node_id: evidence-coder"):
				nodeID = "evidence-coder"
				text = strings.ReplaceAll(validCoderJSON, "resource-coder", "evidence-coder")
				text = strings.ReplaceAll(text, "internal/review/resource.go", "internal/review/review_final.go")
			case strings.Contains(prompt, "role: tester"):
				nodeID = "tester"
				text = `{"schema_version":"codebase-dag/v1","node_id":"tester","verdict":"pass","blocking_findings":[],"non_blocking_findings":[]}`
			case strings.Contains(prompt, "role: reviewer"):
				nodeID = "reviewer"
				text = `{"schema_version":"codebase-dag/v1","node_id":"reviewer","verdict":"pass","blocking_findings":[],"non_blocking_findings":[]}`
			case strings.Contains(prompt, "role: finalizer"):
				nodeID = "finalizer"
				text = `{"schema_version":"codebase-dag/v1","node_id":"finalizer","status":"passed","summary":"ok","limitations":[]}`
			}
			id := callSeq.Add(1)
			payload := map[string]any{
				"id":    "call-" + nodeID + "-" + itoa(id),
				"model": "deepseek-v4-flash",
				"choices": []map[string]any{
					{"message": map[string]string{"content": text}},
				},
				"usage": map[string]int{"prompt_tokens": 3, "completion_tokens": 5, "total_tokens": 8},
			}
			_ = json.NewEncoder(w).Encode(payload)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	repo := newTestGitRepo(t)
	goFile := filepath.Join(repo, "main.go")
	if err := os.WriteFile(goFile, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"git", "init", "--template="},
		{"git", "config", "user.email", "test@example.com"},
		{"git", "config", "user.name", "test"},
		{"git", "add", "main.go"},
		{"git", "commit", "-m", "init"},
	} {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}

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
	})
	if err != nil {
		t.Fatalf("RunLive: %v", err)
	}
	if !result.Summary.AllRequiredPassed {
		t.Fatalf("summary=%#v", result.Summary)
	}
	if len(result.Calls) < 7 {
		t.Fatalf("calls=%d", len(result.Calls))
	}
	if _, err := os.Stat(filepath.Join(result.Dir, "summary.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(result.Dir, "llm_calls.jsonl")); err != nil {
		t.Fatal(err)
	}
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
