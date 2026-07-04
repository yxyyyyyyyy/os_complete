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

func TestAgentsEndpointReturnsDemoAgents(t *testing.T) {
	srv := NewServer(config.Config{HTTPAddr: "127.0.0.1:8080", Mode: "mock"})
	runReq := httptest.NewRequest(http.MethodPost, "/api/demo/run", nil)
	runRec := httptest.NewRecorder()
	srv.ServeHTTP(runRec, runReq)

	agentsReq := httptest.NewRequest(http.MethodGet, "/api/agents", nil)
	agentsRec := httptest.NewRecorder()
	srv.ServeHTTP(agentsRec, agentsReq)
	if agentsRec.Code != http.StatusOK {
		t.Fatalf("status = %d body=%s", agentsRec.Code, agentsRec.Body.String())
	}
	if !strings.Contains(agentsRec.Body.String(), "planner") {
		t.Fatalf("body = %s", agentsRec.Body.String())
	}
}

func TestContextAPIShowsPagesStatsAndPageTable(t *testing.T) {
	srv := NewServer(config.Config{HTTPAddr: "127.0.0.1:8080", Mode: "mock"})
	runReq := httptest.NewRequest(http.MethodPost, "/api/demo/run", nil)
	runRec := httptest.NewRecorder()
	srv.ServeHTTP(runRec, runReq)

	pagesReq := httptest.NewRequest(http.MethodGet, "/api/context/pages", nil)
	pagesRec := httptest.NewRecorder()
	srv.ServeHTTP(pagesRec, pagesReq)
	if pagesRec.Code != http.StatusOK || !strings.Contains(pagesRec.Body.String(), "project") {
		t.Fatalf("pages status=%d body=%s", pagesRec.Code, pagesRec.Body.String())
	}

	statsReq := httptest.NewRequest(http.MethodGet, "/api/context/stats", nil)
	statsRec := httptest.NewRecorder()
	srv.ServeHTTP(statsRec, statsReq)
	if statsRec.Code != http.StatusOK || !strings.Contains(statsRec.Body.String(), "saved_tokens") {
		t.Fatalf("stats status=%d body=%s", statsRec.Code, statsRec.Body.String())
	}

	tableReq := httptest.NewRequest(http.MethodGet, "/api/context/agents/planner-1/pagetable", nil)
	tableRec := httptest.NewRecorder()
	srv.ServeHTTP(tableRec, tableReq)
	if tableRec.Code != http.StatusOK || !strings.Contains(tableRec.Body.String(), "planner-1") {
		t.Fatalf("pagetable status=%d body=%s", tableRec.Code, tableRec.Body.String())
	}
}

func TestDemoRunProducesIPCEvidence(t *testing.T) {
	srv := NewServer(config.Config{HTTPAddr: "127.0.0.1:8080", Mode: "mock"})
	runReq := httptest.NewRequest(http.MethodPost, "/api/demo/run", nil)
	runRec := httptest.NewRecorder()
	srv.ServeHTTP(runRec, runReq)

	syscallsReq := httptest.NewRequest(http.MethodGet, "/api/syscalls", nil)
	syscallsRec := httptest.NewRecorder()
	srv.ServeHTTP(syscallsRec, syscallsReq)
	if syscallsRec.Code != http.StatusOK || !strings.Contains(syscallsRec.Body.String(), "ipc.publish") || !strings.Contains(syscallsRec.Body.String(), "ipc.poll") {
		t.Fatalf("syscalls status=%d body=%s", syscallsRec.Code, syscallsRec.Body.String())
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/api/ipc/metrics", nil)
	metricsRec := httptest.NewRecorder()
	srv.ServeHTTP(metricsRec, metricsReq)
	if metricsRec.Code != http.StatusOK || !strings.Contains(metricsRec.Body.String(), "avoided_copy_bytes") {
		t.Fatalf("metrics status=%d body=%s", metricsRec.Code, metricsRec.Body.String())
	}

	topicsReq := httptest.NewRequest(http.MethodGet, "/api/ipc/topics", nil)
	topicsRec := httptest.NewRecorder()
	srv.ServeHTTP(topicsRec, topicsReq)
	if topicsRec.Code != http.StatusOK || !strings.Contains(topicsRec.Body.String(), "review.feedback") {
		t.Fatalf("topics status=%d body=%s", topicsRec.Code, topicsRec.Body.String())
	}
}

func TestDemoRunCreatesCheckpointEvidence(t *testing.T) {
	srv := NewServer(config.Config{HTTPAddr: "127.0.0.1:8080", Mode: "mock", DataDir: t.TempDir()})
	runReq := httptest.NewRequest(http.MethodPost, "/api/demo/run", nil)
	runRec := httptest.NewRecorder()
	srv.ServeHTTP(runRec, runReq)

	checkpointReq := httptest.NewRequest(http.MethodGet, "/api/checkpoints", nil)
	checkpointRec := httptest.NewRecorder()
	srv.ServeHTTP(checkpointRec, checkpointReq)
	if checkpointRec.Code != http.StatusOK || !strings.Contains(checkpointRec.Body.String(), "runtime-state") {
		t.Fatalf("checkpoints status=%d body=%s", checkpointRec.Code, checkpointRec.Body.String())
	}
	if !strings.Contains(checkpointRec.Body.String(), "planner-1") {
		t.Fatalf("checkpoint missing agents: %s", checkpointRec.Body.String())
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
