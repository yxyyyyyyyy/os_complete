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
	"sync/atomic"
	"testing"
	"time"
)

func TestE2ERunLiveDirValidatesWithAortctl(t *testing.T) {
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
		id := callSeq.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": fmt.Sprintf("call-%d", id), "model": "deepseek-v4-flash",
			"choices": []map[string]any{{"message": map[string]string{"content": fakeLiveResponse(prompt)}}},
			"usage":   map[string]int{"prompt_tokens": 3, "completion_tokens": 5, "total_tokens": 8},
		})
	}))
	defer server.Close()

	out := filepath.Join(os.TempDir(), "aort-e2e-live-out")
	_ = os.RemoveAll(out)
	result, err := RunLive(context.Background(), LiveRunConfig{
		RunnerConfig: RunnerConfig{RunID: "e2e-cli", WorkloadDir: repo, Provider: RequiredDeepSeekProvider, Model: RequiredDeepSeekModel, MaxCalls: 10},
		OutDir:       out, Ticket: "review-remediation", MinPhysical: 1, MinNonblank: 1,
		APIKey: "k", BaseURL: server.URL, SkipRace: true, Cleanup: true, TestTimeout: 2 * time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(t.TempDir(), "aortctl")
	build := exec.Command("go", "build", "-o", bin, "./cmd/aortctl")
	build.Dir = filepath.Join("..", "..")
	if outb, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build aortctl: %v\n%s", err, outb)
	}
	cmd := exec.Command(bin, "evidence", "codebase-dag", "--run", result.Dir)
	if outb, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("aortctl evidence: %v\n%s", err, outb)
	}
}
