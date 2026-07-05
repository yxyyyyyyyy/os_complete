package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aort-r/internal/config"
)

func TestSoftwareRealDemoRunProducesRuntimeEvidence(t *testing.T) {
	srv := NewServer(config.Config{HTTPAddr: "127.0.0.1:8080", Mode: "mock", DataDir: t.TempDir()})
	req := httptest.NewRequest(http.MethodPost, "/api/demo/software-real/run", bytes.NewBufferString(`{"requirement":"实现一个带测试的字符串工具函数"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var result map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["evidence_mode"] != "real-runtime" || result["final_status"] != "success" {
		t.Fatalf("result=%#v", result)
	}
	if int(result["dag_nodes"].(float64)) != 6 || int(result["scheduler_decision_count"].(float64)) < 6 {
		t.Fatalf("scheduler/dag result=%#v", result)
	}
	if int(result["syscall_count"].(float64)) < 8 || int(result["tool_exec_count"].(float64)) < 2 {
		t.Fatalf("syscall/tool result=%#v", result)
	}
	if result["fault_injected"] != true || result["fault_recovered"] != true {
		t.Fatalf("fault result=%#v", result)
	}
	if result["first_test_status"] != "failed" || result["second_test_status"] != "passed" {
		t.Fatalf("test recovery result=%#v", result)
	}
	if result["checkpoint_used"] != true || int(result["checkpoint_count"].(float64)) < 1 {
		t.Fatalf("checkpoint evidence result=%#v", result)
	}

	demoID := result["demo_id"].(string)
	statusReq := httptest.NewRequest(http.MethodGet, "/api/demo/software-real/status", nil)
	statusRec := httptest.NewRecorder()
	srv.ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusOK || !strings.Contains(statusRec.Body.String(), demoID) {
		t.Fatalf("status endpoint=%d body=%s", statusRec.Code, statusRec.Body.String())
	}

	resultReq := httptest.NewRequest(http.MethodGet, "/api/demo/software-real/result", nil)
	resultRec := httptest.NewRecorder()
	srv.ServeHTTP(resultRec, resultReq)
	if resultRec.Code != http.StatusOK || !strings.Contains(resultRec.Body.String(), `"final_status":"success"`) {
		t.Fatalf("result endpoint=%d body=%s", resultRec.Code, resultRec.Body.String())
	}

	for _, check := range []struct {
		path string
		want string
	}{
		{"/api/syscalls", "context.materialize"},
		{"/api/syscalls", "context.write_delta"},
		{"/api/syscalls", "llm.call"},
		{"/api/syscalls", "tool.exec"},
		{"/api/syscalls", "ipc.publish"},
		{"/api/syscalls", "ipc.poll"},
		{"/api/syscalls", "agent.spawn"},
		{"/api/syscalls", "agent.report"},
		{"/api/scheduler/decisions", "token-cfs-prefix-affinity"},
		{"/api/context/stats", "saved_tokens"},
		{"/api/ipc/metrics", "avoided_copy_bytes"},
		{"/api/checkpoints", "software-real"},
		{"/api/faults", "TOOL_TIMEOUT"},
	} {
		req := httptest.NewRequest(http.MethodGet, check.path, nil)
		rec := httptest.NewRecorder()
		srv.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), check.want) {
			t.Fatalf("%s status=%d want %q body=%s", check.path, rec.Code, check.want, rec.Body.String())
		}
	}
}
