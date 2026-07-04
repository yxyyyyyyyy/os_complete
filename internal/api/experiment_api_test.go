package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aort-r/internal/config"
)

func TestExperimentResultsEndpointReturnsE1E2E3(t *testing.T) {
	srv := NewServer(config.Config{HTTPAddr: "127.0.0.1:8080", Mode: "mock"})

	req := httptest.NewRequest(http.MethodGet, "/api/experiments/results", nil)
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	for _, expected := range []string{"e1_scheduler", "e2_fault", "e3_context", "saved_tokens"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("missing %s in %s", expected, body)
		}
	}
}
