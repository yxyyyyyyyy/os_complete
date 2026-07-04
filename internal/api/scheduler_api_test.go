package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aort-r/internal/config"
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
