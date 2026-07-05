package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"aort-r/internal/avp"
	"aort-r/internal/capsule"
	"aort-r/internal/checkpoint"
	"aort-r/internal/config"
	"aort-r/internal/cvm"
	"aort-r/internal/demo"
	"aort-r/internal/events"
	"aort-r/internal/evidence"
	"aort-r/internal/experiment"
	"aort-r/internal/ipc"
	"aort-r/internal/kernel"
	"aort-r/internal/llm"
	"aort-r/internal/pressure"
	"aort-r/internal/scheduler"
	"aort-r/internal/supervisor"
	syscallgw "aort-r/internal/syscall"
	"aort-r/internal/trace"
	"aort-r/internal/worker"
	"aort-r/internal/workspace"
)

func NewServer(cfg config.Config) http.Handler {
	return NewServerWithEvents(cfg, events.NewHub(32))
}

func NewServerWithEvents(cfg config.Config, hub *events.Hub) http.Handler {
	if cfg.DataDir == "" {
		cfg.DataDir = ".aort-dev"
	}
	sink := newEventSink(cfg, hub)
	server := &Server{
		cfg:        cfg,
		hub:        hub,
		sink:       sink,
		demo:       demo.NewSoftwareDemoRunner(),
		cvm:        cvm.NewStore(sink),
		ipc:        ipc.NewBlackboard(),
		kernel:     kernel.NewObserver(kernel.Config{Sink: sink}),
		pressure:   pressure.NewMonitor(pressure.Config{}),
		checkpoint: checkpoint.NewStore(filepath.Join(cfg.DataDir, "checkpoints")),
		workspace:  workspace.NewManager(workspace.Config{Root: workspace.DefaultRoot(), Sink: sink}),
		scheduler:  scheduler.New(scheduler.PolicyTokenCFSPrefixAffinity),
		supervisor: supervisor.NewManager(sink),
		tasks:      make(map[string]demo.Result),
		workerCtx:  context.Background(),
	}
	server.syscalls = syscallgw.NewGateway(syscallgw.Config{
		CVM:           server.cvm,
		IPC:           server.ipc,
		LLM:           newLLMRouter(),
		Sink:          server.sink,
		WorkspaceRoot: workspace.DefaultRoot(),
		Reporter:      server.handleAgentReport,
		Spawner:       server.spawnAgent,
		ExecObserver:  server.observeKernelExec,
	})
	_ = server.kernel.Start(context.Background())
	server.startWorkerRuntime()
	server.recoverFromCheckpoints()
	server.routes()
	return server
}

type Server struct {
	cfg              config.Config
	hub              *events.Hub
	sink             *eventSink
	demo             *demo.Runner
	cvm              *cvm.Store
	ipc              *ipc.Blackboard
	kernel           *kernel.Observer
	pressure         *pressure.Monitor
	checkpoint       *checkpoint.Store
	workspace        *workspace.Manager
	scheduler        *scheduler.Scheduler
	supervisor       *supervisor.Manager
	syscalls         *syscallgw.Gateway
	mux              *http.ServeMux
	mu               sync.RWMutex
	tasks            map[string]demo.Result
	softwareReal     *softwareRealDemoResult
	registry         *worker.Registry
	capsules         *capsule.Manager
	uds              *worker.UDSServer
	launcher         worker.Launcher
	heartbeatTimeout time.Duration
	workerCtx        context.Context
	recovery         checkpoint.RecoveryReport
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
			"mode":   s.cfg.Mode,
		})
	})
	mux.HandleFunc("/api/evidence", s.handleEvidence)
	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		streamEvents(w, r, s.hub)
	})
	mux.HandleFunc("/api/demo/run", s.handleDemoRun)
	mux.HandleFunc("/api/demo/software-real/run", s.handleSoftwareRealDemoRun)
	mux.HandleFunc("/api/demo/software-real/status", s.handleSoftwareRealDemoStatus)
	mux.HandleFunc("/api/demo/software-real/result", s.handleSoftwareRealDemoResult)
	mux.HandleFunc("/api/demo/fault/", s.handleDemoFault)
	mux.HandleFunc("/api/faults", s.handleFaults)
	mux.HandleFunc("/api/agents", s.handleAgents)
	mux.HandleFunc("/api/agents/", s.handleAgentAction)
	mux.HandleFunc("/api/capsules", s.handleCapsules)
	mux.HandleFunc("/api/capsules/", s.handleCapsuleSubresource)
	mux.HandleFunc("/api/context/pages", s.handleContextPages)
	mux.HandleFunc("/api/context/stats", s.handleContextStats)
	mux.HandleFunc("/api/context/agents/", s.handleContextAgent)
	mux.HandleFunc("/api/ipc/metrics", s.handleIPCMetrics)
	mux.HandleFunc("/api/ipc/topics", s.handleIPCTopics)
	mux.HandleFunc("/api/kernel/status", s.handleKernelStatus)
	mux.HandleFunc("/api/kernel/events", s.handleKernelEvents)
	mux.HandleFunc("/api/pressure/status", s.handlePressureStatus)
	mux.HandleFunc("/api/checkpoints", s.handleCheckpoints)
	mux.HandleFunc("/api/recovery/status", s.handleRecoveryStatus)
	mux.HandleFunc("/api/syscalls", s.handleSyscalls)
	mux.HandleFunc("/api/scheduler/decisions", s.handleSchedulerDecisions)
	mux.HandleFunc("/api/scheduler/policies", s.handleSchedulerPolicies)
	mux.HandleFunc("/api/scheduler/resource-pressure", s.handleSchedulerResourcePressure)
	mux.HandleFunc("/api/scheduler/policy", s.handleSchedulerPolicy)
	mux.HandleFunc("/api/workspaces", s.handleWorkspaces)
	mux.HandleFunc("/api/workspaces/", s.handleWorkspaceSubresource)
	mux.HandleFunc("/api/experiments/results", s.handleExperimentResults)
	mux.HandleFunc("/api/tasks", s.handleTasks)
	mux.HandleFunc("/api/tasks/", s.handleTaskSubresource)
	s.mux = mux
}

func streamEvents(w http.ResponseWriter, r *http.Request, hub *events.Hub) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ch, cancel := hub.Subscribe()
	defer cancel()
	flusher, _ := w.(http.Flusher)
	writeSSE(w, flusher, events.Event{
		ID:        "runtime-connected",
		Type:      "runtime.connected",
		Source:    "runtime",
		Timestamp: time.Now().UnixMilli(),
		Payload:   map[string]any{"status": "connected"},
	})
	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			writeSSE(w, flusher, event)
		}
	}
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, event events.Event) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
	if flusher != nil {
		flusher.Flush()
	}
}

func (s *Server) handleDemoRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	result, err := s.runDemo(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.mu.Lock()
	s.tasks[result.TaskID] = result
	s.mu.Unlock()
	writeJSON(w, http.StatusAccepted, map[string]string{"task_id": result.TaskID})
}

func (s *Server) handleDemoFault(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	kind := strings.TrimPrefix(r.URL.Path, "/api/demo/fault/")
	switch kind {
	case "tool-timeout", "timeout":
		record := s.runToolTimeoutFault(r.Context())
		writeJSON(w, http.StatusAccepted, record)
	case "rmrf":
		record := s.runWorkspaceRMFault()
		writeJSON(w, http.StatusAccepted, record)
	case "workspace-rmrf":
		evidence := s.runWorkspaceRMFaultEvidence()
		writeJSON(w, http.StatusAccepted, evidence)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleFaults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.supervisor.Records())
}

func (s *Server) runDemo(ctx context.Context) (demo.Result, error) {
	if s.registry == nil {
		result, err := s.demo.Run(ctx)
		if err != nil {
			return demo.Result{}, err
		}
		s.seedContext(result.TaskID, result.Agents)
		s.runDemoRuntimeEvidence(ctx, result)
		s.saveCheckpoint(result, 1)
		for _, event := range result.Events {
			if event.Type == "scheduler.selected" {
				event.Payload = s.withPressurePayload(event.Payload)
			}
			s.sink.Publish(event)
		}
		return result, nil
	}
	taskID := fmt.Sprintf("task-%d", time.Now().UnixNano())
	result := demo.Result{
		TaskID: taskID,
		Status: "running",
		DAG: []demo.DAGNode{
			{ID: "planner", Role: "Planner"},
			{ID: "coder", Role: "Coder", Dependencies: []string{"planner"}},
			{ID: "tester", Role: "Tester", Dependencies: []string{"coder"}},
			{ID: "reviewer", Role: "Reviewer", Dependencies: []string{"tester"}},
		},
	}
	s.sink.Publish(events.New("task.created", taskID, "", "runtime", map[string]any{"mode": "worker"}))
	specs := []worker.Spec{
		{AgentID: taskID + "-planner", Role: "Planner", TaskID: taskID},
		{AgentID: taskID + "-coder", Role: "Coder", TaskID: taskID},
		{AgentID: taskID + "-tester", Role: "Tester", TaskID: taskID},
		{AgentID: taskID + "-reviewer", Role: "Reviewer", TaskID: taskID},
	}
	for _, spec := range specs {
		agent := s.registry.CreateAgent(spec.AgentID, spec.Role, spec.TaskID)
		result.Agents = append(result.Agents, agentToDemo(agent))
	}
	s.seedContext(result.TaskID, result.Agents)
	s.runDemoRuntimeEvidence(ctx, result)
	s.saveCheckpoint(result, 1)
	pending := append([]worker.Spec(nil), specs...)
	for len(pending) > 0 {
		selected, decision, ok := s.scheduler.Select(taskID, s.schedulerCandidates(taskID, pending))
		if !ok {
			return demo.Result{}, fmt.Errorf("scheduler found no ready agents for %s", taskID)
		}
		s.publishSchedulerDecision(decision)
		spec, nextPending := popSpec(pending, selected.AgentID)
		pending = nextPending
		if _, err := s.launcher.Start(s.workerCtx, spec); err != nil {
			s.sink.Publish(events.New("agent.worker_start_failed", taskID, spec.AgentID, "runtime", map[string]any{"error": err.Error()}))
			return demo.Result{}, err
		}
	}
	return result, nil
}

func (s *Server) saveCheckpoint(result demo.Result, sequence int) {
	agents := make([]avp.AVP, 0, len(result.Agents))
	if s.registry != nil {
		agents = s.registry.ListByTask(result.TaskID)
	} else {
		for _, agent := range result.Agents {
			agents = append(agents, avp.AVP{
				AgentID:    agent.ID,
				TaskID:     result.TaskID,
				Role:       agent.Role,
				State:      avp.AgentState(agent.State),
				PID:        agent.PID,
				CgroupPath: agent.CgroupPath,
				LastSeen:   agent.LastSeen,
			})
		}
	}
	pageTables := make(map[string][]string, len(agents))
	vruntime := make(map[string]uint64, len(agents))
	for _, agent := range agents {
		pageTables[agent.AgentID] = s.cvm.PageTable(agent.AgentID).PageIDs
		vruntime[agent.AgentID] = agent.VRuntime
	}
	snapshot := checkpoint.Snapshot{
		TaskID:            result.TaskID,
		Sequence:          sequence,
		Agents:            agents,
		PageTables:        pageTables,
		SchedulerVRuntime: vruntime,
		Mode:              "runtime-state",
	}
	if err := s.checkpoint.Save(snapshot); err != nil {
		s.sink.Publish(events.New("checkpoint.failed", result.TaskID, "", "runtime", map[string]any{"error": err.Error()}))
		return
	}
	s.sink.Publish(events.New("checkpoint.created", result.TaskID, "", "runtime", map[string]any{
		"sequence":    sequence,
		"mode":        snapshot.Mode,
		"agent_count": len(snapshot.Agents),
	}))
}

func (s *Server) runDemoRuntimeEvidence(ctx context.Context, result demo.Result) {
	plannerID := agentIDForRole(result.Agents, "planner", "planner-1")
	reviewerID := agentIDForRole(result.Agents, "reviewer", result.TaskID+"-reviewer")
	fixerID := agentIDForRole(result.Agents, "fixer", result.TaskID+"-fixer")
	if s.registry != nil {
		if _, ok := s.registry.Get(reviewerID); !ok {
			s.registry.CreateAgent(reviewerID, "Reviewer", result.TaskID)
		}
	}
	reviewPage, err := s.cvm.WriteDelta(reviewerID, "Reviewer feedback: go test failed, spawn Fixer and pass review feedback by page reference.\n")
	if err != nil {
		return
	}
	s.syscalls.Handle(ctx, syscallgw.Request{
		RequestID: result.TaskID + "-llm-call",
		TaskID:    result.TaskID,
		AgentID:   plannerID,
		Name:      "llm.call",
		Args: map[string]any{
			"role": "planner",
		},
	})
	s.syscalls.Handle(ctx, syscallgw.Request{
		RequestID: result.TaskID + "-tool-exec",
		TaskID:    result.TaskID,
		AgentID:   plannerID,
		Name:      "tool.exec",
		Args: map[string]any{
			"command":    "pwd",
			"timeout_ms": 1000,
		},
	})
	s.syscalls.Handle(ctx, syscallgw.Request{
		RequestID: result.TaskID + "-ipc-publish",
		TaskID:    result.TaskID,
		AgentID:   reviewerID,
		Name:      "ipc.publish",
		Args: map[string]any{
			"topic":      "review.feedback",
			"page_id":    reviewPage.ID,
			"size_bytes": reviewPage.Bytes,
		},
	})
	s.syscalls.Handle(ctx, syscallgw.Request{
		RequestID: result.TaskID + "-agent-spawn",
		TaskID:    result.TaskID,
		AgentID:   reviewerID,
		Name:      "agent.spawn",
		Args: map[string]any{
			"agent_id":     fixerID,
			"role":         "fixer",
			"reason":       "tester reported a failing go test",
			"dependencies": []any{reviewerID},
		},
	})
	s.syscalls.Handle(ctx, syscallgw.Request{
		RequestID: result.TaskID + "-ipc-poll",
		TaskID:    result.TaskID,
		AgentID:   fixerID,
		Name:      "ipc.poll",
		Args: map[string]any{
			"topic": "review.feedback",
		},
	})
}

func agentIDForRole(agents []demo.Agent, roleNeedle, fallback string) string {
	for _, agent := range agents {
		if strings.Contains(strings.ToLower(agent.Role), roleNeedle) {
			return agent.ID
		}
	}
	return fallback
}

func newLLMRouter() *llm.Router {
	router := llm.NewRouter()
	router.Register("mock", llm.NewMockProvider("mock"))
	router.Register("deepseek", llm.NewDeepSeekProvider(llm.DeepSeekConfig{
		APIKey:  os.Getenv("DEEPSEEK_API_KEY"),
		BaseURL: os.Getenv("DEEPSEEK_BASE_URL"),
		Model:   os.Getenv("DEEPSEEK_MODEL"),
	}))
	provider := os.Getenv("AORT_LLM_PROVIDER")
	if provider == "" {
		provider = "mock"
	}
	fallback := os.Getenv("AORT_LLM_FALLBACK_PROVIDER")
	if fallback == "" {
		fallback = "mock"
	}
	router.SetDefault(provider)
	router.SetFallback(fallback)
	return router
}

func (s *Server) runToolTimeoutFault(ctx context.Context) supervisor.Record {
	taskID := fmt.Sprintf("fault-%d", time.Now().UnixNano())
	agentID := taskID + "-agent"
	response := s.syscalls.Handle(ctx, syscallgw.Request{
		RequestID: taskID + "-tool-timeout",
		TaskID:    taskID,
		AgentID:   agentID,
		Name:      "tool.exec",
		Args: map[string]any{
			"command":    "sleep",
			"args":       []any{"1"},
			"timeout_ms": 10,
		},
	})
	return s.supervisor.Record(supervisor.Fault{
		Type:           supervisor.FaultToolTimeout,
		TaskID:         taskID,
		AgentID:        agentID,
		RecoveryAction: "tool process killed by timeout context",
		Details: map[string]any{
			"syscall":        "tool.exec",
			"syscall_status": response.Status,
			"error":          response.Error,
		},
	})
}

func (s *Server) runWorkspaceRMFault() supervisor.Record {
	taskID := fmt.Sprintf("fault-%d", time.Now().UnixNano())
	agentID := taskID + "-agent"
	_, snapshotErr := s.workspace.CreateBaseSnapshot(taskID, map[string]string{
		"README.md":      "# AORT-R fault fixture\n",
		"src/service.go": "package src\n\nfunc Stable() string { return \"base\" }\n",
	})
	result, rollbackErr := s.workspace.InjectRMAndRollback(taskID, agentID)
	details := map[string]any{
		"workspace_mode":   result.Mode,
		"rollback_success": result.RollbackSuccess,
		"base_intact":      result.BaseIntact,
		"removed_entries":  result.RemovedEntries,
		"workspace_path":   result.WorkspacePath,
		"base_path":        result.BasePath,
	}
	var recoveryAction string
	if snapshotErr != nil {
		details["snapshot_error"] = snapshotErr.Error()
	} else if rollbackErr != nil {
		details["rollback_error"] = rollbackErr.Error()
	} else {
		recoveryAction = "agent workspace removed and restored from base snapshot"
	}
	return s.supervisor.Record(supervisor.Fault{
		Type:           supervisor.FaultWorkspaceRollback,
		TaskID:         taskID,
		AgentID:        agentID,
		RecoveryAction: recoveryAction,
		Details:        details,
	})
}

func (s *Server) runWorkspaceRMFaultEvidence() workspace.RMFaultEvidence {
	result, err := workspace.RunRMFaultDemo(workspace.Config{Root: workspace.DefaultRoot(), Sink: s.sink})
	if err != nil {
		result.Success = false
		result.Error = err.Error()
	}
	recoveryAction := "workspace_rmrf isolated to target workspace and rolled back"
	if !result.Success {
		recoveryAction = "workspace_rmrf evidence generated with errors"
	}
	_ = s.supervisor.Record(supervisor.Fault{
		Type:           supervisor.FaultWorkspaceRollback,
		TaskID:         "workspace-rmrf",
		AgentID:        result.TargetAgent,
		RecoveryAction: recoveryAction,
		Details: map[string]any{
			"success":               result.Success,
			"fault_type":            result.FaultType,
			"mode":                  result.Mode,
			"evidence_mode":         result.EvidenceMode,
			"fallback_reason":       result.FallbackReason,
			"lowerdir_unchanged":    result.LowerDirUnchanged,
			"target_agent_affected": result.TargetAgentAffected,
			"rollback_success":      result.RollbackSuccess,
			"cascade_failure":       result.CascadeFailure,
			"destroy_success":       result.DestroySuccess,
		},
	})
	return result
}

func (s *Server) seedContext(taskID string, agents []demo.Agent) {
	system, _ := s.cvm.CreatePage(cvm.KindSystem, "You are an AORT-R software engineering agent.\n")
	project, _ := s.cvm.CreatePage(cvm.KindProject, "Project: implement a Todo Web API with create, list, and complete operations.\n")
	task, _ := s.cvm.CreatePage(cvm.KindTask, "Task: produce code, tests, review feedback, and fixes through runtime syscalls.\n")
	for _, agent := range agents {
		_ = s.cvm.MountPage(agent.ID, system.ID)
		_ = s.cvm.MountPage(agent.ID, project.ID)
		_ = s.cvm.MountPage(agent.ID, task.ID)
		_, _ = s.cvm.WriteDelta(agent.ID, "Agent "+agent.Role+" private scratch for "+taskID+".\n")
	}
}

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.RLock()
	tasks := make([]demo.Result, 0, len(s.tasks))
	for _, task := range s.tasks {
		task.Agents = s.agentsForTask(task)
		tasks = append(tasks, task)
	}
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, tasks)
}

func (s *Server) handleAgents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.registry != nil {
		writeJSON(w, http.StatusOK, s.enrichedAgents(s.registry.List()))
		return
	}
	s.mu.RLock()
	agents := make([]demo.Agent, 0)
	for _, task := range s.tasks {
		agents = append(agents, task.Agents...)
	}
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, agents)
}

func (s *Server) handleAgentAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/api/agents/")
	agentID, action, ok := strings.Cut(rest, "/")
	if !ok || agentID == "" {
		http.NotFound(w, r)
		return
	}
	if s.capsules == nil {
		http.Error(w, "capsule runtime is not enabled", http.StatusServiceUnavailable)
		return
	}
	var err error
	switch action {
	case "freeze":
		err = s.capsules.Freeze(agentID)
	case "unfreeze":
		err = s.capsules.Unfreeze(agentID)
	case "kill":
		err = s.capsules.Kill(agentID)
		if err == nil && s.registry != nil {
			s.registry.SetState(agentID, avp.StateKilled)
		}
	default:
		http.NotFound(w, r)
		return
	}
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"status": "error", "error": err.Error()})
		return
	}
	if s.registry != nil {
		if agent, ok := s.registry.Get(agentID); ok {
			s.sink.Publish(events.New("agent."+action, agent.TaskID, agent.AgentID, "runtime", map[string]any{"action": action}))
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "action": action})
}

type capsuleEvidence struct {
	AgentID       string            `json:"agent_id"`
	TaskID        string            `json:"task_id,omitempty"`
	Role          string            `json:"role,omitempty"`
	PID           int               `json:"pid"`
	EvidenceMode  string            `json:"evidence_mode"`
	RealCgroupV2  bool              `json:"real_cgroup_v2"`
	CapsuleMode   string            `json:"capsule_mode"`
	CgroupPath    string            `json:"cgroup_path"`
	MemoryCurrent int64             `json:"memory_current"`
	PidsCurrent   int64             `json:"pids_current"`
	CPUStat       map[string]uint64 `json:"cpu_stat"`
	Events        map[string]uint64 `json:"events"`
	Frozen        bool              `json:"frozen"`
	Error         string            `json:"error,omitempty"`
}

func (s *Server) handleCapsules(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/capsules" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.registry == nil || s.capsules == nil {
		http.Error(w, "capsule runtime is not enabled", http.StatusServiceUnavailable)
		return
	}
	agents := s.registry.List()
	out := make([]capsuleEvidence, 0, len(agents))
	for _, agent := range agents {
		out = append(out, s.capsuleEvidence(agent))
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleCapsuleSubresource(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/capsules/")
	agentID, action, hasAction := strings.Cut(rest, "/")
	if agentID == "" {
		http.NotFound(w, r)
		return
	}
	if !hasAction {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if s.registry == nil || s.capsules == nil {
			http.Error(w, "capsule runtime is not enabled", http.StatusServiceUnavailable)
			return
		}
		agent, ok := s.registry.Get(agentID)
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, s.capsuleEvidence(agent))
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.handleCapsuleAction(w, agentID, action)
}

func (s *Server) handleCapsuleAction(w http.ResponseWriter, agentID, action string) {
	if s.capsules == nil {
		http.Error(w, "capsule runtime is not enabled", http.StatusServiceUnavailable)
		return
	}
	var err error
	switch action {
	case "freeze":
		err = s.capsules.Freeze(agentID)
	case "unfreeze":
		err = s.capsules.Unfreeze(agentID)
	case "kill":
		err = s.capsules.Kill(agentID)
		if err == nil && s.registry != nil {
			s.registry.SetState(agentID, avp.StateKilled)
		}
	default:
		http.NotFound(w, nil)
		return
	}
	if err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"status": "error", "error": err.Error()})
		return
	}
	if s.registry != nil {
		if agent, ok := s.registry.Get(agentID); ok {
			s.sink.Publish(events.New("capsule."+action, agent.TaskID, agent.AgentID, "runtime", map[string]any{"action": action}))
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "action": action})
}

func (s *Server) capsuleEvidence(agent avp.AVP) capsuleEvidence {
	stats := capsule.Stats{Mode: capsule.ModeDegraded, Error: "capsule runtime is not enabled"}
	runtimePath := agent.CgroupPath
	if s.capsules != nil {
		stats = s.capsules.Stats(agent.AgentID)
		if runtime, ok := s.capsules.Runtime(agent.AgentID); ok {
			runtimePath = runtime.CgroupPath
		}
	}
	mode := stats.Mode
	if mode == "" {
		mode = agent.CapsuleMode
	}
	evidenceMode := "degraded"
	if mode == capsule.ModeReal {
		evidenceMode = "real-cgroup-v2"
	}
	return capsuleEvidence{
		AgentID:       agent.AgentID,
		TaskID:        agent.TaskID,
		Role:          agent.Role,
		PID:           agent.PID,
		EvidenceMode:  evidenceMode,
		RealCgroupV2:  mode == capsule.ModeReal,
		CapsuleMode:   mode,
		CgroupPath:    runtimePath,
		MemoryCurrent: stats.MemoryCurrent,
		PidsCurrent:   stats.PidsCurrent,
		CPUStat:       stats.CPUStat,
		Events:        stats.Events,
		Frozen:        stats.Frozen,
		Error:         stats.Error,
	}
}

type evidenceModule struct {
	Name         string        `json:"name"`
	Status       string        `json:"status"`
	Mode         string        `json:"mode"`
	EvidenceMode evidence.Mode `json:"evidence_mode"`
	Endpoint     string        `json:"endpoint,omitempty"`
	Signals      []string      `json:"signals,omitempty"`
	Reason       string        `json:"reason,omitempty"`
}

type evidenceResponse struct {
	UpdatedAt int64            `json:"updated_at"`
	Modules   []evidenceModule `json:"modules"`
}

func (s *Server) handleEvidence(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.evidenceReport())
}

func (s *Server) evidenceReport() evidenceResponse {
	kernelStatus := s.kernel.Status()
	pressureStatus := s.pressure.Sample()
	modules := []evidenceModule{
		s.cgroupEvidenceModule(),
		s.workerEvidenceModule(),
		{
			Name:         "Syscall Gateway",
			Status:       "real",
			Mode:         "uds-json-rpc",
			EvidenceMode: evidence.ModeRealRuntime,
			Endpoint:     "/api/syscalls",
			Signals:      []string{"context.materialize", "llm.call", "tool.exec", "ipc.publish", "agent.spawn"},
		},
		{
			Name:         "Scheduler",
			Status:       "real",
			Mode:         s.scheduler.Policy(),
			EvidenceMode: evidence.ModeRealRuntime,
			Endpoint:     "/api/scheduler/decisions",
			Signals:      []string{"decision_log", "vruntime", "prefix_affinity", "pressure_payload"},
		},
		{
			Name:         "CVM",
			Status:       "real",
			Mode:         "real-partial",
			EvidenceMode: evidence.ModeRealPartial,
			Endpoint:     "/api/context/stats",
			Signals:      []string{"page_store", "page_table", "saved_bytes", "saved_tokens"},
			Reason:       "KV cache is represented as context pages and reuse stats, not a model-server KV memory map.",
		},
		{
			Name:         "Page-reference IPC",
			Status:       "real",
			Mode:         "real-partial",
			EvidenceMode: evidence.ModeRealPartial,
			Endpoint:     "/api/ipc/metrics",
			Signals:      []string{"page_id_publish", "subscriber_poll", "avoided_copy_bytes"},
			Reason:       "IPC passes page references inside the runtime; cross-host transport is not implemented.",
		},
		{
			Name:         "Workspace Isolation",
			Status:       "degraded",
			Mode:         "degraded-copy",
			EvidenceMode: evidence.ModeDegradedCopy,
			Endpoint:     "/api/demo/fault/workspace-rmrf",
			Signals:      []string{"lowerdir", "upperdir", "merged", "rollback_success", "destroy_success"},
			Reason:       "overlayfs mount is attempted on Linux/root; unsupported hosts use degraded-copy fallback.",
		},
		{
			Name:         "Kernel Observer",
			Status:       "degraded",
			Mode:         kernelStatus.Mode,
			EvidenceMode: evidence.ModeDegraded,
			Endpoint:     "/api/kernel/status",
			Signals:      []string{kernelStatus.Probe, "kernel.exec"},
			Reason:       kernelStatus.Reason,
		},
		{
			Name:         "PSI Monitor",
			Status:       pressureEvidenceStatus(pressureStatus),
			Mode:         pressureStatus.Mode,
			EvidenceMode: pressureEvidenceMode(pressureStatus),
			Endpoint:     "/api/pressure/status",
			Signals:      []string{"cpu.some", "memory.some", "io.some", "pressure_throttle"},
			Reason:       pressureStatus.Reason,
		},
		{
			Name:         "eBPF Observer",
			Status:       "planned",
			Mode:         "planned",
			EvidenceMode: evidence.ModePlanned,
			Signals:      []string{"sched_process_exec"},
			Reason:       "Not implemented in this build; kernel observer reports degraded mode with syscall-gateway-proxy as the probe label.",
		},
		{
			Name:         "Software Real Demo",
			Status:       "real-runtime",
			Mode:         "software-real",
			EvidenceMode: evidence.ModeRealRuntime,
			Endpoint:     "/api/demo/software-real/run",
			Signals: []string{
				"POST /api/demo/software-real/run",
				"GET /api/demo/software-real/status",
				"GET /api/demo/software-real/result",
				"agent.spawn",
				"scheduler decision",
				"context.materialize",
				"context.write_delta",
				"llm.call",
				"tool.exec",
				"ipc.publish",
				"ipc.poll",
				"agent.report",
				"checkpoint",
				"test failure recovery",
			},
			Reason: "Planner -> Coder -> Tester -> Reviewer -> Fixer -> Reporter demo runs through Runtime syscalls, scheduler, CVM, IPC, checkpoint, and recovery paths.",
		},
		{
			Name:         "LLM Provider",
			Status:       llmEvidenceStatus(),
			Mode:         llmEvidenceMode(),
			EvidenceMode: evidence.Mode(llmEvidenceMode()),
			Signals:      []string{"llm.call", "provider", "model", "duration_ms", "tokens", "fallback", "evidence_mode"},
			Reason:       "DeepSeek provider is environment-backed with mock fallback; API keys are read only from environment variables.",
		},
	}
	return evidenceResponse{UpdatedAt: time.Now().UnixMilli(), Modules: modules}
}

func llmEvidenceStatus() string {
	if os.Getenv("AORT_LLM_PROVIDER") == "deepseek" && os.Getenv("DEEPSEEK_API_KEY") != "" {
		return "real-api"
	}
	return "mock"
}

func llmEvidenceMode() string {
	if os.Getenv("AORT_LLM_PROVIDER") == "deepseek" {
		if os.Getenv("DEEPSEEK_API_KEY") != "" {
			return "real-api"
		}
		return "mock"
	}
	return "mock"
}

func (s *Server) cgroupEvidenceModule() evidenceModule {
	module := evidenceModule{
		Name:         "Cgroup Capsule",
		Status:       "degraded",
		Mode:         "degraded",
		EvidenceMode: evidence.ModeDegraded,
		Endpoint:     "/api/capsules",
		Signals:      []string{"capsule_mode", "real_cgroup_v2", "memory.current", "pids.current", "cpu.stat", "cgroup.events"},
		Reason:       "No real cgroup v2 capsule has been observed in this runtime process yet.",
	}
	if s.registry == nil || s.capsules == nil {
		module.Reason = "Worker runtime or capsule manager is not enabled."
		return module
	}
	for _, agent := range s.registry.List() {
		capsuleEvidence := s.capsuleEvidence(agent)
		if capsuleEvidence.RealCgroupV2 && capsuleEvidence.CapsuleMode == capsule.ModeReal {
			module.Status = "real"
			module.Mode = "cgroup-v2"
			module.EvidenceMode = evidence.ModeRealCgroupV2
			module.Reason = "At least one Agent is attached to a real cgroup v2 capsule."
			return module
		}
	}
	return module
}

func (s *Server) workerEvidenceModule() evidenceModule {
	module := evidenceModule{
		Name:         "Worker Process",
		Status:       "degraded",
		Mode:         "worker-registry",
		EvidenceMode: evidence.ModeDegraded,
		Endpoint:     "/api/agents",
		Signals:      []string{"worker_pid", "uds_register", "heartbeat"},
		Reason:       "No registered worker PID has been observed yet.",
	}
	if s.registry == nil {
		module.Reason = "Worker runtime is not enabled."
		return module
	}
	for _, agent := range s.registry.List() {
		if agent.PID > 0 {
			module.Status = "real"
			module.Mode = "process"
			module.EvidenceMode = evidence.ModeRealRuntime
			module.Reason = "At least one Agent has a real worker PID."
			return module
		}
	}
	return module
}

func pressureEvidenceStatus(status pressure.Status) string {
	if status.Mode == pressure.ModePSI && !status.Degraded {
		return "real"
	}
	if status.Degraded {
		return "unavailable"
	}
	return "degraded"
}

func pressureEvidenceMode(status pressure.Status) evidence.Mode {
	if status.Mode == pressure.ModePSI && !status.Degraded {
		return evidence.ModeRealCgroupV2
	}
	return evidence.ModeDegraded
}

func (s *Server) handleContextPages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.cvm.Pages())
}

func (s *Server) handleContextStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.cvm.Stats())
}

func (s *Server) handleContextAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/api/context/agents/")
	agentID, subresource, ok := strings.Cut(rest, "/")
	if !ok || subresource != "pagetable" {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, s.cvm.PageTable(agentID))
}

func (s *Server) handleIPCMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.ipc.Metrics())
}

func (s *Server) handleIPCTopics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.ipc.Topics())
}

func (s *Server) handleKernelStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.kernel.Status())
}

func (s *Server) handleKernelEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.kernel.Events())
}

func (s *Server) handlePressureStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	status := s.pressure.Sample()
	s.sink.Publish(events.New("pressure.sampled", "", "", "runtime", pressurePayload(status)))
	writeJSON(w, http.StatusOK, status)
}

func (s *Server) handleCheckpoints(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	snapshots, err := s.checkpoint.List()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, snapshots)
}

func (s *Server) handleRecoveryStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.recovery)
}

func (s *Server) handleSyscalls(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.syscalls.Records())
}

func (s *Server) handleSchedulerDecisions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.scheduler.Decisions())
}

func (s *Server) handleSchedulerPolicies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success":         true,
		"policies":        scheduler.Policies(),
		"current_policy":  s.scheduler.Policy(),
		"evidence_mode":   evidence.ModeRealRuntime,
		"fallback_reason": "",
	})
}

func (s *Server) handleSchedulerResourcePressure(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	pressure := s.sampleSchedulerResourcePressure()
	writeJSON(w, http.StatusOK, map[string]any{
		"success":               true,
		"memory_pressure":       pressure.MemoryPressure,
		"pids_pressure":         pressure.PidsPressure,
		"cpu_throttle_pressure": pressure.CPUThrottlePressure,
		"psi_pressure":          pressure.PSIPressure,
		"evidence_mode":         pressure.EvidenceMode,
		"fallback_reason":       pressure.FallbackReason,
	})
}

func (s *Server) handleSchedulerPolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Policy string `json:"policy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := s.scheduler.SetPolicy(body.Policy); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"policy": s.scheduler.Policy()})
}

func (s *Server) handleWorkspaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"success":         true,
		"mode":            "workspace-manager",
		"evidence_mode":   evidence.ModeRealRuntime,
		"fallback_reason": "",
		"workspaces":      s.workspace.List(),
	})
}

func (s *Server) handleWorkspaceSubresource(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/workspaces/")
	agentID, action, hasAction := strings.Cut(rest, "/")
	if agentID == "" {
		http.NotFound(w, r)
		return
	}
	if !hasAction {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		status, err := s.workspace.Status(agentID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, status)
			return
		}
		writeJSON(w, http.StatusOK, status)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	switch action {
	case "rollback":
		result, err := s.workspace.Rollback(agentID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, workspaceActionResponse(false, "", "", err))
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"success":         result.RollbackSuccess,
			"mode":            result.Mode,
			"evidence_mode":   result.EvidenceMode,
			"fallback_reason": result.FallbackReason,
			"result":          result,
		})
	case "commit":
		err := s.workspace.Commit(agentID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, workspaceActionResponse(false, "", "", err))
			return
		}
		status, _ := s.workspace.Status(agentID)
		writeJSON(w, http.StatusOK, workspaceActionResponse(true, status.Mode, string(status.EvidenceMode), nil))
	case "destroy":
		status, _ := s.workspace.Status(agentID)
		err := s.workspace.Destroy(agentID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, workspaceActionResponse(false, status.Mode, string(status.EvidenceMode), err))
			return
		}
		writeJSON(w, http.StatusOK, workspaceActionResponse(true, status.Mode, string(status.EvidenceMode), nil))
	default:
		http.NotFound(w, r)
	}
}

func workspaceActionResponse(success bool, mode, evidenceMode string, err error) map[string]any {
	if evidenceMode == "" {
		evidenceMode = string(evidence.ModeMissing)
	}
	response := map[string]any{
		"success":         success,
		"mode":            mode,
		"evidence_mode":   evidenceMode,
		"fallback_reason": "",
	}
	if err != nil {
		response["error"] = err.Error()
	}
	return response
}

func (s *Server) handleExperimentResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, loadExperimentResults())
}

func (s *Server) handleTaskSubresource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	rest := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	taskID, subresource, ok := strings.Cut(rest, "/")
	if !ok || subresource != "dag" {
		http.NotFound(w, r)
		return
	}
	s.mu.RLock()
	task, found := s.tasks[taskID]
	s.mu.RUnlock()
	if !found {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, task.DAG)
}

type experimentResultsResponse struct {
	E1Scheduler    []experiment.E1SchedulerResult     `json:"e1_scheduler"`
	E2Fault        []experiment.E2FaultResult         `json:"e2_fault"`
	E3Context      experiment.E3ContextResult         `json:"e3_context"`
	E1RealSchedule []experiment.E1RealSchedulerResult `json:"e1_real_scheduler"`
	E2RealFault    []experiment.E2RealFaultResult     `json:"e2_real_fault"`
	E3RealContext  []experiment.E3RealContextResult   `json:"e3_real_context"`
	E4RealIPC      []experiment.E4RealIPCResult       `json:"e4_real_ipc"`
	E5EndToEnd     experiment.E5EndToEndResult        `json:"e5_end_to_end"`
}

func loadExperimentResults() experimentResultsResponse {
	base := filepath.Join("experiments", "results")
	response := experimentResultsResponse{}
	if !readJSON(filepath.Join(base, "e1-scheduler.json"), &response.E1Scheduler) {
		response.E1Scheduler = experiment.RunLegacyE1Scheduler(5)
	}
	if !readJSON(filepath.Join(base, "e2-fault.json"), &response.E2Fault) {
		response.E2Fault = experiment.RunLegacyE2FaultIsolation(5)
	}
	if !readJSON(filepath.Join(base, "e3-context.json"), &response.E3Context) {
		response.E3Context = experiment.RunE3ContextSharing(5)
	}
	if !readJSON(filepath.Join(base, "e1-real-scheduler.json"), &response.E1RealSchedule) {
		response.E1RealSchedule = experiment.RunE1RealSchedulerBenchmark(5)
	}
	if !readJSON(filepath.Join(base, "e2-real-fault.json"), &response.E2RealFault) {
		response.E2RealFault = experiment.RunE2RealFaultIsolation(5)
	}
	if !readJSON(filepath.Join(base, "e3-real-context.json"), &response.E3RealContext) {
		response.E3RealContext = experiment.RunE3RealContextReuse(5)
	}
	if !readJSON(filepath.Join(base, "e4-real-ipc.json"), &response.E4RealIPC) {
		response.E4RealIPC = experiment.RunE4RealIPCBenchmark(5)
	}
	if !readJSON(filepath.Join(base, "e5-end-to-end.json"), &response.E5EndToEnd) {
		response.E5EndToEnd = experiment.RunE5EndToEndBenchmark(5)
	}
	return response
}

func readJSON(path string, target any) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return json.Unmarshal(data, target) == nil
}

func (s *Server) schedulerCandidates(taskID string, pending []worker.Spec) []avp.AVP {
	candidates := make([]avp.AVP, 0, len(pending))
	for _, spec := range pending {
		agent, ok := s.registry.Get(spec.AgentID)
		if !ok {
			continue
		}
		agent.State = avp.StateReady
		agent.PageTable = s.cvm.PageTable(spec.AgentID).PageIDs
		candidates = append(candidates, agent)
	}
	return candidates
}

func (s *Server) publishSchedulerDecision(decision scheduler.DecisionLog) {
	pressure := s.sampleSchedulerResourcePressure()
	status := s.pressure.Sample()
	payload := map[string]any{
		"policy":                decision.Policy,
		"decision_id":           decision.ID,
		"decision_reason":       decision.Reason,
		"candidates":            decision.Candidates,
		"shared_pages":          decision.SharedPages,
		"vruntime_before":       decision.VRuntimeBefore,
		"pressure_mode":         status.Mode,
		"pressure_degraded":     status.Degraded,
		"pressure_throttle":     status.Throttle,
		"pressure_reason":       status.ThrottleReason,
		"pressure_cpu_avg10":    status.CPU.Some.Avg10,
		"pressure_memory_avg10": status.Memory.Some.Avg10,
		"pressure_io_avg10":     status.IO.Some.Avg10,
		"memory_pressure":       pressure.MemoryPressure,
		"pids_pressure":         pressure.PidsPressure,
		"cpu_throttle_pressure": pressure.CPUThrottlePressure,
		"psi_pressure":          pressure.PSIPressure,
		"evidence_mode":         pressure.EvidenceMode,
		"fallback_reason":       pressure.FallbackReason,
	}
	s.sink.Publish(events.New("pressure.sampled", decision.TaskID, decision.SelectedAgent, "runtime", pressurePayload(status)))
	if status.Throttle {
		s.sink.Publish(events.New("scheduler.pressure_throttle", decision.TaskID, decision.SelectedAgent, "scheduler", payload))
	}
	s.sink.Publish(events.New("scheduler.selected", decision.TaskID, decision.SelectedAgent, "scheduler", payload))
}

func (s *Server) sampleSchedulerResourcePressure() scheduler.ResourcePressure {
	status := s.pressure.Sample()
	psiPressure := status.CPU.Some.Avg10
	if status.Memory.Some.Avg10 > psiPressure {
		psiPressure = status.Memory.Some.Avg10
	}
	if status.IO.Some.Avg10 > psiPressure {
		psiPressure = status.IO.Some.Avg10
	}
	mode := evidence.ModeRealCgroupV2
	reason := ""
	if status.Degraded {
		mode = evidence.ModeDegraded
		reason = status.Reason
		if reason == "" {
			reason = "PSI pressure files unavailable"
		}
	}
	if reason == "" {
		reason = ""
	}
	pressure := scheduler.ResourcePressure{
		PSIPressure:    psiPressure / 100,
		EvidenceMode:   mode,
		FallbackReason: reason,
	}
	s.scheduler.SetResourcePressure(pressure)
	return s.scheduler.ResourcePressure()
}

func pressurePayload(status pressure.Status) map[string]any {
	return map[string]any{
		"mode":            status.Mode,
		"degraded":        status.Degraded,
		"reason":          status.Reason,
		"throttle":        status.Throttle,
		"throttle_reason": status.ThrottleReason,
		"cpu_avg10":       status.CPU.Some.Avg10,
		"memory_avg10":    status.Memory.Some.Avg10,
		"io_avg10":        status.IO.Some.Avg10,
		"sampled_at":      status.SampledAt,
	}
}

func (s *Server) withPressurePayload(payload map[string]any) map[string]any {
	if payload == nil {
		payload = map[string]any{}
	}
	status := s.pressure.Sample()
	for key, value := range map[string]any{
		"pressure_mode":         status.Mode,
		"pressure_degraded":     status.Degraded,
		"pressure_throttle":     status.Throttle,
		"pressure_reason":       status.ThrottleReason,
		"pressure_cpu_avg10":    status.CPU.Some.Avg10,
		"pressure_memory_avg10": status.Memory.Some.Avg10,
		"pressure_io_avg10":     status.IO.Some.Avg10,
	} {
		payload[key] = value
	}
	s.sink.Publish(events.New("pressure.sampled", "", "", "runtime", pressurePayload(status)))
	return payload
}

func popSpec(specs []worker.Spec, agentID string) (worker.Spec, []worker.Spec) {
	for index, spec := range specs {
		if spec.AgentID == agentID {
			next := append([]worker.Spec(nil), specs[:index]...)
			next = append(next, specs[index+1:]...)
			return spec, next
		}
	}
	return specs[0], specs[1:]
}

func (s *Server) recoverFromCheckpoints() {
	report, err := s.checkpoint.RecoverAll()
	if err != nil {
		s.recovery = checkpoint.RecoveryReport{
			Mode:           "checkpoint-light",
			Degraded:       true,
			Reason:         err.Error(),
			RecoveredAt:    time.Now().UnixMilli(),
			RecoveredTasks: []checkpoint.RecoveredTask{},
		}
		s.sink.Publish(events.New("runtime.recovery_failed", "", "", "runtime", map[string]any{"error": err.Error()}))
		return
	}
	s.recovery = report
	if report.TaskCount == 0 {
		return
	}
	snapshots, err := s.checkpoint.List()
	if err != nil {
		s.sink.Publish(events.New("runtime.recovery_failed", "", "", "runtime", map[string]any{"error": err.Error()}))
		return
	}
	latest := latestSnapshotsByTask(snapshots)
	s.mu.Lock()
	for _, recovered := range report.RecoveredTasks {
		if snapshot, ok := latest[recovered.TaskID]; ok {
			s.tasks[recovered.TaskID] = taskFromSnapshot(snapshot, recovered.Status)
			s.recoverRegistryAgents(snapshot)
		}
	}
	s.mu.Unlock()
	for _, recovered := range report.RecoveredTasks {
		s.sink.Publish(events.New("checkpoint.recovered", recovered.TaskID, "", "runtime", map[string]any{
			"sequence":         recovered.Sequence,
			"status":           recovered.Status,
			"agent_count":      recovered.AgentCount,
			"ready_agents":     recovered.ReadyAgents,
			"completed_agents": recovered.CompletedAgents,
			"page_table_refs":  recovered.PageTableRefs,
			"mode":             report.Mode,
			"degraded":         report.Degraded,
		}))
	}
	s.sink.Publish(events.New("runtime.recovered", "", "", "runtime", map[string]any{
		"mode":       report.Mode,
		"degraded":   report.Degraded,
		"task_count": report.TaskCount,
		"reason":     report.Reason,
	}))
}

func latestSnapshotsByTask(snapshots []checkpoint.Snapshot) map[string]checkpoint.Snapshot {
	latest := make(map[string]checkpoint.Snapshot)
	for _, snapshot := range snapshots {
		if snapshot.TaskID == "" {
			continue
		}
		latest[snapshot.TaskID] = snapshot
	}
	return latest
}

func taskFromSnapshot(snapshot checkpoint.Snapshot, status string) demo.Result {
	if status == "" {
		status = "recovered"
	}
	result := demo.Result{
		TaskID: snapshot.TaskID,
		Status: status,
		Agents: make([]demo.Agent, 0, len(snapshot.Agents)),
		DAG:    make([]demo.DAGNode, 0, len(snapshot.Agents)),
		Events: []events.Event{
			events.New("checkpoint.recovered", snapshot.TaskID, "", "runtime", map[string]any{
				"sequence": snapshot.Sequence,
				"mode":     snapshot.Mode,
			}),
		},
	}
	for _, agent := range snapshot.Agents {
		state := agent.State
		if !isRecoveredTerminal(state) {
			state = avp.StateReady
		}
		result.Agents = append(result.Agents, demo.Agent{
			ID:                  agent.AgentID,
			Role:                agent.Role,
			State:               string(state),
			PID:                 agent.PID,
			LastSeen:            agent.LastSeen,
			CapsuleMode:         agent.CapsuleMode,
			CapsuleEvidenceMode: capsuleEvidenceMode(agent.CapsuleMode, agent.CgroupPath),
			CgroupPath:          agent.CgroupPath,
		})
		result.DAG = append(result.DAG, demo.DAGNode{
			ID:           agent.AgentID,
			Role:         agent.Role,
			Dependencies: append([]string(nil), agent.Dependencies...),
		})
	}
	return result
}

func (s *Server) recoverRegistryAgents(snapshot checkpoint.Snapshot) {
	if s.registry == nil {
		return
	}
	for _, agent := range snapshot.Agents {
		restored := agent
		state := agent.State
		if !isRecoveredTerminal(state) {
			state = avp.StateReady
		}
		restored.State = state
		s.registry.RestoreAgent(restored)
	}
}

func isRecoveredTerminal(state avp.AgentState) bool {
	return state == avp.StateCompleted || state == avp.StateFailed || state == avp.StateKilled
}

func (s *Server) startWorkerRuntime() {
	if s.cfg.SocketPath == "" || s.cfg.WorkerCommand == "" {
		return
	}
	s.registry = worker.NewRegistry(s.sink)
	s.capsules = capsule.NewManager(capsule.Config{
		Root:          s.cfg.CgroupRoot,
		AllowDegraded: true,
	})
	s.registry.SetOnRegister(func(agent avp.AVP) {
		runtime, err := s.capsules.Prepare(agent.AgentID, agent.PID)
		if err != nil {
			s.sink.Publish(events.New("agent.capsule_failed", agent.TaskID, agent.AgentID, "runtime", map[string]any{"error": err.Error()}))
			return
		}
		s.registry.SetCapsule(agent.AgentID, runtime.CgroupPath, runtime.Mode)
		if ws, err := s.ensureAgentWorkspace(agent.TaskID, agent.AgentID); err == nil {
			s.registry.SetWorkspace(agent.AgentID, ws.MergedDir)
		}
	})
	s.launcher = worker.Launcher{Command: s.cfg.WorkerCommand, SocketPath: s.cfg.SocketPath}
	if s.cfg.HeartbeatTimeoutMS <= 0 {
		s.cfg.HeartbeatTimeoutMS = 6000
	}
	s.heartbeatTimeout = time.Duration(s.cfg.HeartbeatTimeoutMS) * time.Millisecond
	s.uds = worker.NewUDSServer(s.cfg.SocketPath, s.registry, workerSyscallAdapter{gateway: s.syscalls})
	if err := s.uds.Start(); err != nil {
		s.sink.Publish(events.New("runtime.degraded", "", "", "runtime", map[string]any{
			"component": "uds",
			"error":     err.Error(),
		}))
		s.registry = nil
		return
	}
	go s.monitorHeartbeats()
}

func (s *Server) handleAgentReport(report syscallgw.Report) {
	if s.registry == nil {
		return
	}
	s.registry.HandleMessage(worker.Message{
		Type:    worker.MessageReport,
		AgentID: report.AgentID,
		TaskID:  report.TaskID,
		Status:  report.Status,
		Payload: report.Payload,
	})
}

func (s *Server) observeKernelExec(observation syscallgw.ExecObservation) {
	s.kernel.ObserveExec(kernel.ExecObservation{
		TaskID:    observation.TaskID,
		AgentID:   observation.AgentID,
		PID:       observation.PID,
		Command:   observation.Command,
		Args:      observation.Args,
		Workspace: observation.Workspace,
		Status:    observation.Status,
	})
}

func (s *Server) spawnAgent(req syscallgw.SpawnRequest) (syscallgw.SpawnResult, error) {
	agentID := req.AgentID
	if agentID == "" {
		agentID = fmt.Sprintf("%s-%s-%d", req.TaskID, strings.ToLower(req.Role), time.Now().UnixNano())
	}
	state := string(avp.StateCreated)
	if s.registry != nil {
		agent, ok := s.registry.Get(agentID)
		if !ok {
			agent = s.registry.CreateAgent(agentID, req.Role, req.TaskID)
		}
		state = string(agent.State)
		if ws, err := s.ensureAgentWorkspace(req.TaskID, agentID); err == nil {
			s.registry.SetWorkspace(agentID, ws.MergedDir)
		}
	}
	if req.ParentAgentID != "" {
		for _, pageID := range s.cvm.PageTable(req.ParentAgentID).PageIDs {
			_ = s.cvm.MountPage(agentID, pageID)
		}
	}
	s.mu.Lock()
	if task, ok := s.tasks[req.TaskID]; ok {
		task.Agents = append(task.Agents, demo.Agent{ID: agentID, Role: req.Role, State: state})
		task.DAG = append(task.DAG, demo.DAGNode{ID: agentID, Role: req.Role, Dependencies: req.Dependencies})
		s.tasks[req.TaskID] = task
	}
	s.mu.Unlock()
	return syscallgw.SpawnResult{AgentID: agentID, Role: req.Role, TaskID: req.TaskID, State: state}, nil
}

func (s *Server) ensureAgentWorkspace(taskID, agentID string) (workspace.Workspace, error) {
	if status, err := s.workspace.Status(agentID); err == nil {
		return status.Workspace, nil
	}
	ws, err := s.workspace.Create(agentID)
	if err != nil {
		s.sink.Publish(events.New("agent.workspace_failed", taskID, agentID, "runtime", map[string]any{
			"error": err.Error(),
		}))
		return workspace.Workspace{}, err
	}
	s.sink.Publish(events.New("agent.workspace_attached", taskID, agentID, "runtime", map[string]any{
		"workspace_path":  ws.MergedDir,
		"mode":            ws.Mode,
		"evidence_mode":   ws.EvidenceMode,
		"fallback_reason": ws.FallbackReason,
	}))
	return ws, nil
}

type workerSyscallAdapter struct {
	gateway *syscallgw.Gateway
}

func (a workerSyscallAdapter) HandleSyscall(message worker.Message) worker.Response {
	if a.gateway == nil {
		return worker.Response{
			Type:      worker.MessageSyscallResult,
			RequestID: message.RequestID,
			AgentID:   message.AgentID,
			TaskID:    message.TaskID,
			Status:    syscallgw.StatusError,
			Error:     "syscall gateway is not configured",
		}
	}
	response := a.gateway.Handle(context.Background(), syscallgw.Request{
		RequestID: message.RequestID,
		AgentID:   message.AgentID,
		TaskID:    message.TaskID,
		Name:      message.Name,
		Args:      message.Args,
	})
	return worker.Response{
		Type:      worker.MessageSyscallResult,
		RequestID: response.RequestID,
		AgentID:   message.AgentID,
		TaskID:    message.TaskID,
		Status:    response.Status,
		Error:     response.Error,
		Payload:   response.Payload,
	}
}

func (s *Server) monitorHeartbeats() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for now := range ticker.C {
		if s.registry == nil {
			return
		}
		s.registry.MarkHeartbeatLost(now, s.heartbeatTimeout)
	}
}

func (s *Server) agentsForTask(task demo.Result) []demo.Agent {
	if s.registry == nil {
		return task.Agents
	}
	agents := s.registry.ListByTask(task.TaskID)
	if len(agents) == 0 {
		return task.Agents
	}
	byID := make(map[string]avp.AVP, len(agents))
	for _, agent := range s.enrichedAgents(agents) {
		byID[agent.AgentID] = agent
	}
	out := make([]demo.Agent, 0, len(task.Agents))
	for _, original := range task.Agents {
		if agent, ok := byID[original.ID]; ok {
			out = append(out, agentToDemo(agent))
			continue
		}
		out = append(out, original)
	}
	return out
}

func (s *Server) enrichedAgents(agents []avp.AVP) []avp.AVP {
	if s.capsules == nil {
		return agents
	}
	out := make([]avp.AVP, 0, len(agents))
	for _, agent := range agents {
		stats := s.capsules.Stats(agent.AgentID)
		agent.CapsuleMode = stats.Mode
		agent.MemoryCurrent = stats.MemoryCurrent
		agent.PidsCurrent = stats.PidsCurrent
		agent.CPUStat = stats.CPUStat
		if runtime, ok := s.capsules.Runtime(agent.AgentID); ok {
			agent.CgroupPath = runtime.CgroupPath
			if agent.CapsuleMode == "" {
				agent.CapsuleMode = runtime.Mode
			}
		}
		out = append(out, agent)
	}
	return out
}

func agentToDemo(agent avp.AVP) demo.Agent {
	return demo.Agent{
		ID:                  agent.AgentID,
		Role:                agent.Role,
		State:               string(agent.State),
		PID:                 agent.PID,
		LastSeen:            agent.LastSeen,
		CapsuleMode:         agent.CapsuleMode,
		CapsuleEvidenceMode: capsuleEvidenceMode(agent.CapsuleMode, agent.CgroupPath),
		CgroupPath:          agent.CgroupPath,
	}
}

func capsuleEvidenceMode(mode, cgroupPath string) string {
	switch {
	case mode == capsule.ModeReal && strings.HasPrefix(cgroupPath, "/sys/fs/cgroup/"):
		return "real-cgroup-v2"
	case mode == capsule.ModeReal && cgroupPath != "":
		return "real-cgroup-v2"
	case mode == capsule.ModeDegraded:
		return "degraded"
	default:
		return string(evidence.ModeMissing)
	}
}

type eventSink struct {
	hub      *events.Hub
	recorder *trace.Recorder
}

func newEventSink(cfg config.Config, hub *events.Hub) *eventSink {
	sink := &eventSink{hub: hub}
	if cfg.DataDir != "" {
		recorder, err := trace.NewRecorder(filepath.Join(cfg.DataDir, "traces"))
		if err == nil {
			sink.recorder = recorder
		}
	}
	return sink
}

func (s *eventSink) Publish(event events.Event) {
	s.hub.Publish(event)
	if s.recorder != nil && event.TaskID != "" {
		_ = s.recorder.Append(event)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
