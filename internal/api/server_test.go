package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"aort-r/internal/config"
)

func TestHealthEndpoint(t *testing.T) {
	handler := NewServer(config.Config{HTTPAddr: "127.0.0.1:8080", Mode: "mock"})
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if body["status"] != "ok" || body["mode"] != "mock" {
		t.Fatalf("body = %#v", body)
	}
}
