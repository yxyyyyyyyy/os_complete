package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"aort-r/internal/events"
)

type recordingSink struct {
	events []events.Event
}

func (s *recordingSink) Publish(event events.Event) {
	s.events = append(s.events, event)
}

func TestManagerRollsBackAgentWorkspaceWithoutTouchingBaseSnapshot(t *testing.T) {
	sink := &recordingSink{}
	manager := NewManager(Config{Root: t.TempDir(), Sink: sink})

	snapshot, err := manager.CreateBaseSnapshot("task-1", map[string]string{
		"README.md":      "# base\n",
		"src/service.go": "package src\n",
	})
	if err != nil {
		t.Fatalf("CreateBaseSnapshot: %v", err)
	}
	runtime, err := manager.PrepareAgent("task-1", "agent-1")
	if err != nil {
		t.Fatalf("PrepareAgent: %v", err)
	}
	t.Cleanup(func() {
		_ = manager.Destroy("agent-1")
	})
	if runtime.Mode != ModeDegradedCopy && runtime.Mode != ModeOverlayFS {
		t.Fatalf("mode = %q", runtime.Mode)
	}

	if err := os.WriteFile(filepath.Join(runtime.WorkspacePath, "src/service.go"), []byte("corrupted\n"), 0o644); err != nil {
		t.Fatalf("write corruption: %v", err)
	}
	if _, err := removeChildren(manager.root, runtime.WorkspacePath); err != nil {
		t.Fatalf("remove workspace children: %v", err)
	}

	result, err := manager.Rollback("agent-1")
	if err != nil {
		t.Fatalf("Rollback: %v", err)
	}
	if !result.RollbackSuccess || !result.BaseIntact {
		t.Fatalf("rollback result = %#v", result)
	}
	baseContent, err := os.ReadFile(filepath.Join(snapshot.BasePath, "src/service.go"))
	if err != nil {
		t.Fatalf("read base: %v", err)
	}
	if string(baseContent) != "package src\n" {
		t.Fatalf("base was modified: %q", baseContent)
	}
	restored, err := os.ReadFile(filepath.Join(runtime.WorkspacePath, "src/service.go"))
	if err != nil {
		t.Fatalf("read restored workspace: %v", err)
	}
	if string(restored) != "package src\n" {
		t.Fatalf("workspace was not restored: %q", restored)
	}
	if !hasEvent(sink.events, "workspace.created") || !hasEvent(sink.events, "workspace.rollback") {
		t.Fatalf("events = %#v", sink.events)
	}
}

func TestManagerRunsRMFaultAndReportsEvidence(t *testing.T) {
	manager := NewManager(Config{Root: t.TempDir()})
	if _, err := manager.CreateBaseSnapshot("task-1", map[string]string{"main.go": "package main\n"}); err != nil {
		t.Fatalf("CreateBaseSnapshot: %v", err)
	}

	result, err := manager.InjectRMAndRollback("task-1", "agent-1")
	if err != nil {
		t.Fatalf("InjectRMAndRollback: %v", err)
	}
	t.Cleanup(func() {
		_ = manager.Destroy("agent-1")
	})
	if result.Mode != ModeDegradedCopy && result.Mode != ModeOverlayFS {
		t.Fatalf("mode = %q", result.Mode)
	}
	if !result.RollbackSuccess || !result.BaseIntact {
		t.Fatalf("result = %#v", result)
	}
	if result.RemovedEntries == 0 {
		t.Fatalf("expected removed entries: %#v", result)
	}
}

func hasEvent(events []events.Event, eventType string) bool {
	for _, event := range events {
		if event.Type == eventType {
			return true
		}
	}
	return false
}
