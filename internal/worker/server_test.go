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
