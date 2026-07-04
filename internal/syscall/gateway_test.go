package syscallgw

import (
	"context"
	"strings"
	"testing"

	"aort-r/internal/cvm"
	"aort-r/internal/events"
)

type recordingSink struct {
	events []events.Event
}

func (s *recordingSink) Publish(event events.Event) {
	s.events = append(s.events, event)
}

func TestGatewayMaterializesContextAndAuditsRecord(t *testing.T) {
	store := cvm.NewStore(nil)
	page, err := store.CreatePage(cvm.KindSystem, "system prompt\n")
	if err != nil {
		t.Fatalf("CreatePage: %v", err)
	}
	if err := store.MountPage("agent-1", page.ID); err != nil {
		t.Fatalf("MountPage: %v", err)
	}
	sink := &recordingSink{}
	gateway := NewGateway(Config{CVM: store, Sink: sink, WorkspaceRoot: t.TempDir()})

	response := gateway.Handle(context.Background(), Request{
		AgentID: "agent-1",
		TaskID:  "task-1",
		Name:    "context.materialize",
	})

	if response.Status != StatusOK {
		t.Fatalf("status = %s error=%s", response.Status, response.Error)
	}
	if response.Payload["content"] != "system prompt\n" {
		t.Fatalf("payload = %#v", response.Payload)
	}
	records := gateway.Records()
	if len(records) != 1 {
		t.Fatalf("records = %#v", records)
	}
	if records[0].Name != "context.materialize" || records[0].Status != StatusOK {
		t.Fatalf("record = %#v", records[0])
	}
	if len(sink.events) != 2 || sink.events[0].Type != "syscall.started" || sink.events[1].Type != "syscall.finished" {
		t.Fatalf("events = %#v", sink.events)
	}
}

func TestGatewayToolExecTimeoutIsAudited(t *testing.T) {
	gateway := NewGateway(Config{WorkspaceRoot: t.TempDir()})

	response := gateway.Handle(context.Background(), Request{
		AgentID: "agent-1",
		TaskID:  "task-1",
		Name:    "tool.exec",
		Args: map[string]any{
			"command":    "sleep",
			"args":       []any{"1"},
			"timeout_ms": 10,
		},
	})

	if response.Status != StatusTimeout {
		t.Fatalf("status = %s error=%s", response.Status, response.Error)
	}
	if !strings.Contains(response.Error, "timeout") {
		t.Fatalf("error = %q", response.Error)
	}
	records := gateway.Records()
	if len(records) != 1 || records[0].Status != StatusTimeout {
		t.Fatalf("records = %#v", records)
	}
	if records[0].DurationMS <= 0 {
		t.Fatalf("duration was not recorded: %#v", records[0])
	}
}

func TestGatewayRejectsToolExecOutsideWorkspace(t *testing.T) {
	gateway := NewGateway(Config{WorkspaceRoot: t.TempDir()})

	response := gateway.Handle(context.Background(), Request{
		AgentID: "agent-1",
		TaskID:  "task-1",
		Name:    "tool.exec",
		Args: map[string]any{
			"command": "pwd",
			"cwd":     "/tmp",
		},
	})

	if response.Status != StatusDenied {
		t.Fatalf("status = %s error=%s", response.Status, response.Error)
	}
}
