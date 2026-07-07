package trace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aort-r/internal/events"
)

func TestRecorderWritesJSONL(t *testing.T) {
	dir := t.TempDir()
	rec, err := NewRecorder(dir)
	if err != nil {
		t.Fatalf("NewRecorder: %v", err)
	}
	event := events.Event{ID: "e1", TaskID: "t1", Type: "agent.created", Source: "runtime", Timestamp: 1}
	if err := rec.Append(event); err != nil {
		t.Fatalf("Append: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "t1.jsonl"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(data), `"agent.created"`) {
		t.Fatalf("trace data = %s", data)
	}
}

func TestRecorderRejectsEventWithoutTaskID(t *testing.T) {
	rec, err := NewRecorder(t.TempDir())
	if err != nil {
		t.Fatalf("NewRecorder: %v", err)
	}
	if err := rec.Append(events.Event{ID: "e1", Type: "agent.created"}); err == nil {
		t.Fatalf("expected missing task id error")
	}
}

func TestTraceWriteReadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.json")
	want := []TraceEvent{
		{EventID: "e1", Timestamp: "2026-07-07T00:00:00Z", Type: "scheduler_decision", AgentID: "agent-1", TaskID: "task-1", Payload: map[string]any{"score": 1}},
	}
	if err := WriteTrace(path, want); err != nil {
		t.Fatalf("WriteTrace: %v", err)
	}
	got, err := ReadTrace(path)
	if err != nil {
		t.Fatalf("ReadTrace: %v", err)
	}
	if len(got) != 1 || got[0].EventID != "e1" || got[0].Type != "scheduler_decision" {
		t.Fatalf("got = %#v", got)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var decoded []TraceEvent
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("trace must be a JSON array: %v\n%s", err, raw)
	}
}
