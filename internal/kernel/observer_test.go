package kernel

import (
	"context"
	"strings"
	"testing"

	"aort-r/internal/events"
)

type testSink struct {
	events []events.Event
}

func (s *testSink) Publish(event events.Event) {
	s.events = append(s.events, event)
}

func TestObserverStartsInDegradedProxyModeWhenEBPFUnavailable(t *testing.T) {
	sink := &testSink{}
	observer := NewObserver(Config{
		Sink:    sink,
		BTFPath: "/path/that/does/not/exist",
		BPFFS:   "/path/that/does/not/exist",
	})

	if err := observer.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	status := observer.Status()
	if status.Enabled {
		t.Fatalf("status = %#v", status)
	}
	if status.Mode != ModeDegradedProxy {
		t.Fatalf("mode = %q", status.Mode)
	}
	if !strings.Contains(status.Reason, "eBPF") {
		t.Fatalf("reason = %q", status.Reason)
	}
	if len(sink.events) != 1 || sink.events[0].Type != "kernel.observer_disabled" {
		t.Fatalf("events = %#v", sink.events)
	}
}

func TestObserverRecordsExecEventWithKernelTimelinePayload(t *testing.T) {
	sink := &testSink{}
	observer := NewObserver(Config{Sink: sink, BTFPath: "/missing", BPFFS: "/missing"})
	if err := observer.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	event := observer.ObserveExec(ExecObservation{
		TaskID:    "task-1",
		AgentID:   "agent-1",
		PID:       123,
		Command:   "go",
		Args:      []string{"test", "./..."},
		Workspace: "/tmp/aort/agent-1",
		Status:    "OK",
	})

	if event.Type != "kernel.exec" || event.Source != "kernel" || event.AgentID != "agent-1" {
		t.Fatalf("event = %#v", event)
	}
	if event.Payload["probe"] != ProbeSyscallGatewayProxy || event.Payload["mode"] != ModeDegradedProxy {
		t.Fatalf("payload = %#v", event.Payload)
	}
	records := observer.Events()
	if len(records) != 1 || records[0].PID != 123 || records[0].Command != "go" {
		t.Fatalf("records = %#v", records)
	}
	if records[0].Args == nil {
		t.Fatalf("args should be an empty slice, got nil")
	}
	if observer.Status().EventCount != 1 {
		t.Fatalf("status = %#v", observer.Status())
	}
}
