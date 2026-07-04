package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"aort-r/internal/avp"
	"aort-r/internal/capsule"
	"aort-r/internal/config"
	"aort-r/internal/cvm"
	"aort-r/internal/demo"
	"aort-r/internal/events"
	"aort-r/internal/trace"
	"aort-r/internal/worker"
)

func NewServer(cfg config.Config) http.Handler {
	return NewServerWithEvents(cfg, events.NewHub(32))
}

func NewServerWithEvents(cfg config.Config, hub *events.Hub) http.Handler {
	server := &Server{
		cfg:       cfg,
		hub:       hub,
		sink:      newEventSink(cfg, hub),
		demo:      demo.NewSoftwareDemoRunner(),
		cvm:       cvm.NewStore(newEventSink(cfg, hub)),
		tasks:     make(map[string]demo.Result),
		workerCtx: context.Background(),
	}
	server.startWorkerRuntime()
	server.routes()
	return server
}

type Server struct {
	cfg              config.Config
	hub              *events.Hub
	sink             *eventSink
	demo             *demo.Runner
	cvm              *cvm.Store
	mux              *http.ServeMux
	mu               sync.RWMutex
	tasks            map[string]demo.Result
	registry         *worker.Registry
	capsules         *capsule.Manager
	uds              *worker.UDSServer
	launcher         worker.Launcher
	heartbeatTimeout time.Duration
	workerCtx        context.Context
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
	mux.HandleFunc("/api/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		streamEvents(w, r, s.hub)
	})
	mux.HandleFunc("/api/demo/run", s.handleDemoRun)
	mux.HandleFunc("/api/agents", s.handleAgents)
	mux.HandleFunc("/api/agents/", s.handleAgentAction)
	mux.HandleFunc("/api/context/pages", s.handleContextPages)
	mux.HandleFunc("/api/context/stats", s.handleContextStats)
	mux.HandleFunc("/api/context/agents/", s.handleContextAgent)
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

func (s *Server) runDemo(ctx context.Context) (demo.Result, error) {
	if s.registry == nil {
		result, err := s.demo.Run(ctx)
		if err != nil {
			return demo.Result{}, err
		}
		s.seedContext(result.TaskID, result.Agents)
		for _, event := range result.Events {
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
		},
	}
	s.sink.Publish(events.New("task.created", taskID, "", "runtime", map[string]any{"mode": "worker"}))
	specs := []worker.Spec{
		{AgentID: taskID + "-planner", Role: "Planner", TaskID: taskID},
		{AgentID: taskID + "-coder", Role: "Coder", TaskID: taskID},
		{AgentID: taskID + "-tester", Role: "Tester", TaskID: taskID},
	}
	for _, spec := range specs {
		agent := s.registry.CreateAgent(spec.AgentID, spec.Role, spec.TaskID)
		result.Agents = append(result.Agents, agentToDemo(agent))
		if _, err := s.launcher.Start(s.workerCtx, spec); err != nil {
			s.sink.Publish(events.New("agent.worker_start_failed", taskID, spec.AgentID, "runtime", map[string]any{"error": err.Error()}))
			return demo.Result{}, err
		}
	}
	s.seedContext(result.TaskID, result.Agents)
	return result, nil
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
	})
	s.launcher = worker.Launcher{Command: s.cfg.WorkerCommand, SocketPath: s.cfg.SocketPath}
	if s.cfg.HeartbeatTimeoutMS <= 0 {
		s.cfg.HeartbeatTimeoutMS = 6000
	}
	s.heartbeatTimeout = time.Duration(s.cfg.HeartbeatTimeoutMS) * time.Millisecond
	s.uds = worker.NewUDSServer(s.cfg.SocketPath, s.registry)
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
	out := make([]demo.Agent, 0, len(agents))
	for _, agent := range s.enrichedAgents(agents) {
		out = append(out, agentToDemo(agent))
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
		ID:         agent.AgentID,
		Role:       agent.Role,
		State:      string(agent.State),
		PID:        agent.PID,
		LastSeen:   agent.LastSeen,
		CgroupPath: agent.CgroupPath,
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
