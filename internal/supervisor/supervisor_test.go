package supervisor

import (
	"testing"

	"aort-r/internal/events"
)

type recordingSink struct {
	events []events.Event
}

func (s *recordingSink) Publish(event events.Event) {
	s.events = append(s.events, event)
}

func TestManagerRecordsToolTimeoutFault(t *testing.T) {
	sink := &recordingSink{}
	manager := NewManager(sink)

	record := manager.Record(Fault{
		Type:           FaultToolTimeout,
		TaskID:         "task-1",
		AgentID:        "agent-1",
		RecoveryAction: "tool process killed by timeout context",
		Details:        map[string]any{"syscall": "tool.exec"},
	})

	if record.ID == "" || record.Type != FaultToolTimeout || record.Status != StatusRecovered {
		t.Fatalf("record = %#v", record)
	}
	records := manager.Records()
	if len(records) != 1 {
		t.Fatalf("records = %#v", records)
	}
	if len(sink.events) != 1 || sink.events[0].Type != "supervisor.detected" {
		t.Fatalf("events = %#v", sink.events)
	}
}
