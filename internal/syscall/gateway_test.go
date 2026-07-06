package syscallgw

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aort-r/internal/cvm"
	"aort-r/internal/events"
	"aort-r/internal/ipc"
	"aort-r/internal/llm"
)

type recordingSink struct {
	events []events.Event
}

func (s *recordingSink) Publish(event events.Event) {
	s.events = append(s.events, event)
}

type fakeWorkspaceRuntime struct {
	dir       string
	commits   []string
	rollbacks []string
	destroys  []string
}

func (r *fakeWorkspaceRuntime) WorkspaceDir(agentID string) (string, error) {
	return r.dir, nil
}

func (r *fakeWorkspaceRuntime) Commit(agentID string) error {
	r.commits = append(r.commits, agentID)
	return nil
}

func (r *fakeWorkspaceRuntime) Rollback(agentID string) error {
	r.rollbacks = append(r.rollbacks, agentID)
	return nil
}

func (r *fakeWorkspaceRuntime) Destroy(agentID string) error {
	r.destroys = append(r.destroys, agentID)
	return nil
}

func TestGatewayMaterializesContextAndAuditsRecord(t *testing.T) {
	store := cvm.NewStore(nil)
	page, err := store.CreatePage(cvm.KindSystem, "system prompt\n")
	if err != nil {
		t.Fatalf("CreatePage: %v", err)
	}
	if err := store.MountPage("agent-1", page.ID); err != nil {
		t.Fatalf("MountPage: %v", err)
	}
	sink := &recordingSink{}
	gateway := NewGateway(Config{CVM: store, Sink: sink, WorkspaceRoot: t.TempDir()})

	response := gateway.Handle(context.Background(), Request{
		AgentID: "agent-1",
		TaskID:  "task-1",
		Name:    "context.materialize",
	})

	if response.Status != StatusOK {
		t.Fatalf("status = %s error=%s", response.Status, response.Error)
	}
	if response.Payload["content"] != "system prompt\n" {
		t.Fatalf("payload = %#v", response.Payload)
	}
	records := gateway.Records()
	if len(records) != 1 {
		t.Fatalf("records = %#v", records)
	}
	if records[0].Name != "context.materialize" || records[0].Status != StatusOK {
		t.Fatalf("record = %#v", records[0])
	}
	if len(sink.events) != 2 || sink.events[0].Type != "syscall.started" || sink.events[1].Type != "syscall.finished" {
		t.Fatalf("events = %#v", sink.events)
	}
}

func TestGatewayToolExecUsesWorkspaceRuntimeAndCommitsOnSuccess(t *testing.T) {
	dir := t.TempDir()
	runtime := &fakeWorkspaceRuntime{dir: dir}
	gateway := NewGateway(Config{WorkspaceRuntime: runtime})

	response := gateway.Handle(context.Background(), Request{
		AgentID: "agent-1",
		TaskID:  "task-1",
		Name:    "tool.exec",
		Args: map[string]any{
			"command": "pwd",
		},
	})

	if response.Status != StatusOK {
		t.Fatalf("status = %s error=%s", response.Status, response.Error)
	}
	stdout, _ := response.Payload["stdout"].(string)
	if strings.TrimSpace(stdout) != dir {
		t.Fatalf("tool.exec cwd = %q, want %q", stdout, dir)
	}
	if len(runtime.commits) != 1 || runtime.commits[0] != "agent-1" {
		t.Fatalf("commit calls = %#v", runtime.commits)
	}
	if len(runtime.rollbacks) != 0 {
		t.Fatalf("unexpected rollback calls = %#v", runtime.rollbacks)
	}
}

func TestGatewayToolExecRollsBackWorkspaceRuntimeOnFailureAndTimeout(t *testing.T) {
	dir := t.TempDir()
	runtime := &fakeWorkspaceRuntime{dir: dir}
	gateway := NewGateway(Config{WorkspaceRuntime: runtime})

	failed := gateway.Handle(context.Background(), Request{
		AgentID: "agent-1",
		TaskID:  "task-1",
		Name:    "tool.exec",
		Args: map[string]any{
			"command": "sh",
			"args":    []any{"-c", "exit 7"},
		},
	})
	if failed.Status != StatusError {
		t.Fatalf("failure status = %s error=%s", failed.Status, failed.Error)
	}

	timedOut := gateway.Handle(context.Background(), Request{
		AgentID: "agent-2",
		TaskID:  "task-1",
		Name:    "tool.exec",
		Args: map[string]any{
			"command":    "sleep",
			"args":       []any{"1"},
			"timeout_ms": 10,
		},
	})
	if timedOut.Status != StatusTimeout {
		t.Fatalf("timeout status = %s error=%s", timedOut.Status, timedOut.Error)
	}

	if len(runtime.rollbacks) != 2 || runtime.rollbacks[0] != "agent-1" || runtime.rollbacks[1] != "agent-2" {
		t.Fatalf("rollback calls = %#v", runtime.rollbacks)
	}
	if len(runtime.commits) != 0 {
		t.Fatalf("unexpected commit calls = %#v", runtime.commits)
	}
}

func TestGatewayDestroyAgentWorkspaceRuntime(t *testing.T) {
	runtime := &fakeWorkspaceRuntime{dir: t.TempDir()}
	gateway := NewGateway(Config{WorkspaceRuntime: runtime})

	if err := gateway.DestroyAgent("agent-1"); err != nil {
		t.Fatalf("DestroyAgent: %v", err)
	}
	if len(runtime.destroys) != 1 || runtime.destroys[0] != "agent-1" {
		t.Fatalf("destroy calls = %#v", runtime.destroys)
	}
}

func TestGatewayToolExecTimeoutIsAudited(t *testing.T) {
	gateway := NewGateway(Config{WorkspaceRoot: t.TempDir()})

	response := gateway.Handle(context.Background(), Request{
		AgentID: "agent-1",
		TaskID:  "task-1",
		Name:    "tool.exec",
		Args: map[string]any{
			"command":    "sleep",
			"args":       []any{"1"},
			"timeout_ms": 10,
		},
	})

	if response.Status != StatusTimeout {
		t.Fatalf("status = %s error=%s", response.Status, response.Error)
	}
	if !strings.Contains(response.Error, "timeout") {
		t.Fatalf("error = %q", response.Error)
	}
	records := gateway.Records()
	if len(records) != 1 || records[0].Status != StatusTimeout {
		t.Fatalf("records = %#v", records)
	}
	if records[0].DurationMS <= 0 {
		t.Fatalf("duration was not recorded: %#v", records[0])
	}
}

func TestGatewayToolExecReportsKernelExecObservation(t *testing.T) {
	var observed ExecObservation
	root := t.TempDir()
	gateway := NewGateway(Config{
		WorkspaceRoot: root,
		ExecObserver: func(event ExecObservation) {
			observed = event
		},
	})

	response := gateway.Handle(context.Background(), Request{
		RequestID: "exec-1",
		AgentID:   "agent-1",
		TaskID:    "task-1",
		Name:      "tool.exec",
		Args: map[string]any{
			"command": "pwd",
		},
	})

	if response.Status != StatusOK {
		t.Fatalf("status = %s error=%s", response.Status, response.Error)
	}
	if observed.TaskID != "task-1" || observed.AgentID != "agent-1" || observed.Command != "pwd" {
		t.Fatalf("observed = %#v", observed)
	}
	if observed.PID == 0 || observed.Workspace == "" || observed.Status != StatusOK {
		t.Fatalf("observed = %#v", observed)
	}
	rootAbs, _ := filepath.Abs(root)
	expected := filepath.Join(rootAbs, "agent-1", "merged")
	if observed.Workspace != expected {
		t.Fatalf("tool exec should run in agent merged workspace, got %q", observed.Workspace)
	}
	if _, err := os.Stat(expected); err != nil {
		t.Fatalf("expected fallback workspace missing: %v", err)
	}
}

func TestGatewayRejectsToolExecOutsideWorkspace(t *testing.T) {
	gateway := NewGateway(Config{WorkspaceRoot: t.TempDir()})

	response := gateway.Handle(context.Background(), Request{
		AgentID: "agent-1",
		TaskID:  "task-1",
		Name:    "tool.exec",
		Args: map[string]any{
			"command": "pwd",
			"cwd":     "/tmp",
		},
	})

	if response.Status != StatusDenied {
		t.Fatalf("status = %s error=%s", response.Status, response.Error)
	}
}

func TestGatewayPublishesAndPollsPageReferenceIPC(t *testing.T) {
	store := cvm.NewStore(nil)
	page, err := store.CreatePage(cvm.KindDelta, "review feedback content")
	if err != nil {
		t.Fatalf("CreatePage: %v", err)
	}
	sink := &recordingSink{}
	gateway := NewGateway(Config{CVM: store, Sink: sink, IPC: ipc.NewBlackboard(), WorkspaceRoot: t.TempDir()})

	publish := gateway.Handle(context.Background(), Request{
		RequestID: "pub-1",
		AgentID:   "reviewer-1",
		TaskID:    "task-1",
		Name:      "ipc.publish",
		Args: map[string]any{
			"topic":      "review.feedback",
			"page_id":    page.ID,
			"size_bytes": page.Bytes,
		},
	})
	if publish.Status != StatusOK {
		t.Fatalf("publish status=%s error=%s", publish.Status, publish.Error)
	}
	if publish.Payload["avoided_copy_bytes"] != page.Bytes {
		t.Fatalf("publish payload = %#v", publish.Payload)
	}

	poll := gateway.Handle(context.Background(), Request{
		RequestID: "poll-1",
		AgentID:   "fixer-1",
		TaskID:    "task-1",
		Name:      "ipc.poll",
		Args: map[string]any{
			"topic": "review.feedback",
		},
	})
	if poll.Status != StatusOK {
		t.Fatalf("poll status=%s error=%s", poll.Status, poll.Error)
	}
	pageIDs, ok := poll.Payload["page_ids"].([]string)
	if !ok || len(pageIDs) != 1 || pageIDs[0] != page.ID {
		t.Fatalf("poll payload = %#v", poll.Payload)
	}
	table := store.PageTable("fixer-1")
	if len(table.PageIDs) != 1 || table.PageIDs[0] != page.ID {
		t.Fatalf("fixer page table = %#v", table)
	}
	if !containsEventType(sink.events, "ipc.published") || !containsEventType(sink.events, "ipc.polled") {
		t.Fatalf("expected IPC events, got %#v", sink.events)
	}
}

func TestGatewayAgentSpawnCallsRuntimeSpawner(t *testing.T) {
	var spawned SpawnRequest
	sink := &recordingSink{}
	gateway := NewGateway(Config{
		Sink:          sink,
		WorkspaceRoot: t.TempDir(),
		Spawner: func(req SpawnRequest) (SpawnResult, error) {
			spawned = req
			return SpawnResult{AgentID: "fixer-1", Role: req.Role, TaskID: req.TaskID, State: "CREATED"}, nil
		},
	})

	response := gateway.Handle(context.Background(), Request{
		RequestID: "spawn-1",
		AgentID:   "reviewer-1",
		TaskID:    "task-1",
		Name:      "agent.spawn",
		Args: map[string]any{
			"agent_id":     "fixer-1",
			"role":         "fixer",
			"reason":       "tester failed",
			"dependencies": []any{"reviewer-1"},
		},
	})

	if response.Status != StatusOK {
		t.Fatalf("status=%s error=%s", response.Status, response.Error)
	}
	if spawned.AgentID != "fixer-1" || spawned.ParentAgentID != "reviewer-1" || spawned.Role != "fixer" || spawned.Reason != "tester failed" {
		t.Fatalf("spawned = %#v", spawned)
	}
	if response.Payload["agent_id"] != "fixer-1" {
		t.Fatalf("payload = %#v", response.Payload)
	}
	if !containsEventType(sink.events, "agent.spawn.requested") || !containsEventType(sink.events, "agent.spawned") {
		t.Fatalf("expected spawn events, got %#v", sink.events)
	}
}

func TestGatewayLLMCallUsesRouterAndAuditsUsage(t *testing.T) {
	router := llm.NewRouter()
	router.Register("mock", llm.NewMockProvider("mock"))
	router.SetDefault("mock")
	sink := &recordingSink{}
	gateway := NewGateway(Config{LLM: router, Sink: sink, WorkspaceRoot: t.TempDir()})

	response := gateway.Handle(context.Background(), Request{
		RequestID: "llm-1",
		AgentID:   "planner-1",
		TaskID:    "task-1",
		Name:      "llm.call",
		Args: map[string]any{
			"prompt": "plan the task",
		},
	})

	if response.Status != StatusOK {
		t.Fatalf("status=%s error=%s", response.Status, response.Error)
	}
	if response.Payload["provider"] != "mock" || response.Payload["model"] != "mock" {
		t.Fatalf("payload = %#v", response.Payload)
	}
	usage, ok := response.Payload["usage"].(map[string]any)
	if !ok || usage["prompt_tokens"] == 0 {
		t.Fatalf("usage payload = %#v", response.Payload["usage"])
	}
	records := gateway.Records()
	if len(records) != 1 {
		t.Fatalf("records = %#v", records)
	}
	if records[0].Evidence["provider"] != "mock" || records[0].Evidence["model"] != "mock" || records[0].Evidence["evidence_mode"] != "mock" {
		t.Fatalf("record evidence = %#v", records[0].Evidence)
	}
	if records[0].Evidence["duration_ms"] == nil || records[0].Evidence["tokens"] == nil {
		t.Fatalf("record evidence missing duration/tokens = %#v", records[0].Evidence)
	}
	if !containsEventType(sink.events, "llm.called") {
		t.Fatalf("expected llm.called event, got %#v", sink.events)
	}
}

func TestGatewayLLMCallRecordsDeepSeekFallbackEvidence(t *testing.T) {
	router := llm.NewRouter()
	router.Register("deepseek", llm.NewDeepSeekProvider(llm.DeepSeekConfig{Model: "deepseek-v4-flash"}))
	router.Register("mock", llm.NewMockProvider("mock"))
	router.SetDefault("deepseek")
	router.SetFallback("mock")
	gateway := NewGateway(Config{LLM: router, WorkspaceRoot: t.TempDir()})

	response := gateway.Handle(context.Background(), Request{
		RequestID: "llm-fallback-1",
		AgentID:   "planner-1",
		TaskID:    "task-1",
		Name:      "llm.call",
		Args: map[string]any{
			"prompt": "plan the task",
		},
	})

	if response.Status != StatusOK {
		t.Fatalf("status=%s error=%s", response.Status, response.Error)
	}
	if response.Payload["provider"] != "mock" || response.Payload["requested_provider"] != "deepseek" {
		t.Fatalf("payload = %#v", response.Payload)
	}
	if response.Payload["model"] != "mock" || response.Payload["fallback"] != true || response.Payload["fallback_reason"] != "no_api_key" || response.Payload["evidence_mode"] != "mock" {
		t.Fatalf("fallback payload = %#v", response.Payload)
	}
	records := gateway.Records()
	if len(records) != 1 {
		t.Fatalf("records = %#v", records)
	}
	if records[0].Evidence["provider"] != "mock" || records[0].Evidence["model"] != "mock" || records[0].Evidence["requested_provider"] != "deepseek" || records[0].Evidence["fallback"] != true {
		t.Fatalf("record evidence = %#v", records[0].Evidence)
	}
}

func containsEventType(events []events.Event, eventType string) bool {
	for _, event := range events {
		if event.Type == eventType {
			return true
		}
	}
	return false
}
