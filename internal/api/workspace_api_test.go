package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
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
	dataDir := t.TempDir()
	srv := NewServer(config.Config{HTTPAddr: "127.0.0.1:8080", Mode: "mock", DataDir: dataDir})

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
	var demo map[string]any
	if err := json.Unmarshal(demoRec.Body.Bytes(), &demo); err != nil {
		t.Fatalf("decode workspace demo: %v\n%s", err, body)
	}
	if demo["success"] != true || demo["fault_type"] != "workspace_rmrf" {
		t.Fatalf("workspace demo identity missing: %#v", demo)
	}
	runtimeRoot, _ := demo["runtime_root"].(string)
	if !strings.HasPrefix(runtimeRoot, filepath.Join(dataDir, "runtime", "workspaces")) {
		t.Fatalf("workspace demo should use server data dir root, got %q", runtimeRoot)
	}
	mode, _ := demo["evidence_mode"].(string)
	if mode != string(evidence.ModeDegradedCopy) && mode != string(evidence.ModeRealOverlayFS) {
		t.Fatalf("unexpected workspace evidence mode %q in %s", mode, body)
	}
	fallbackReason, _ := demo["fallback_reason"].(string)
	if mode == string(evidence.ModeDegradedCopy) && fallbackReason == "" {
		t.Fatalf("degraded-copy demo must explain fallback: %s", body)
	}
	if mode == string(evidence.ModeRealOverlayFS) && fallbackReason != "" {
		t.Fatalf("real-overlayfs demo should not report fallback: %s", body)
	}

	if !strings.Contains(body, `"destroy_success":true`) {
		t.Fatalf("destroy evidence missing: %s", body)
	}
}
