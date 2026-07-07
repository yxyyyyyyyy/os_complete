package experiment

import (
	"fmt"
	"sync"
	"time"

	"aort-r/internal/events"
	"aort-r/internal/trace"
)

type runtimeTraceCollector struct {
	mu     sync.Mutex
	events []trace.TraceEvent
}

func newRuntimeTraceCollector() *runtimeTraceCollector {
	return &runtimeTraceCollector{}
}

func (c *runtimeTraceCollector) Publish(event events.Event) {
	payload := map[string]any{}
	for key, value := range event.Payload {
		payload[key] = value
	}
	eventID := event.ID
	if requestID, ok := payload["request_id"].(string); ok && requestID != "" {
		eventID = requestID
	}
	if eventID == "" {
		eventID = fmt.Sprintf("%s-%d", event.Type, event.Timestamp)
	}
	timestamp := time.UnixMilli(event.Timestamp).UTC().Format(time.RFC3339Nano)
	c.mu.Lock()
	c.events = append(c.events, trace.TraceEvent{
		EventID:   eventID,
		Timestamp: timestamp,
		Type:      event.Type,
		AgentID:   event.AgentID,
		TaskID:    event.TaskID,
		Payload:   payload,
	})
	c.mu.Unlock()
}

func (c *runtimeTraceCollector) Events() []trace.TraceEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]trace.TraceEvent, len(c.events))
	copy(out, c.events)
	return out
}

func (c *runtimeTraceCollector) Append(event trace.TraceEvent) {
	c.mu.Lock()
	c.events = append(c.events, event)
	c.mu.Unlock()
}
