package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aort-r/internal/avp"
	"aort-r/internal/capsule"
	"aort-r/internal/config"
	"aort-r/internal/worker"
)

func TestCapsulesEndpointReturnsEvidenceModeAndStats(t *testing.T) {
	srv := NewServer(config.Config{HTTPAddr: "127.0.0.1:8080", Mode: "mock", DataDir: t.TempDir()})
	server, ok := srv.(*Server)
	if !ok {
		t.Fatalf("server type = %T", srv)
	}
	server.registry = worker.NewRegistry(server.sink)
	server.capsules = capsule.NewManager(capsule.Config{AllowDegraded: true})
	server.registry.CreateAgent("agent-capsule", "Coder", "task-1")
	rt, err := server.capsules.Prepare("agent-capsule", 0)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	server.registry.SetCapsule("agent-capsule", rt.CgroupPath, rt.Mode)
	server.registry.SetState("agent-capsule", avp.StateRunning)

	req := httptest.NewRequest(http.MethodGet, "/api/capsules", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var capsules []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &capsules); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(capsules) != 1 {
		t.Fatalf("capsules=%#v", capsules)
	}
	got := capsules[0]
	if got["agent_id"] != "agent-capsule" || got["capsule_mode"] != "degraded" {
		t.Fatalf("capsule summary=%#v", got)
	}
	if got["evidence_mode"] != "degraded" {
		t.Fatalf("evidence_mode=%#v", got["evidence_mode"])
	}
	if got["real_cgroup_v2"] != false {
		t.Fatalf("real_cgroup_v2=%#v", got["real_cgroup_v2"])
	}
	if got["cgroup_path"] != "degraded://agent-capsule" {
		t.Fatalf("cgroup_path=%#v", got["cgroup_path"])
	}
}

func TestCapsuleDetailAndActionsUseCapsuleRoutes(t *testing.T) {
	srv := NewServer(config.Config{HTTPAddr: "127.0.0.1:8080", Mode: "mock", DataDir: t.TempDir()})
	server := srv.(*Server)
	server.registry = worker.NewRegistry(server.sink)
	server.capsules = capsule.NewManager(capsule.Config{AllowDegraded: true})
	server.registry.CreateAgent("agent-capsule", "Coder", "task-1")
	rt, err := server.capsules.Prepare("agent-capsule", 0)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	server.registry.SetCapsule("agent-capsule", rt.CgroupPath, rt.Mode)

	detailReq := httptest.NewRequest(http.MethodGet, "/api/capsules/agent-capsule", nil)
	detailRec := httptest.NewRecorder()
	srv.ServeHTTP(detailRec, detailReq)
	if detailRec.Code != http.StatusOK || !strings.Contains(detailRec.Body.String(), `"evidence_mode":"degraded"`) {
		t.Fatalf("detail status=%d body=%s", detailRec.Code, detailRec.Body.String())
	}

	freezeReq := httptest.NewRequest(http.MethodPost, "/api/capsules/agent-capsule/freeze", nil)
	freezeRec := httptest.NewRecorder()
	srv.ServeHTTP(freezeRec, freezeReq)
	if freezeRec.Code != http.StatusConflict || !strings.Contains(freezeRec.Body.String(), "capsule degraded") {
		t.Fatalf("freeze status=%d body=%s", freezeRec.Code, freezeRec.Body.String())
	}
}
