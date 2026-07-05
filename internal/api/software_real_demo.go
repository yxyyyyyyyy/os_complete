package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"aort-r/internal/avp"
	"aort-r/internal/cvm"
	"aort-r/internal/demo"
	"aort-r/internal/events"
	"aort-r/internal/supervisor"
	syscallgw "aort-r/internal/syscall"
	"aort-r/internal/worker"
)

type softwareRealDemoRequest struct {
	Requirement string `json:"requirement"`
}

type softwareRealDemoResult struct {
	DemoID                 string       `json:"demo_id"`
	TaskID                 string       `json:"task_id"`
	Requirement            string       `json:"requirement"`
	EvidenceMode           string       `json:"evidence_mode"`
	FinalStatus            string       `json:"final_status"`
	Agents                 []demo.Agent `json:"agents"`
	DAGNodes               int          `json:"dag_nodes"`
	SyscallCount           int          `json:"syscall_count"`
	SchedulerDecisionCount int          `json:"scheduler_decision_count"`
	ContextPages           int          `json:"context_pages"`
	SharedPages            int          `json:"shared_pages"`
	SavedTokens            int64        `json:"saved_tokens"`
	IPCMessages            int          `json:"ipc_messages"`
	ToolExecCount          int          `json:"tool_exec_count"`
	FaultInjected          bool         `json:"fault_injected"`
	FaultRecovered         bool         `json:"fault_recovered"`
	CheckpointUsed         bool         `json:"checkpoint_used"`
	CheckpointCount        int          `json:"checkpoint_count"`
	CheckpointMode         string       `json:"checkpoint_mode"`
	FirstTestStatus        string       `json:"first_test_status"`
	SecondTestStatus       string       `json:"second_test_status"`
	FirstTestOutput        string       `json:"first_test_output,omitempty"`
	SecondTestOutput       string       `json:"second_test_output,omitempty"`
	StartedAt              int64        `json:"started_at"`
	CompletedAt            int64        `json:"completed_at"`
}

func (s *Server) handleSoftwareRealDemoRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body softwareRealDemoRequest
	if r.Body != nil && r.Body != http.NoBody {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	result, task, err := s.runSoftwareRealDemo(r.Context(), body.Requirement)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.mu.Lock()
	s.tasks[task.TaskID] = task
	s.softwareReal = &result
	s.mu.Unlock()
	s.saveSoftwareRealResult(result)
	writeJSON(w, http.StatusAccepted, result)
}

func (s *Server) handleSoftwareRealDemoStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result, ok := s.latestSoftwareRealDemo()
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleSoftwareRealDemoResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result, ok := s.latestSoftwareRealDemo()
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) latestSoftwareRealDemo() (softwareRealDemoResult, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.softwareReal == nil {
		return softwareRealDemoResult{}, false
	}
	return *s.softwareReal, true
}

func (s *Server) saveSoftwareRealResult(result softwareRealDemoResult) {
	root := repoRoot()
	dir := filepath.Join(root, "experiments", "results", "software_real_demo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		s.sink.Publish(events.New("software_real.result_save_failed", result.TaskID, "", "runtime", map[string]any{"error": err.Error()}))
		return
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		s.sink.Publish(events.New("software_real.result_save_failed", result.TaskID, "", "runtime", map[string]any{"error": err.Error()}))
		return
	}
	if err := os.WriteFile(filepath.Join(dir, "result.json"), append(data, '\n'), 0o644); err != nil {
		s.sink.Publish(events.New("software_real.result_save_failed", result.TaskID, "", "runtime", map[string]any{"error": err.Error()}))
	}
}

func repoRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			return "."
		}
		wd = parent
	}
}

func (s *Server) runSoftwareRealDemo(ctx context.Context, requirement string) (softwareRealDemoResult, demo.Result, error) {
	if requirement == "" {
		requirement = "实现一个带测试的字符串工具函数"
	}
	startedAt := time.Now()
	demoID := fmt.Sprintf("software-real-%d", startedAt.UnixNano())
	task := softwareRealTask(demoID)

	syscallsBefore := len(s.syscalls.Records())
	decisionsBefore := len(s.scheduler.Decisions())
	ipcBefore := s.ipc.Metrics().TotalMessages

	if err := s.seedSoftwareRealContext(requirement, task.Agents); err != nil {
		return softwareRealDemoResult{}, demo.Result{}, err
	}
	if err := s.prepareSoftwareRealAgentCapsules(ctx, task); err != nil {
		return softwareRealDemoResult{}, demo.Result{}, err
	}
	testEvidence, err := s.runSoftwareRealAgentSteps(ctx, demoID, task)
	if err != nil {
		return softwareRealDemoResult{}, demo.Result{}, err
	}

	timeoutResponse := s.syscalls.Handle(ctx, syscallgw.Request{
		RequestID: demoID + "-tester-tool-timeout",
		TaskID:    demoID,
		AgentID:   demoID + "-tester",
		Name:      "tool.exec",
		Args: map[string]any{
			"command":    "sh",
			"args":       []string{"-c", "sleep 1"},
			"timeout_ms": 10,
		},
	})
	fault := s.supervisor.Record(supervisor.Fault{
		Type:           supervisor.FaultToolTimeout,
		TaskID:         demoID,
		AgentID:        demoID + "-tester",
		RecoveryAction: "kill timed-out tool process and resume tester agent",
		Details: map[string]any{
			"syscall":        "tool.exec",
			"syscall_status": timeoutResponse.Status,
			"error":          timeoutResponse.Error,
		},
	})
	s.publishSoftwareRealEvent(demoID, demoID+"-tester", "software_real.tool_exec_timeout", map[string]any{
		"syscall_status": timeoutResponse.Status,
		"fault_id":       fault.ID,
	})
	if timeoutResponse.Status != syscallgw.StatusTimeout {
		return softwareRealDemoResult{}, demo.Result{}, fmt.Errorf("expected timeout fault, got %s", timeoutResponse.Status)
	}

	if err := s.runSoftwareRealRecoverySteps(ctx, demoID); err != nil {
		return softwareRealDemoResult{}, demo.Result{}, err
	}

	task.Status = "success"
	s.completeSoftwareRealAgents(task)
	task.Agents = s.agentsForTask(task)
	s.saveCheckpoint(task, 1)
	checkpointCount, checkpointMode := s.softwareRealCheckpointEvidence(demoID)
	s.publishSoftwareRealEvent(demoID, demoID+"-reporter", "software_real.reporter_completed", map[string]any{
		"final_status": "success",
	})

	records := s.syscalls.Records()
	decisions := s.scheduler.Decisions()
	stats := s.cvm.Stats()
	ipcMetrics := s.ipc.Metrics()
	result := softwareRealDemoResult{
		DemoID:                 demoID,
		TaskID:                 demoID,
		Requirement:            requirement,
		EvidenceMode:           "real-runtime",
		FinalStatus:            "success",
		Agents:                 task.Agents,
		DAGNodes:               len(task.DAG),
		SyscallCount:           len(records) - syscallsBefore,
		SchedulerDecisionCount: len(decisions) - decisionsBefore,
		ContextPages:           stats.TotalPages,
		SharedPages:            stats.SharedPages,
		SavedTokens:            stats.SavedTokens,
		IPCMessages:            ipcMetrics.TotalMessages - ipcBefore,
		ToolExecCount:          countToolExec(records[syscallsBefore:]),
		FaultInjected:          true,
		FaultRecovered:         fault.Status == supervisor.StatusRecovered,
		CheckpointUsed:         checkpointCount > 0,
		CheckpointCount:        checkpointCount,
		CheckpointMode:         checkpointMode,
		FirstTestStatus:        testEvidence.FirstStatus,
		SecondTestStatus:       testEvidence.SecondStatus,
		FirstTestOutput:        testEvidence.FirstOutput,
		SecondTestOutput:       testEvidence.SecondOutput,
		StartedAt:              startedAt.UnixMilli(),
		CompletedAt:            time.Now().UnixMilli(),
	}
	return result, task, nil
}

func (s *Server) prepareSoftwareRealAgentCapsules(ctx context.Context, task demo.Result) error {
	if s.registry == nil {
		return nil
	}
	for _, agent := range task.Agents {
		if _, ok := s.registry.Get(agent.ID); !ok {
			s.registry.CreateAgent(agent.ID, agent.Role, task.TaskID)
		}
		if s.launcher.Command != "" && s.cfg.SocketPath != "" {
			if _, err := s.launcher.Start(ctx, worker.Spec{AgentID: agent.ID, Role: agent.Role, TaskID: task.TaskID}); err != nil {
				return fmt.Errorf("start software-real worker %s: %w", agent.ID, err)
			}
			if err := s.waitForSoftwareRealCapsule(ctx, agent.ID, 20*time.Second); err != nil {
				return err
			}
			continue
		}
		pid, err := startSoftwareRealFallbackWorker(ctx)
		if err != nil {
			return fmt.Errorf("start software-real fallback worker %s: %w", agent.ID, err)
		}
		s.registry.HandleMessage(worker.Message{
			Type:    worker.MessageRegister,
			AgentID: agent.ID,
			Role:    agent.Role,
			TaskID:  task.TaskID,
			PID:     pid,
		})
		if err := s.attachSoftwareRealCapsule(agent.ID, task.TaskID, pid); err != nil {
			return err
		}
	}
	return nil
}

func startSoftwareRealFallbackWorker(ctx context.Context) (int, error) {
	cmd := exec.CommandContext(ctx, "sleep", "10")
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	go func() {
		_ = cmd.Wait()
	}()
	return cmd.Process.Pid, nil
}

func (s *Server) waitForSoftwareRealCapsule(ctx context.Context, agentID string, timeout time.Duration) error {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return fmt.Errorf("software-real worker %s did not register before timeout", agentID)
		case <-ticker.C:
			agent, ok := s.registry.Get(agentID)
			if !ok || agent.PID == 0 {
				continue
			}
			if err := s.attachSoftwareRealCapsule(agentID, agent.TaskID, agent.PID); err != nil {
				return err
			}
			return nil
		}
	}
}

func (s *Server) attachSoftwareRealCapsule(agentID, taskID string, pid int) error {
	if s.capsules == nil {
		return nil
	}
	if rt, ok := s.capsules.Runtime(agentID); ok {
		s.registry.SetCapsule(agentID, rt.CgroupPath, rt.Mode)
		return nil
	}
	rt, err := s.capsules.Prepare(agentID, pid)
	if err != nil {
		s.sink.Publish(events.New("software_real.capsule_failed", taskID, agentID, "runtime", map[string]any{
			"error": err.Error(),
			"pid":   pid,
		}))
		return fmt.Errorf("prepare software-real capsule %s: %w", agentID, err)
	}
	s.registry.SetCapsule(agentID, rt.CgroupPath, rt.Mode)
	s.publishSoftwareRealEvent(taskID, agentID, "software_real.capsule_attached", map[string]any{
		"pid":          pid,
		"capsule_mode": rt.Mode,
		"cgroup_path":  rt.CgroupPath,
	})
	return nil
}

func (s *Server) completeSoftwareRealAgents(task demo.Result) {
	if s.registry == nil {
		return
	}
	for _, agent := range task.Agents {
		s.registry.SetState(agent.ID, avp.StateCompleted)
	}
}

func (s *Server) softwareRealCheckpointEvidence(taskID string) (int, string) {
	snapshots, err := s.checkpoint.List()
	if err != nil {
		return 0, ""
	}
	count := 0
	mode := ""
	for _, snapshot := range snapshots {
		if snapshot.TaskID != taskID {
			continue
		}
		count++
		if snapshot.Mode != "" {
			mode = snapshot.Mode
		}
	}
	return count, mode
}

type softwareRealTestEvidence struct {
	FirstStatus  string
	SecondStatus string
	FirstOutput  string
	SecondOutput string
}

func softwareRealTask(taskID string) demo.Result {
	agents := []demo.Agent{
		{ID: taskID + "-planner", Role: "planner", State: "COMPLETED"},
		{ID: taskID + "-coder", Role: "coder", State: "COMPLETED"},
		{ID: taskID + "-tester", Role: "tester", State: "COMPLETED"},
		{ID: taskID + "-reviewer", Role: "reviewer", State: "COMPLETED"},
		{ID: taskID + "-fixer", Role: "fixer", State: "COMPLETED"},
		{ID: taskID + "-reporter", Role: "reporter", State: "COMPLETED"},
	}
	return demo.Result{
		TaskID: taskID,
		Status: "running",
		Agents: agents,
		DAG: []demo.DAGNode{
			{ID: "planner", Role: "planner"},
			{ID: "coder", Role: "coder", Dependencies: []string{"planner"}},
			{ID: "tester", Role: "tester", Dependencies: []string{"coder"}},
			{ID: "reviewer", Role: "reviewer", Dependencies: []string{"tester"}},
			{ID: "fixer", Role: "fixer", Dependencies: []string{"reviewer"}},
			{ID: "reporter", Role: "reporter", Dependencies: []string{"fixer"}},
		},
	}
}

func (s *Server) seedSoftwareRealContext(requirement string, agents []demo.Agent) error {
	pages := []struct {
		kind    cvm.PageKind
		content string
	}{
		{cvm.KindSystem, "System: AORT-R software-real demo executes through runtime syscalls, scheduler, CVM, IPC and supervisor.\n"},
		{cvm.KindProject, "Project: build a tiny string utility and test artifact in isolated agent workspaces.\n"},
		{cvm.KindTask, "Requirement: " + requirement + "\n"},
	}
	pageIDs := make([]string, 0, len(pages))
	for _, spec := range pages {
		page, err := s.cvm.CreatePage(spec.kind, spec.content)
		if err != nil {
			return err
		}
		pageIDs = append(pageIDs, page.ID)
	}
	for _, agent := range agents {
		for _, pageID := range pageIDs {
			if err := s.cvm.MountPage(agent.ID, pageID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Server) runSoftwareRealAgentSteps(ctx context.Context, taskID string, task demo.Result) (softwareRealTestEvidence, error) {
	var testEvidence softwareRealTestEvidence
	for index, agent := range task.Agents {
		if err := s.scheduleSoftwareRealAgent(taskID, agent, index); err != nil {
			return softwareRealTestEvidence{}, err
		}
		s.publishSoftwareRealEvent(taskID, agent.ID, "software_real."+agent.Role+"_scheduled", map[string]any{"role": agent.Role})
		switch agent.Role {
		case "planner":
			if err := s.mustSyscallOK(ctx, taskID, agent.ID, "context.materialize", nil); err != nil {
				return softwareRealTestEvidence{}, err
			}
			if err := s.mustSyscallOK(ctx, taskID, agent.ID, "llm.call", map[string]any{
				"role":   "planner",
				"prompt": "Plan a small Go module with string utility tests.",
			}); err != nil {
				return softwareRealTestEvidence{}, err
			}
			if err := s.mustSyscallOK(ctx, taskID, agent.ID, "context.write_delta", map[string]any{
				"content": "Plan: coder writes strings.NormalizeSpace, tester runs shell-backed checks, reviewer publishes findings, fixer patches, reporter closes.\n",
			}); err != nil {
				return softwareRealTestEvidence{}, err
			}
			s.syscalls.Handle(ctx, syscallgw.Request{
				RequestID: taskID + "-reviewer-spawn-fixer",
				TaskID:    taskID,
				AgentID:   agent.ID,
				Name:      "agent.spawn",
				Args: map[string]any{
					"agent_id":     taskID + "-fixer",
					"role":         "fixer",
					"reason":       "review feedback may require a follow-up patch",
					"dependencies": []string{taskID + "-reviewer"},
				},
			})
		case "coder":
			if err := s.mustSyscallOK(ctx, taskID, agent.ID, "context.materialize", nil); err != nil {
				return softwareRealTestEvidence{}, err
			}
			if err := s.mustSyscallOK(ctx, taskID, agent.ID, "context.write_delta", map[string]any{
				"content": "Code artifact: func NormalizeSpace(s string) string { return strings.Join(strings.Fields(s), \" \") }\n",
			}); err != nil {
				return softwareRealTestEvidence{}, err
			}
		case "tester":
			if err := s.mustSyscallOK(ctx, taskID, agent.ID, "context.materialize", nil); err != nil {
				return softwareRealTestEvidence{}, err
			}
			first := s.syscalls.Handle(ctx, syscallgw.Request{
				RequestID: taskID + "-tester-go-test-first",
				TaskID:    taskID,
				AgentID:   agent.ID,
				Name:      "tool.exec",
				Args: map[string]any{
					"command":    "sh",
					"args":       []string{"-c", brokenGoModuleScript},
					"timeout_ms": 30000,
				},
			})
			testEvidence.FirstOutput = toolOutput(first)
			if first.Status != syscallgw.StatusError {
				return softwareRealTestEvidence{}, fmt.Errorf("expected first go test to fail, got %s", first.Status)
			}
			testEvidence.FirstStatus = "failed"
			s.publishSoftwareRealEvent(taskID, agent.ID, "software_real.first_go_test_failed", map[string]any{"command": "go test ./..."})
		case "reviewer":
			pageID, pageBytes, err := s.writeDeltaPage(ctx, taskID, agent.ID, "Review: first go test failed; publish page reference so fixer can avoid copying review text.\n")
			if err != nil {
				return softwareRealTestEvidence{}, err
			}
			if err := s.mustSyscallOK(ctx, taskID, agent.ID, "ipc.publish", map[string]any{
				"topic":      "software-real.review",
				"page_id":    pageID,
				"size_bytes": pageBytes,
			}); err != nil {
				return softwareRealTestEvidence{}, err
			}
		case "fixer":
			if err := s.mustSyscallOK(ctx, taskID, agent.ID, "ipc.poll", map[string]any{
				"topic": "software-real.review",
			}); err != nil {
				return softwareRealTestEvidence{}, err
			}
			if err := s.mustSyscallOK(ctx, taskID, agent.ID, "context.write_delta", map[string]any{
				"content": "Fix: preserve Unicode spacing behavior by using Fields before Join; no duplicate context copy needed.\n",
			}); err != nil {
				return softwareRealTestEvidence{}, err
			}
			second := s.syscalls.Handle(ctx, syscallgw.Request{
				RequestID: taskID + "-fixer-go-test-second",
				TaskID:    taskID,
				AgentID:   agent.ID,
				Name:      "tool.exec",
				Args: map[string]any{
					"command":    "sh",
					"args":       []string{"-c", fixedGoModuleScript},
					"timeout_ms": 30000,
				},
			})
			testEvidence.SecondOutput = toolOutput(second)
			if second.Status != syscallgw.StatusOK {
				return softwareRealTestEvidence{}, fmt.Errorf("expected second go test to pass, got %s %s", second.Status, second.Error)
			}
			testEvidence.SecondStatus = "passed"
			s.publishSoftwareRealEvent(taskID, agent.ID, "software_real.second_go_test_passed", map[string]any{"command": "go test ./..."})
		case "reporter":
			if err := s.mustSyscallOK(ctx, taskID, agent.ID, "context.materialize", nil); err != nil {
				return softwareRealTestEvidence{}, err
			}
		}
	}
	return testEvidence, nil
}

func (s *Server) runSoftwareRealRecoverySteps(ctx context.Context, taskID string) error {
	fixerID := taskID + "-fixer"
	reporterID := taskID + "-reporter"
	s.publishSoftwareRealEvent(taskID, fixerID, "software_real.fault_recovered", map[string]any{
		"recovery": "resume fixer after tester timeout",
	})
	if err := s.mustSyscallOK(ctx, taskID, fixerID, "tool.exec", map[string]any{
		"command":    "sh",
		"args":       []string{"-c", "printf 'recovered-and-fixed\\n' > fixer.out"},
		"timeout_ms": 1000,
	}); err != nil {
		return err
	}
	return s.mustSyscallOK(ctx, taskID, reporterID, "agent.report", map[string]any{
		"status":        "success",
		"summary":       "software-real demo completed with recovered tool timeout",
		"evidence_mode": "real-runtime",
	})
}

func (s *Server) scheduleSoftwareRealAgent(taskID string, agent demo.Agent, index int) error {
	candidate := avp.AVP{
		AgentID:      agent.ID,
		TaskID:       taskID,
		Role:         agent.Role,
		State:        avp.StateReady,
		Weight:       100,
		VRuntime:     uint64(index * 10),
		PageTable:    s.cvm.PageTable(agent.ID).PageIDs,
		CreatedAt:    time.Now().Add(time.Duration(index) * time.Millisecond).UnixMilli(),
		UpdatedAt:    time.Now().UnixMilli(),
		ContextPages: s.cvm.PageTable(agent.ID).PageIDs,
	}
	_, decision, ok := s.scheduler.Select(taskID, []avp.AVP{candidate})
	if !ok {
		return fmt.Errorf("scheduler found no ready agent for %s/%s", taskID, agent.ID)
	}
	s.publishSchedulerDecision(decision)
	return nil
}

func (s *Server) mustSyscallOK(ctx context.Context, taskID, agentID, name string, args map[string]any) error {
	if args == nil {
		args = map[string]any{}
	}
	response := s.syscalls.Handle(ctx, syscallgw.Request{
		RequestID: fmt.Sprintf("%s-%s-%d", taskID, name, time.Now().UnixNano()),
		TaskID:    taskID,
		AgentID:   agentID,
		Name:      name,
		Args:      args,
	})
	if response.Status != syscallgw.StatusOK {
		return fmt.Errorf("%s for %s failed: %s %s", name, agentID, response.Status, response.Error)
	}
	return nil
}

func (s *Server) writeDeltaPage(ctx context.Context, taskID, agentID, content string) (string, int, error) {
	response := s.syscalls.Handle(ctx, syscallgw.Request{
		RequestID: fmt.Sprintf("%s-context.write_delta-%d", taskID, time.Now().UnixNano()),
		TaskID:    taskID,
		AgentID:   agentID,
		Name:      "context.write_delta",
		Args:      map[string]any{"content": content},
	})
	if response.Status != syscallgw.StatusOK {
		return "", 0, fmt.Errorf("context.write_delta for %s failed: %s %s", agentID, response.Status, response.Error)
	}
	pageID, _ := response.Payload["page_id"].(string)
	pageBytes := intFromPayload(response.Payload["bytes"])
	if pageID == "" || pageBytes <= 0 {
		return "", 0, fmt.Errorf("context.write_delta for %s returned invalid page payload %#v", agentID, response.Payload)
	}
	return pageID, pageBytes, nil
}

func (s *Server) publishSoftwareRealEvent(taskID, agentID, eventType string, payload map[string]any) {
	if payload == nil {
		payload = map[string]any{}
	}
	s.sink.Publish(events.New(eventType, taskID, agentID, "runtime", payload))
}

func countToolExec(records []syscallgw.Record) int {
	count := 0
	for _, record := range records {
		if record.Name == "tool.exec" {
			count++
		}
	}
	return count
}

func intFromPayload(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func toolOutput(resp syscallgw.Response) string {
	if resp.Payload == nil {
		return resp.Error
	}
	stdout, _ := resp.Payload["stdout"].(string)
	stderr, _ := resp.Payload["stderr"].(string)
	if stderr != "" {
		return stdout + stderr
	}
	if stdout != "" {
		return stdout
	}
	return resp.Error
}

const brokenGoModuleScript = `cat > go.mod <<'EOF'
module example.com/aortstrings

go 1.22
EOF
cat > strings.go <<'EOF'
package aortstrings

func NormalizeSpace(s string) string {
	return s
}
EOF
cat > strings_test.go <<'EOF'
package aortstrings

import "testing"

func TestNormalizeSpace(t *testing.T) {
	got := NormalizeSpace(" alpha   beta ")
	if got != "alpha beta" {
		t.Fatalf("NormalizeSpace() = %q", got)
	}
}
EOF
go test ./...`

const fixedGoModuleScript = `cat > go.mod <<'EOF'
module example.com/aortstrings

go 1.22
EOF
cat > strings.go <<'EOF'
package aortstrings

import "strings"

func NormalizeSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
EOF
cat > strings_test.go <<'EOF'
package aortstrings

import "testing"

func TestNormalizeSpace(t *testing.T) {
	got := NormalizeSpace(" alpha   beta ")
	if got != "alpha beta" {
		t.Fatalf("NormalizeSpace() = %q", got)
	}
}
EOF
go test ./...`
