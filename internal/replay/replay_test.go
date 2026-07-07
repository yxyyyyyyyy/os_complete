package replay

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aort-r/internal/trace"
)

func TestReplaySuccess(t *testing.T) {
	dir := t.TempDir()
	tracePath := filepath.Join(dir, "trace.json")
	events := []trace.TraceEvent{
		{EventID: "e1", Timestamp: "2026-07-07T00:00:00Z", Type: "scheduler_decision", AgentID: "agent-1", TaskID: "task-1", Payload: map[string]any{"status": "running"}},
		{EventID: "e2", Timestamp: "2026-07-07T00:00:01Z", Type: "task_completed", AgentID: "agent-1", TaskID: "task-1", Payload: map[string]any{"final_status": "completed"}},
	}
	if err := trace.WriteTrace(tracePath, events); err != nil {
		t.Fatalf("WriteTrace: %v", err)
	}
	result, err := Run(tracePath, filepath.Join(dir, "replay"))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !result.ReplaySuccess || result.Divergence || result.EventCount != 2 {
		t.Fatalf("result = %#v", result)
	}
	if result.OriginalFinalStatus != "completed" || result.ReplayFinalStatus != "completed" {
		t.Fatalf("final statuses = %#v", result)
	}
	if _, err := os.Stat(filepath.Join(dir, "replay", "replay_result.json")); err != nil {
		t.Fatalf("replay evidence missing: %v", err)
	}
}

func TestReplayDetectsDivergence(t *testing.T) {
	dir := t.TempDir()
	tracePath := filepath.Join(dir, "trace.json")
	events := []trace.TraceEvent{
		{EventID: "late", Timestamp: "2026-07-07T00:00:02Z", Type: "scheduler_decision", TaskID: "task-1", Payload: map[string]any{"status": "running"}},
		{EventID: "early", Timestamp: "2026-07-07T00:00:01Z", Type: "task_completed", TaskID: "task-1", Payload: map[string]any{"final_status": "completed"}},
	}
	if err := trace.WriteTrace(tracePath, events); err != nil {
		t.Fatalf("WriteTrace: %v", err)
	}
	result, err := Run(tracePath, filepath.Join(dir, "replay"))
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !result.Divergence || result.DivergenceReason == "" {
		t.Fatalf("expected divergence: %#v", result)
	}
}

func TestReplayMissingTraceError(t *testing.T) {
	_, err := Run(filepath.Join(t.TempDir(), "missing.json"), t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "trace not found") {
		t.Fatalf("err = %v", err)
	}
}
