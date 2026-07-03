package events

import (
	"testing"
	"time"
)

func TestHubPublishesToSubscriber(t *testing.T) {
	hub := NewHub(4)
	ch, cancel := hub.Subscribe()
	defer cancel()
	event := Event{ID: "e1", TaskID: "t1", Type: "task.updated", Source: "runtime", Timestamp: 1}
	hub.Publish(event)
	got := <-ch
	if got.ID != "e1" || got.Type != "task.updated" {
		t.Fatalf("event = %#v", got)
	}
}

func TestHubCancelRemovesSubscriber(t *testing.T) {
	hub := NewHub(1)
	_, cancel := hub.Subscribe()
	cancel()
	if got := hub.SubscriberCount(); got != 0 {
		t.Fatalf("subscribers = %d", got)
	}
}

func TestHubReplaysHistoryToNewSubscriber(t *testing.T) {
	hub := NewHub(4)
	event := Event{ID: "e1", TaskID: "t1", Type: "task.completed", Source: "runtime", Timestamp: 1}
	hub.Publish(event)
	ch, cancel := hub.Subscribe()
	defer cancel()
	select {
	case got := <-ch:
		if got.ID != "e1" || got.Type != "task.completed" {
			t.Fatalf("event = %#v", got)
		}
	case <-time.After(time.Second):
		t.Fatalf("did not receive replayed event")
	}
}
