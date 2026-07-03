package trace

import (
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
