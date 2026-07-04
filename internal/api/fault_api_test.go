package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aort-r/internal/config"
)

func TestToolTimeoutFaultEndpointRecordsFaultAndSyscall(t *testing.T) {
	srv := NewServer(config.Config{HTTPAddr: "127.0.0.1:8080", Mode: "mock"})

	faultReq := httptest.NewRequest(http.MethodPost, "/api/demo/fault/tool-timeout", nil)
	faultRec := httptest.NewRecorder()
	srv.ServeHTTP(faultRec, faultReq)
	if faultRec.Code != http.StatusAccepted || !strings.Contains(faultRec.Body.String(), "TOOL_TIMEOUT") {
		t.Fatalf("fault status=%d body=%s", faultRec.Code, faultRec.Body.String())
	}

	faultsReq := httptest.NewRequest(http.MethodGet, "/api/faults", nil)
	faultsRec := httptest.NewRecorder()
	srv.ServeHTTP(faultsRec, faultsReq)
	if faultsRec.Code != http.StatusOK || !strings.Contains(faultsRec.Body.String(), "TOOL_TIMEOUT") {
		t.Fatalf("faults status=%d body=%s", faultsRec.Code, faultsRec.Body.String())
	}

	syscallsReq := httptest.NewRequest(http.MethodGet, "/api/syscalls", nil)
	syscallsRec := httptest.NewRecorder()
	srv.ServeHTTP(syscallsRec, syscallsReq)
	if syscallsRec.Code != http.StatusOK || !strings.Contains(syscallsRec.Body.String(), "TIMEOUT") {
		t.Fatalf("syscalls status=%d body=%s", syscallsRec.Code, syscallsRec.Body.String())
	}
}
