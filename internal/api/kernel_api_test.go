package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aort-r/internal/config"
)

func TestKernelAPIsExposeObserverStatusAndExecEvents(t *testing.T) {
	srv := NewServer(config.Config{HTTPAddr: "127.0.0.1:8080", Mode: "mock", DataDir: t.TempDir()})

	statusReq := httptest.NewRequest(http.MethodGet, "/api/kernel/status", nil)
	statusRec := httptest.NewRecorder()
	srv.ServeHTTP(statusRec, statusReq)
	if statusRec.Code != http.StatusOK || !strings.Contains(statusRec.Body.String(), `"mode":"degraded"`) {
		t.Fatalf("kernel status=%d body=%s", statusRec.Code, statusRec.Body.String())
	}

	runReq := httptest.NewRequest(http.MethodPost, "/api/demo/run", nil)
	runRec := httptest.NewRecorder()
	srv.ServeHTTP(runRec, runReq)
	if runRec.Code != http.StatusAccepted {
		t.Fatalf("demo status=%d body=%s", runRec.Code, runRec.Body.String())
	}

	eventsReq := httptest.NewRequest(http.MethodGet, "/api/kernel/events", nil)
	eventsRec := httptest.NewRecorder()
	srv.ServeHTTP(eventsRec, eventsReq)
	if eventsRec.Code != http.StatusOK || !strings.Contains(eventsRec.Body.String(), "kernel.exec") {
		t.Fatalf("kernel events=%d body=%s", eventsRec.Code, eventsRec.Body.String())
	}
	if !strings.Contains(eventsRec.Body.String(), "syscall-gateway-proxy") {
		t.Fatalf("kernel events missing degraded probe: %s", eventsRec.Body.String())
	}
}
