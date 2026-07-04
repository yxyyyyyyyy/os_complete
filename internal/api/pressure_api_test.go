package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aort-r/internal/config"
	"aort-r/internal/events"
)

func TestPressureStatusEndpoint(t *testing.T) {
	srv := NewServer(config.Config{HTTPAddr: "127.0.0.1:8080", Mode: "mock", DataDir: t.TempDir()})

	req := httptest.NewRequest(http.MethodGet, "/api/pressure/status", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "mode") || !strings.Contains(rec.Body.String(), "throttle") {
		t.Fatalf("body=%s", rec.Body.String())
	}
}

func TestSchedulerEventIncludesPressureSnapshot(t *testing.T) {
	hub := events.NewHub(32)
	ch, cancel := hub.Subscribe()
	defer cancel()
	srv := NewServerWithEvents(config.Config{HTTPAddr: "127.0.0.1:8080", Mode: "mock", DataDir: t.TempDir()}, hub)

	req := httptest.NewRequest(http.MethodPost, "/api/demo/run", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("demo status=%d body=%s", rec.Code, rec.Body.String())
	}

	deadline := time.After(time.Second)
	for {
		select {
		case event := <-ch:
			if event.Type == "scheduler.selected" {
				if _, ok := event.Payload["pressure_mode"]; !ok {
					t.Fatalf("scheduler event missing pressure fields: %#v", event.Payload)
				}
				return
			}
		case <-deadline:
			t.Fatalf("did not receive scheduler.selected")
		}
	}
}
