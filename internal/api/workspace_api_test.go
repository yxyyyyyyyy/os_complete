package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aort-r/internal/config"
	"aort-r/internal/evidence"
)

func TestSchedulerPoliciesAndResourcePressureEndpoints(t *testing.T) {
	srv := NewServer(config.Config{HTTPAddr: "127.0.0.1:8080", Mode: "mock", DataDir: t.TempDir()})

	policiesReq := httptest.NewRequest(http.MethodGet, "/api/scheduler/policies", nil)
	policiesRec := httptest.NewRecorder()
	srv.ServeHTTP(policiesRec, policiesReq)
	if policiesRec.Code != http.StatusOK || !strings.Contains(policiesRec.Body.String(), "token-cfs-prefix-affinity-resource-aware") {
		t.Fatalf("policies status=%d body=%s", policiesRec.Code, policiesRec.Body.String())
	}

	pressureReq := httptest.NewRequest(http.MethodGet, "/api/scheduler/resource-pressure", nil)
	pressureRec := httptest.NewRecorder()
	srv.ServeHTTP(pressureRec, pressureReq)
	if pressureRec.Code != http.StatusOK {
		t.Fatalf("pressure status=%d body=%s", pressureRec.Code, pressureRec.Body.String())
	}
	var pressure map[string]any
	if err := json.Unmarshal(pressureRec.Body.Bytes(), &pressure); err != nil {
		t.Fatalf("decode pressure: %v", err)
	}
	if pressure["evidence_mode"] == "" || pressure["fallback_reason"] == nil {
		t.Fatalf("pressure evidence fields missing: %#v", pressure)
	}
}

func TestWorkspaceEndpointsAndFaultDemoReturnEvidenceMode(t *testing.T) {
	srv := NewServer(config.Config{HTTPAddr: "127.0.0.1:8080", Mode: "mock", DataDir: t.TempDir()})

	listReq := httptest.NewRequest(http.MethodGet, "/api/workspaces", nil)
	listRec := httptest.NewRecorder()
	srv.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listRec.Code, listRec.Body.String())
	}

	demoReq := httptest.NewRequest(http.MethodPost, "/api/demo/fault/workspace-rmrf", nil)
	demoRec := httptest.NewRecorder()
	srv.ServeHTTP(demoRec, demoReq)
	if demoRec.Code != http.StatusAccepted {
		t.Fatalf("demo status=%d body=%s", demoRec.Code, demoRec.Body.String())
	}
	body := demoRec.Body.String()
	for _, want := range []string{`"success":true`, `"fault_type":"workspace_rmrf"`, `"evidence_mode":"` + string(evidence.ModeDegradedCopy)} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %s in %s", want, body)
		}
	}

	if !strings.Contains(body, `"destroy_success":true`) {
		t.Fatalf("destroy evidence missing: %s", body)
	}
}
