package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"aort-r/internal/capsule"
	"aort-r/internal/config"
	"aort-r/internal/worker"
)

func TestEvidenceEndpointReportsModuleModes(t *testing.T) {
	for _, key := range []string{
		"DEEPSEEK_API_KEY", "DEEPSEEK_BASE_URL", "DEEPSEEK_MODEL",
		"AORT_LLM_PROVIDER", "AORT_LLM_FALLBACK_PROVIDER",
	} {
		t.Setenv(key, "")
	}
	srv := NewServer(config.Config{HTTPAddr: "127.0.0.1:8080", Mode: "mock", DataDir: t.TempDir()})
	server := srv.(*Server)
	server.registry = worker.NewRegistry(server.sink)
	server.capsules = fakeRealCapsuleManager(t)
	server.registry.CreateAgent("agent-real", "Coder", "task-1")
	server.registry.HandleMessage(worker.Message{
		Type:    worker.MessageRegister,
		AgentID: "agent-real",
		TaskID:  "task-1",
		Role:    "Coder",
		PID:     12345,
	})
	rt, err := server.capsules.Prepare("agent-real", 12345)
	if err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	server.registry.SetCapsule("agent-real", rt.CgroupPath, rt.Mode)

	req := httptest.NewRequest(http.MethodGet, "/api/evidence", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var body struct {
		Modules []struct {
			Name     string `json:"name"`
			Status   string `json:"status"`
			Mode     string `json:"mode"`
			Endpoint string `json:"endpoint"`
		} `json:"modules"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	modes := map[string]string{}
	statuses := map[string]string{}
	endpoints := map[string]string{}
	for _, module := range body.Modules {
		modes[module.Name] = module.Mode
		statuses[module.Name] = module.Status
		endpoints[module.Name] = module.Endpoint
	}
	if statuses["Cgroup Capsule"] != "real" || modes["Cgroup Capsule"] != "cgroup-v2" {
		t.Fatalf("cgroup evidence status=%q mode=%q body=%s", statuses["Cgroup Capsule"], modes["Cgroup Capsule"], rec.Body.String())
	}
	if statuses["Worker Process"] != "real" {
		t.Fatalf("worker process status=%q", statuses["Worker Process"])
	}
	if statuses["LLM Provider"] != "mock" {
		t.Fatalf("llm provider status=%q", statuses["LLM Provider"])
	}
	if statuses["eBPF Observer"] != "degraded" && statuses["eBPF Observer"] != "real" && statuses["eBPF Observer"] != "real-ebpf" {
		t.Fatalf("ebpf observer status=%q", statuses["eBPF Observer"])
	}
	if statuses["Software Real Demo"] != "real-runtime" || endpoints["Software Real Demo"] != "/api/demo/software-real/run" {
		t.Fatalf("software-real evidence missing status=%q endpoint=%q body=%s", statuses["Software Real Demo"], endpoints["Software Real Demo"], rec.Body.String())
	}
}

func fakeRealCapsuleManager(t *testing.T) *capsule.Manager {
	t.Helper()
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "cgroup.controllers"), "cpu memory pids\n")
	writeTestFile(t, filepath.Join(root, "cgroup.subtree_control"), "\n")
	return capsule.NewManager(capsule.Config{
		Root:          root,
		ForceReal:     true,
		AllowDegraded: false,
	})
}

func writeTestFile(t *testing.T, path string, value string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(value), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
}
