package events

import (
	"sync"
	"time"
)

type Event struct {
	ID        string         `json:"id"`
	TaskID    string         `json:"task_id"`
	AgentID   string         `json:"agent_id,omitempty"`
	Type      string         `json:"type"`
	Source    string         `json:"source"`
	Timestamp int64          `json:"timestamp"`
	Payload   map[string]any `json:"payload"`
}

func New(eventType, taskID, agentID, source string, payload map[string]any) Event {
	return Event{
		ID:        time.Now().Format("20060102150405.000000000"),
		TaskID:    taskID,
		AgentID:   agentID,
		Type:      eventType,
		Source:    source,
		Timestamp: time.Now().UnixMilli(),
		Payload:   payload,
	}
}

type Hub struct {
	mu          sync.RWMutex
	buffer      int
	subscribers map[chan Event]struct{}
}

func NewHub(buffer int) *Hub {
	if buffer <= 0 {
		buffer = 1
	}
	return &Hub{
		buffer:      buffer,
		subscribers: make(map[chan Event]struct{}),
	}
}

func (h *Hub) Subscribe() (<-chan Event, func()) {
	ch := make(chan Event, h.buffer)
	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	h.mu.Unlock()
	cancel := func() {
		h.mu.Lock()
		if _, ok := h.subscribers[ch]; ok {
			delete(h.subscribers, ch)
			close(ch)
		}
		h.mu.Unlock()
	}
	return ch, cancel
}

func (h *Hub) Publish(event Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.subscribers {
		select {
		case ch <- event:
		default:
		}
	}
}

func (h *Hub) SubscriberCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subscribers)
}
