package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"aort-r/internal/avp"
	"aort-r/internal/worker"
)

const defaultRetainBytes = 1 << 20

func main() {
	agentID := flag.String("agent-id", "", "agent id")
	role := flag.String("role", "", "agent role")
	taskID := flag.String("task-id", "", "task id")
	socketPath := flag.String("sock", "", "aortd Unix domain socket path")
	workMS := flag.Int("work-ms", 800, "mock work duration before report")
	retainBytes := flag.Int("retain-bytes", defaultRetainBytes, "bytes to retain after cgroup registration")
	flag.Parse()

	if *agentID == "" || *role == "" || *taskID == "" || *socketPath == "" {
		log.Fatalf("--agent-id, --role, --task-id, and --sock are required")
	}

	conn, err := net.Dial("unix", *socketPath)
	if err != nil {
		log.Fatalf("dial aortd socket: %v", err)
	}
	defer conn.Close()

	encoder := json.NewEncoder(conn)
	decoder := json.NewDecoder(conn)
	pid := os.Getpid()
	send := func(message worker.Message) {
		if err := encoder.Encode(message); err != nil {
			log.Fatalf("send %s: %v", message.Type, err)
		}
	}
	syscallSeq := 0
	call := func(name string, args map[string]any) worker.Response {
		syscallSeq++
		requestID := fmt.Sprintf("%s-syscall-%03d", *agentID, syscallSeq)
		send(worker.Message{
			Type:      worker.MessageSyscall,
			RequestID: requestID,
			AgentID:   *agentID,
			Role:      *role,
			TaskID:    *taskID,
			PID:       pid,
			Name:      name,
			Args:      args,
		})
		var response worker.Response
		if err := decoder.Decode(&response); err != nil {
			log.Fatalf("decode syscall %s response: %v", name, err)
		}
		if response.Status != "OK" {
			log.Printf("syscall %s status=%s error=%s", name, response.Status, response.Error)
		}
		return response
	}

	send(worker.Message{
		Type:    worker.MessageRegister,
		AgentID: *agentID,
		Role:    *role,
		TaskID:  *taskID,
		PID:     pid,
	})

	materialized := call("context.materialize", nil)
	retained := retainMemory(*retainBytes)
	tool := call("tool.exec", map[string]any{
		"command":    "go",
		"args":       []any{"version"},
		"timeout_ms": 2000,
	})
	call("context.write_delta", map[string]any{
		"content": fmt.Sprintf("%s saw %d context bytes and tool status %s.\n", *role, materialized.Payload["bytes"], tool.Status),
	})
	call("agent.report", map[string]any{
		"status": string(avp.StateCompleted),
		"role":   *role,
	})

	heartbeat := time.NewTicker(2 * time.Second)
	defer heartbeat.Stop()
	reportTimer := time.NewTimer(time.Duration(*workMS) * time.Millisecond)
	defer reportTimer.Stop()
	reported := true

	for {
		select {
		case <-heartbeat.C:
			if len(retained) > 0 {
				retained[0]++
			}
			send(worker.Message{Type: worker.MessageHeartbeat, AgentID: *agentID, Role: *role, TaskID: *taskID, PID: pid})
		case <-reportTimer.C:
			if !reported {
				send(worker.Message{
					Type:    worker.MessageReport,
					AgentID: *agentID,
					Role:    *role,
					TaskID:  *taskID,
					PID:     pid,
					Status:  string(avp.StateCompleted),
				})
				reported = true
			}
		}
	}
}

func retainMemory(bytes int) []byte {
	if bytes <= 0 {
		return nil
	}
	retained := make([]byte, bytes)
	pageSize := os.Getpagesize()
	if pageSize <= 0 {
		pageSize = 4096
	}
	for i := 0; i < len(retained); i += pageSize {
		retained[i] = byte((i / pageSize) + 1)
	}
	retained[len(retained)-1] = 1
	return retained
}
