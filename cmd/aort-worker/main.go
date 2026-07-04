package main

import (
	"encoding/json"
	"flag"
	"log"
	"net"
	"os"
	"time"

	"aort-r/internal/avp"
	"aort-r/internal/worker"
)

func main() {
	agentID := flag.String("agent-id", "", "agent id")
	role := flag.String("role", "", "agent role")
	taskID := flag.String("task-id", "", "task id")
	socketPath := flag.String("sock", "", "aortd Unix domain socket path")
	workMS := flag.Int("work-ms", 800, "mock work duration before report")
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
	send := func(message worker.Message) {
		if err := encoder.Encode(message); err != nil {
			log.Fatalf("send %s: %v", message.Type, err)
		}
	}

	pid := os.Getpid()
	send(worker.Message{
		Type:    worker.MessageRegister,
		AgentID: *agentID,
		Role:    *role,
		TaskID:  *taskID,
		PID:     pid,
	})

	heartbeat := time.NewTicker(2 * time.Second)
	defer heartbeat.Stop()
	reportTimer := time.NewTimer(time.Duration(*workMS) * time.Millisecond)
	defer reportTimer.Stop()
	reported := false

	for {
		select {
		case <-heartbeat.C:
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
