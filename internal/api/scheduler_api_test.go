package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aort-r/internal/config"
	"aort-r/internal/resource"
	"aort-r/internal/worker"
)

func TestSchedulerPolicyAndDecisionEndpoints(t *testing.T) {
	srv := NewServer(config.Config{HTTPAddr: "127.0.0.1:8080", Mode: "mock"})

	policyReq := httptest.NewRequest(http.MethodPost, "/api/scheduler/policy", strings.NewReader(`{"policy":"token-cfs"}`))
	policyRec := httptest.NewRecorder()
	srv.ServeHTTP(policyRec, policyReq)
	if policyRec.Code != http.StatusOK || !strings.Contains(policyRec.Body.String(), "token-cfs") {
		t.Fatalf("policy status=%d body=%s", policyRec.Code, policyRec.Body.String())
	}

	decisionsReq := httptest.NewRequest(http.MethodGet, "/api/scheduler/decisions", nil)
	decisionsRec := httptest.NewRecorder()
	srv.ServeHTTP(decisionsRec, decisionsReq)
	if decisionsRec.Code != http.StatusOK {
		t.Fatalf("decisions status=%d body=%s", decisionsRec.Code, decisionsRec.Body.String())
	}
}

func TestSchedulerPolicyRejectsUnsupportedPolicy(t *testing.T) {
	srv := NewServer(config.Config{HTTPAddr: "127.0.0.1:8080", Mode: "mock"})

	req := httptest.NewRequest(http.MethodPost, "/api/scheduler/policy", strings.NewReader(`{"policy":"random"}`))
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestSchedulerCandidatesUseResourceSampler(t *testing.T) {
	srv := NewServer(config.Config{HTTPAddr: "127.0.0.1:8080", Mode: "mock", DataDir: t.TempDir()})
	server := srv.(*Server)
	server.registry = worker.NewRegistry(server.sink)

	root := t.TempDir()
	cgroup := filepath.Join(root, "cgroup", "agent-sampled")
	writeFile(t, filepath.Join(cgroup, "memory.current"), "512\n")
	writeFile(t, filepath.Join(cgroup, "memory.max"), "1024\n")
	writeFile(t, filepath.Join(cgroup, "pids.current"), "4\n")
	writeFile(t, filepath.Join(cgroup, "pids.max"), "8\n")
	writeFile(t, filepath.Join(cgroup, "cpu.stat"), "nr_throttled 25\n")
	pressureRoot := filepath.Join(root, "pressure")
	writeFile(t, filepath.Join(pressureRoot, "cpu"), "some avg10=10.00 avg60=0.00 avg300=0.00 total=1\n")
	writeFile(t, filepath.Join(pressureRoot, "memory"), "some avg10=5.00 avg60=0.00 avg300=0.00 total=1\n")
	writeFile(t, filepath.Join(pressureRoot, "io"), "some avg10=0.00 avg60=0.00 avg300=0.00 total=1\n")
	server.resourceSampler = resource.NewCgroupSampler(pressureRoot)

	agent := server.registry.CreateAgent("agent-sampled", "Coder", "task-1")
	server.registry.SetCapsule(agent.AgentID, cgroup, "real")
	candidates := server.schedulerCandidates("task-1", []worker.Spec{{AgentID: agent.AgentID, Role: "Coder", TaskID: "task-1"}})
	if len(candidates) != 1 {
		t.Fatalf("candidates = %#v", candidates)
	}
	if candidates[0].MemoryCurrent != 512 || candidates[0].PidsCurrent != 4 || candidates[0].CPUStat["nr_throttled"] != 25 {
		t.Fatalf("candidate was not enriched: %#v", candidates[0])
	}
	pressure := server.scheduler.ResourcePressure()
	if pressure.MemoryPressure != 0.5 || pressure.PidsPressure != 0.5 || pressure.CPUThrottlePressure != 0.25 || pressure.PSIPressure != 0.1 {
		t.Fatalf("scheduler pressure not sampled: %#v", pressure)
	}
}

func writeFile(t *testing.T, path string, value string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir parent %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
