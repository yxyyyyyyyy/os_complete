package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"aort-r/internal/config"
	"aort-r/internal/events"
)

func TestDemoRunEndpointCreatesTask(t *testing.T) {
	srv := NewServer(config.Config{HTTPAddr: "127.0.0.1:8080", Mode: "mock"})
	req := httptest.NewRequest(http.MethodPost, "/api/demo/run", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "task_id") {
		t.Fatalf("body = %s", rec.Body.String())
	}
}

func TestTaskAPIListsDemoTaskAndDAG(t *testing.T) {
	srv := NewServer(config.Config{HTTPAddr: "127.0.0.1:8080", Mode: "mock"})
	runReq := httptest.NewRequest(http.MethodPost, "/api/demo/run", nil)
	runRec := httptest.NewRecorder()
	srv.ServeHTTP(runRec, runReq)

	var body map[string]string
	if err := json.Unmarshal(runRec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid run response: %v", err)
	}
	taskID := body["task_id"]

	listReq := httptest.NewRequest(http.MethodGet, "/api/tasks", nil)
	listRec := httptest.NewRecorder()
	srv.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK || !strings.Contains(listRec.Body.String(), taskID) {
		t.Fatalf("list status=%d body=%s", listRec.Code, listRec.Body.String())
	}

	dagReq := httptest.NewRequest(http.MethodGet, "/api/tasks/"+taskID+"/dag", nil)
	dagRec := httptest.NewRecorder()
	srv.ServeHTTP(dagRec, dagReq)
	if dagRec.Code != http.StatusOK || !strings.Contains(dagRec.Body.String(), "planner") {
		t.Fatalf("dag status=%d body=%s", dagRec.Code, dagRec.Body.String())
	}
}

func TestDemoRunPublishesEventsToHub(t *testing.T) {
	hub := events.NewHub(32)
	ch, cancel := hub.Subscribe()
	defer cancel()
	srv := NewServerWithEvents(config.Config{HTTPAddr: "127.0.0.1:8080", Mode: "mock"}, hub)

	req := httptest.NewRequest(http.MethodPost, "/api/demo/run", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
	}

	deadline := time.After(time.Second)
	for {
		select {
		case event := <-ch:
			if event.Type == "task.completed" {
				return
			}
		case <-deadline:
			t.Fatalf("did not receive task.completed")
		}
	}
}
