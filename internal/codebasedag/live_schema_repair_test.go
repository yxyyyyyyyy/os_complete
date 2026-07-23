package codebasedag

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSchemaRepairRecoversInvalidCoderJSON(t *testing.T) {
	repo := initReviewFixtureRepo(t)
	var callSeq atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/models" {
			_, _ = w.Write([]byte(`{"data":[{"id":"deepseek-v4-flash"}]}`))
			return
		}
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
		for _, line := range strings.Split(prompt, "\n") {
			if strings.HasPrefix(line, "node_id: ") {
				nodeID = strings.TrimSpace(strings.TrimPrefix(line, "node_id: "))
				break
			}
		}
		text := fakeLiveResponse(prompt)
		if nodeID == "evidence-coder" && !strings.Contains(prompt, "decode_error:") {
			// Invalid escape that previously killed the Huawei live run.
			text = `{"schema_version":"codebase-dag/v1","node_id":"evidence-coder","summary":"bad","patch":"diff --git a/x b/x\n--- a/x\n+++ b/x\n@@ -1 +1 @@\n-\)\n+ok\n","changed_files":["internal/review/live_evidence_hook.go"],"tests":[["go","test","./internal/review/"]]}`
		}
		id := callSeq.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":    fmt.Sprintf("call-%d", id),
			"model": "deepseek-v4-flash",
			"choices": []map[string]any{
				{"message": map[string]string{"content": text}},
			},
			"usage": map[string]int{"prompt_tokens": 3, "completion_tokens": 5, "total_tokens": 8},
		})
	}))
	defer server.Close()

	result, err := RunLive(context.Background(), LiveRunConfig{
		RunnerConfig: RunnerConfig{
			RunID: "schema-repair-run", WorkloadDir: repo,
			Provider: RequiredDeepSeekProvider, Model: RequiredDeepSeekModel, MaxCalls: 10,
		},
		OutDir: t.TempDir(), Ticket: "review-remediation",
		MinPhysical: 1, MinNonblank: 1, APIKey: "k", BaseURL: server.URL,
		SkipRace: true, Cleanup: true, TestTimeout: 2 * time.Minute,
	})
	if err != nil {
		t.Fatalf("RunLive: %v", err)
	}
	if callSeq.Load() < 8 {
		t.Fatalf("expected schema-repair extra call, got %d", callSeq.Load())
	}
	rawPath := filepath.Join(result.Dir, "outputs", "evidence-coder.raw.txt")
	if _, err := os.Stat(rawPath); err != nil {
		t.Fatalf("missing raw failed output: %v", err)
	}
	if !result.AllRequiredPassed {
		t.Fatal("expected repaired run to pass")
	}
}
