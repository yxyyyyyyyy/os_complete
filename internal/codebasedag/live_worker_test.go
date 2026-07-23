package codebasedag

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

type recordingProcessRuntime struct {
	last ProcessConfig
	pid  int
}

func (r *recordingProcessRuntime) StartPrepared(_ context.Context, cfg ProcessConfig) (ProcessResult, error) {
	r.last = cfg
	if r.pid == 0 {
		r.pid = 4242
	}
	return ProcessResult{
		PID:                 r.pid,
		CgroupPath:          "/sys/fs/cgroup/aort.slice/codebase-dag/node-4242",
		ProcessEvidenceMode: "test-process",
		CgroupEvidenceMode:  "test-cgroup",
		EvidenceMode:        "test-process",
		ExitCode:            0,
		OutputSHA256:        "abc",
		Metrics:             map[string]string{"memory.current": "1024", "pids.current": "1"},
	}, nil
}

func TestRunLiveExecutesWorkerCommandAndRecordsPID(t *testing.T) {
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
	}))
	defer server.Close()

	recorder := &recordingProcessRuntime{pid: 7777}
	out := t.TempDir()
	result, err := RunLive(context.Background(), LiveRunConfig{
		RunnerConfig: RunnerConfig{
			RunID: "worker-pid-run", WorkloadDir: repo,
			Provider: RequiredDeepSeekProvider, Model: RequiredDeepSeekModel, MaxCalls: 10,
		},
		OutDir: out, Ticket: "review-remediation",
		MinPhysical: 1, MinNonblank: 1, APIKey: "k", BaseURL: server.URL,
		SkipRace: true, Cleanup: true, TestTimeout: 2 * time.Minute,
		WorkerCommand: "/usr/bin/true",
		ProcessRT:     recorder,
	})
	if err != nil {
		t.Fatalf("RunLive: %v", err)
	}
	if recorder.last.Worker.Command != "/usr/bin/true" {
		t.Fatalf("worker command not executed via ProcessRT: %#v", recorder.last)
	}
	if recorder.last.NodeID != "integrate-worker" {
		t.Fatalf("unexpected node: %q", recorder.last.NodeID)
	}
	procPath := filepath.Join(result.Dir, "processes", "integrate-worker.json")
	raw, err := os.ReadFile(procPath)
	if err != nil {
		t.Fatal(err)
	}
	var proc ProcessResult
	if err := json.Unmarshal(raw, &proc); err != nil {
		t.Fatal(err)
	}
	if proc.PID != 7777 {
		t.Fatalf("recorded pid = %d, want 7777", proc.PID)
	}
	if proc.Metrics["memory.current"] != "1024" {
		t.Fatalf("metrics not persisted: %#v", proc.Metrics)
	}
	if len(result.Evidence.Processes) == 0 || result.Evidence.Processes[0].PID != 7777 {
		t.Fatalf("evidence processes missing: %#v", result.Evidence.Processes)
	}
}

func TestEvidenceWriteFailureFailsIntegrateWorker(t *testing.T) {
	dir := t.TempDir()
	store, err := NewRunStore(dir, "write-fail")
	if err != nil {
		t.Fatal(err)
	}
	// Block process evidence path: make "processes" a file so nested WriteJSON cannot create the artifact.
	if err := os.WriteFile(filepath.Join(store.Dir, "processes"), []byte("blocked"), 0o600); err != nil {
		t.Fatal(err)
	}

	session := newLiveSession(store, nil, ReviewRemediationTicket(), dir)
	session.WorkerCommand = "true"
	session.ProcessRT = &recordingProcessRuntime{pid: 9}
	err = session.runWorkerOnce(context.Background(), "write-fail")
	if err == nil {
		t.Fatal("expected evidence write failure")
	}
}
