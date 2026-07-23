package codebasedag

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestFixerLoopExhaustionFailsRun(t *testing.T) {
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
		if strings.HasPrefix(nodeID, "reviewer") {
			text = fmt.Sprintf(`{"schema_version":"codebase-dag/v1","node_id":%q,"verdict":"fix","blocking_findings":["needs repair"],"non_blocking_findings":[]}`, nodeID)
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

	_, err := RunLive(context.Background(), LiveRunConfig{
		RunnerConfig: RunnerConfig{
			RunID: "fixer-exhaust", WorkloadDir: repo,
			Provider: RequiredDeepSeekProvider, Model: RequiredDeepSeekModel, MaxCalls: 10,
		},
		OutDir: t.TempDir(), Ticket: "review-remediation",
		MinPhysical: 1, MinNonblank: 1, APIKey: "k", BaseURL: server.URL,
		SkipRace: true, Cleanup: true, TestTimeout: 2 * time.Minute,
	})
	if err == nil {
		t.Fatal("expected failure")
	}
	msg := err.Error()
	if !strings.Contains(msg, "fixer loop exhausted") &&
		!strings.Contains(msg, "review verdict") &&
		!strings.Contains(msg, "budget") &&
		!strings.Contains(msg, "max calls") &&
		!strings.Contains(msg, "call budget") &&
		!strings.Contains(msg, "git apply") &&
		!strings.Contains(msg, "patch") {
		t.Fatalf("unexpected failure: %v", err)
	}
}

func TestWorktreeCleanupRemovesDirectory(t *testing.T) {
	repo := initTinyRepo(t)
	parent := t.TempDir()
	wt, err := CreateWorktree(context.Background(), repo, parent, "cleanup1", "git")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(wt.WorkDir); err != nil {
		t.Fatal(err)
	}
	if err := wt.Cleanup(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(wt.WorkDir); !os.IsNotExist(err) {
		t.Fatalf("worktree directory should be removed, err=%v", err)
	}
}
