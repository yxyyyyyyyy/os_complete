package worker

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"aort-r/internal/avp"
)

func TestUDSServerAcceptsRegisterMessage(t *testing.T) {
	registry := NewRegistry(nil)
	registry.CreateAgent("agent-planner", "Planner", "task-1")
	dir, err := os.MkdirTemp("/private/tmp", "aortd-uds-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(dir)
	socketPath := filepath.Join(dir, "a.sock")
	server := NewUDSServer(socketPath, registry)
	if err := server.Start(); err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("unix socket bind blocked by sandbox: %v", err)
		}
		t.Fatalf("Start: %v", err)
	}
	defer server.Close()

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	msg := Message{Type: MessageRegister, AgentID: "agent-planner", Role: "Planner", TaskID: "task-1", PID: 12345}
	if err := json.NewEncoder(conn).Encode(msg); err != nil {
		t.Fatalf("Encode: %v", err)
	}

	deadline := time.After(time.Second)
	for {
		agent, _ := registry.Get("agent-planner")
		if agent.PID == 12345 && agent.State == avp.StateRunning {
			return
		}
		select {
		case <-deadline:
			t.Fatalf("agent did not register: %#v", agent)
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
}

type fakeSyscallHandler struct {
	request Message
}

func (h *fakeSyscallHandler) HandleSyscall(request Message) Response {
	h.request = request
	return Response{
		Type:      MessageSyscallResult,
		RequestID: request.RequestID,
		AgentID:   request.AgentID,
		TaskID:    request.TaskID,
		Status:    "OK",
		Payload:   map[string]any{"content": "materialized"},
	}
}

func TestUDSServerReturnsSyscallResponse(t *testing.T) {
	registry := NewRegistry(nil)
	registry.CreateAgent("agent-planner", "Planner", "task-1")
	handler := &fakeSyscallHandler{}
	dir, err := os.MkdirTemp("/private/tmp", "aortd-uds-test-*")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	defer os.RemoveAll(dir)
	socketPath := filepath.Join(dir, "a.sock")
	server := NewUDSServer(socketPath, registry, handler)
	if err := server.Start(); err != nil {
		if strings.Contains(err.Error(), "operation not permitted") {
			t.Skipf("unix socket bind blocked by sandbox: %v", err)
		}
		t.Fatalf("Start: %v", err)
	}
	defer server.Close()

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer conn.Close()

	request := Message{
		Type:      MessageSyscall,
		RequestID: "req-1",
		AgentID:   "agent-planner",
		TaskID:    "task-1",
		Name:      "context.materialize",
	}
	if err := json.NewEncoder(conn).Encode(request); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	var response Response
	if err := json.NewDecoder(conn).Decode(&response); err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if response.Type != MessageSyscallResult || response.RequestID != "req-1" || response.Status != "OK" {
		t.Fatalf("response = %#v", response)
	}
	if handler.request.Name != "context.materialize" {
		t.Fatalf("handler request = %#v", handler.request)
	}
}
