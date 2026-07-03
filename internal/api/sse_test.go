package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aort-r/internal/config"
	"aort-r/internal/events"
)

func TestEventsEndpointStreamsSSE(t *testing.T) {
	hub := events.NewHub(4)
	handler := NewServerWithEvents(config.Config{Mode: "mock"}, hub)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/api/events", nil)
	rec := httptest.NewRecorder()

	done := make(chan struct{})
	go func() {
		handler.ServeHTTP(rec, req)
		close(done)
	}()

	waitForSubscriber(t, hub)
	hub.Publish(events.Event{ID: "e1", TaskID: "t1", Type: "task.updated", Source: "runtime", Timestamp: 1})

	deadline := time.After(time.Second)
	for {
		if strings.Contains(rec.Body.String(), "event: task.updated") &&
			strings.Contains(rec.Body.String(), `"type":"task.updated"`) {
			cancel()
			<-done
			return
		}
		select {
		case <-deadline:
			t.Fatalf("SSE body = %q", rec.Body.String())
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func waitForSubscriber(t *testing.T, hub *events.Hub) {
	t.Helper()
	deadline := time.After(time.Second)
	for {
		if hub.SubscriberCount() == 1 {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("subscriber was not registered")
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}
