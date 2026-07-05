# AORT-R Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build AORT-R in three iterative versions, ending with a full openEuler Agent Runtime prototype that demonstrates OS-level Agent abstraction, scheduling, isolation, context optimization, communication, observability, and recovery.

**Architecture:** AORT-R is a Go `aortd` runtime daemon with per-Agent worker processes, a Vue dashboard, and a set of testable runtime subsystems. V1 builds the runnable runtime skeleton and simulated demo; V2 adds real OS mechanisms and context/scheduling innovation; V3 adds full observability, checkpoint recovery, experiments, and competition delivery material.

**Tech Stack:** Go 1.22+, Vue 3 + Vite + ECharts, REST + SSE, Unix Domain Socket JSON-RPC, cgroup v2, overlayfs, namespace, bbolt, JSONL trace, cilium/ebpf, llama.cpp local provider, DeepSeek OpenAI-compatible relay, openEuler 24.03 LTS.

## Global Constraints

- Main platform is openEuler 24.03 LTS with root permission in a VM.
- Team size is 2 people; member A owns backend/runtime/deployment, member B owns dashboard/demo/experiments/reporting.
- Backend language is Go; frontend stack is Vue 3 + Vite + ECharts.
- The project must not modify the Linux kernel.
- The project must support mock mode so demos and tests can run without a network LLM call.
- Every runtime mechanism shown in the defense must have evidence: implementation module, dashboard view, and test or experiment output.
- The implementation must degrade in this order: eBPF openat/connect, workspace tar checkpoint, PSI throttle, parallel dual Coder merge, summary compression, openKylin smoke.
- The non-negotiable baseline is AVP state machine, worker process, per-Agent cgroup, syscall gateway, CVM page table, token-CFS, overlayfs rollback, at least three fault injections, dashboard core pages, and E1/E2/E3 experiments.

---

## Iteration Overview

### Version 1: Runtime Skeleton and Demo Baseline

**Purpose:** Build a runnable AORT-R shell that already looks like an OS-level Agent Runtime from the outside. It should create AVPs, stream events to the dashboard, run a deterministic software-engineering demo in mock mode, and produce trace files.

**After V1, what was built:**

- Go monorepo skeleton with `aortd`.
- AVP lifecycle and DAG execution.
- REST + SSE API.
- Mock worker execution without root-only OS isolation.
- Mock CVM pages and syscall audit.
- Vue dashboard with Overview, AVP, Context, Timeline, Experiments placeholders backed by real API data.
- Demo task: Todo Web API workflow using Planner, Coder, Tester, Reviewer, Fixer.
- First iteration report at `docs/iteration-reports/v1-foundation.md`.

**How the user tests V1:**

```bash
go test ./...
go run ./cmd/aortd --config configs/dev.yaml
curl -s http://127.0.0.1:8080/api/health
curl -s -X POST http://127.0.0.1:8080/api/demo/run
curl -N http://127.0.0.1:8080/api/events
cd dashboard
npm install
npm run dev
```

Expected result:

- `/api/health` returns `{"status":"ok","mode":"mock"}`.
- `/api/demo/run` returns a task id.
- `/api/events` streams `task.updated`, `agent.state_changed`, `scheduler.selected`, `syscall.finished`, and `task.completed`.
- Dashboard shows the task DAG, AVP state changes, context pages, and timeline events.

### Version 2: Real OS Mechanisms and Core Innovation

**Purpose:** Convert the V1 skeleton into a real OS-backed runtime. Each Agent gets a real worker process and cgroup, tool calls go through UDS syscall gateway, workspaces are isolated by overlayfs, CVM is real content-addressed storage, and token-CFS + prefix affinity decisions are logged.

**After V2, what was built:**

- Worker re-exec model with UDS registration.
- Per-Agent cgroup v2 creation, resource stats, freeze/unfreeze/kill.
- Tool sandbox with overlayfs upper/work/merged directories.
- Workspace commit and rollback.
- Real CVM page store using sha256 page ids.
- Syscall gateway with capability, quota, timeout, audit, and trace.
- token-CFS scheduler and prefix affinity grouping.
- Page reference IPC blackboard.
- Supervisor for retry, OOM/PID event handling, dynamic Fixer spawn.
- Fault injection scripts for forkbomb, OOM, rmrf, and conflict.
- Second iteration report at `docs/iteration-reports/v2-os-core.md`.

**How the user tests V2 on openEuler:**

```bash
sudo ./scripts/check-openeuler-env.sh
go test ./...
sudo go run ./cmd/aortd --config configs/openeuler-dev.yaml
curl -s -X POST http://127.0.0.1:8080/api/demo/run
curl -s http://127.0.0.1:8080/api/agents
curl -s -X POST http://127.0.0.1:8080/api/demo/fault/forkbomb
curl -s -X POST http://127.0.0.1:8080/api/demo/fault/oom
curl -s -X POST http://127.0.0.1:8080/api/demo/fault/rmrf
```

Expected result:

- Each Agent has a PID and cgroup path under `/sys/fs/cgroup/aort.slice/`.
- Freeze writes `cgroup.freeze=1`; unfreeze writes `cgroup.freeze=0`.
- `forkbomb` is stopped by `pids.max`.
- `oom` increments `memory.events` and triggers Supervisor retry.
- `rmrf` only damages the Agent upper layer, and the main snapshot remains readable.
- Scheduler logs include `strategy=token-cfs-prefix`, `vruntime`, `prefix_group`, and `decision_reason`.

### Version 3: Full Innovation, Experiments, and Competition Package

**Purpose:** Finish the full AORT-R plan. Add llama.cpp prefix-cache measurements, eBPF kernel timeline, checkpoint/recovery, experiment automation, systemd deployment, cross-distro smoke, PPT evidence, and demo recording material.

**After V3, what was built:**

- LLM Router with mock, DeepSeek relay, and llama.cpp local provider.
- Prefix cache or timing metric collection for scheduler experiments.
- eBPF Observer for at least `sched_process_exec`; `openat` and `connect` are included when time allows.
- Checkpoint/recovery with daemon crash demonstration.
- E1/E2/E3/E4 experiment scripts and JSON outputs.
- Dashboard experiment page with charts.
- systemd service and install script.
- openEuler full deployment guide and Ubuntu or openKylin smoke guide.
- Third iteration report at `docs/iteration-reports/v3-full-aort-r.md`.
- Competition package checklist at `docs/delivery/competition-checklist.md`.

**How the user tests V3:**

```bash
sudo ./scripts/install-openeuler.sh
sudo systemctl enable --now aortd
systemctl status aortd --no-pager
curl -s -X POST http://127.0.0.1:8080/api/demo/run
sudo ./scripts/demo-daemonkill.sh
go run ./cmd/aort-experiment --name e1-scheduler --runs 5
go run ./cmd/aort-experiment --name e2-fault --runs 5
go run ./cmd/aort-experiment --name e3-context-ipc --runs 5
go run ./cmd/aort-experiment --name e4-observability-recovery --runs 3
```

Expected result:

- `aortd` restarts after daemon kill and resumes from checkpoint.
- Timeline shows application events, syscall audit, and at least one kernel event lane.
- Experiment JSON files are written under `experiments/results/`.
- Dashboard Experiments page renders E1/E2/E3/E4 charts.
- `docs/delivery/competition-checklist.md` marks all required competition artifacts as present.

---

## File Structure Map

### Backend

- Create `go.mod`: Go module definition, Go version, backend dependencies.
- Create `cmd/aortd/main.go`: starts config, state store, managers, HTTP API, SSE hub, demo runner.
- Create `cmd/aort-worker/main.go`: worker process entrypoint; connects to UDS and executes assigned Agent script.
- Create `cmd/aort-experiment/main.go`: runs E1/E2/E3/E4 experiments and writes JSON results.
- Create `internal/config/config.go`: config loading and default values.
- Create `internal/avp/types.go`: AVP structs, Agent states, roles, lifecycle events.
- Create `internal/avp/manager.go`: AVP creation, state transitions, dependency readiness.
- Create `internal/dag/dag.go`: DAG nodes, edges, ready-node calculation.
- Create `internal/events/hub.go`: in-memory publish/subscribe hub for SSE and trace recorder.
- Create `internal/api/server.go`: REST routes and SSE endpoint.
- Create `internal/demo/software_demo.go`: deterministic Todo API demo workflow.
- Create `internal/trace/recorder.go`: JSONL trace writer and in-memory query index.
- Create `internal/state/store.go`: bbolt or JSON state abstraction; V1 may use JSON file, V2 migrates to bbolt.
- Create `internal/capsule/manager.go`: cgroup lifecycle, worker start, freeze/unfreeze/kill.
- Create `internal/workspace/overlay.go`: snapshot, overlay upper/work/merged setup, commit, rollback.
- Create `internal/cvm/page.go`: content-addressed page model.
- Create `internal/cvm/store.go`: page store, page table, materialize, reference accounting.
- Create `internal/syscall/gateway.go`: UDS JSON-RPC, capability, quota, timeout, audit.
- Create `internal/tool/runner.go`: tool execution through sandbox.
- Create `internal/scheduler/scheduler.go`: FIFO, token-CFS, prefix affinity, decision logs.
- Create `internal/ipc/blackboard.go`: topic publish/poll with page references.
- Create `internal/supervisor/supervisor.go`: retry, fault classification, dynamic spawn.
- Create `internal/checkpoint/checkpoint.go`: runtime checkpoint and recovery.
- Create `internal/llm/router.go`: provider interface and routing policy.
- Create `internal/llm/mock.go`: deterministic LLM provider.
- Create `internal/llm/deepseek_provider.go`: OpenAI-compatible relay provider.
- Create `internal/llm/llamacpp.go`: llama.cpp local provider and timing parser.
- Create `internal/ebpf/observer.go`: kernel event observer with graceful disabled mode.
- Create `internal/experiment/e1_scheduler.go`: scheduler experiment.
- Create `internal/experiment/e2_fault.go`: fault isolation experiment.
- Create `internal/experiment/e3_context_ipc.go`: context and IPC experiment.
- Create `internal/experiment/e4_observe_recover.go`: observability and recovery experiment.

### Frontend

- Create `dashboard/package.json`: frontend scripts and dependencies.
- Create `dashboard/src/main.ts`: Vue entry.
- Create `dashboard/src/App.vue`: app shell.
- Create `dashboard/src/api/client.ts`: REST and SSE client.
- Create `dashboard/src/stores/runtime.ts`: task, agent, context, trace, experiment state.
- Create `dashboard/src/pages/Overview.vue`: DAG and task overview.
- Create `dashboard/src/pages/AvpCapsule.vue`: Agent state and cgroup controls.
- Create `dashboard/src/pages/ContextMemory.vue`: pages, page tables, ref counts, IPC metrics.
- Create `dashboard/src/pages/Timeline.vue`: application, syscall, kernel event lanes.
- Create `dashboard/src/pages/Experiments.vue`: experiment charts and raw JSON links.
- Create `dashboard/src/components/StatusBadge.vue`: status rendering.
- Create `dashboard/src/components/MetricCard.vue`: compact metrics.
- Create `dashboard/src/components/EventTimeline.vue`: timeline rendering.

### Scripts and Documentation

- Create `configs/dev.yaml`: local mock config.
- Create `configs/openeuler-dev.yaml`: root-enabled openEuler config.
- Create `scripts/check-openeuler-env.sh`: checks cgroup v2, overlayfs, bpffs, Go, Node, root.
- Create `scripts/install-openeuler.sh`: builds and installs service.
- Create `scripts/demo-daemonkill.sh`: triggers daemon crash recovery demo.
- Create `deploy/aortd.service`: systemd unit.
- Create `docs/iteration-reports/v1-foundation.md`: V1 report.
- Create `docs/iteration-reports/v2-os-core.md`: V2 report.
- Create `docs/iteration-reports/v3-full-aort-r.md`: V3 report.
- Create `docs/testing/manual-test-guide.md`: manual verification guide.
- Create `docs/delivery/competition-checklist.md`: final delivery checklist.

---

## Shared Interfaces

All tasks must use these names so modules fit together.

```go
package avp

type AgentState string

const (
    StateCreated    AgentState = "CREATED"
    StateReady      AgentState = "READY"
    StateRunning    AgentState = "RUNNING"
    StateWaitingLLM AgentState = "WAITING_LLM"
    StateWaitingTool AgentState = "WAITING_TOOL"
    StateWaitingIPC AgentState = "WAITING_IPC"
    StateSuspended  AgentState = "SUSPENDED"
    StateCompleted  AgentState = "COMPLETED"
    StateFailed     AgentState = "FAILED"
    StateKilled     AgentState = "KILLED"
)

type AVP struct {
    AgentID      string     `json:"agent_id"`
    TaskID       string     `json:"task_id"`
    Role         string     `json:"role"`
    State        AgentState `json:"state"`
    Weight       int        `json:"weight"`
    VRuntime     uint64     `json:"vruntime"`
    Dependencies []string   `json:"dependencies"`
    PageTable    []string   `json:"page_table"`
    PID          int        `json:"pid"`
    CgroupPath   string     `json:"cgroup_path"`
    RetryCount   int        `json:"retry_count"`
}
```

```go
package events

type Event struct {
    ID        string         `json:"id"`
    TaskID    string         `json:"task_id"`
    AgentID   string         `json:"agent_id,omitempty"`
    Type      string         `json:"type"`
    Source    string         `json:"source"`
    Timestamp int64          `json:"timestamp"`
    Payload   map[string]any `json:"payload"`
}
```

```go
package cvm

type Page struct {
    ID        string            `json:"id"`
    Kind      string            `json:"kind"`
    Bytes     []byte            `json:"-"`
    Tokens    int               `json:"tokens"`
    RefCount  int               `json:"ref_count"`
    Meta      map[string]string `json:"meta"`
}
```

```go
package scheduler

type Decision struct {
    TaskID         string `json:"task_id"`
    AgentID        string `json:"agent_id"`
    Strategy       string `json:"strategy"`
    VRuntime       uint64 `json:"vruntime"`
    PrefixGroup    string `json:"prefix_group"`
    DecisionReason string `json:"decision_reason"`
}
```

---

## Version 1 Tasks: Runtime Skeleton and Demo Baseline

### Task 1: Repository Skeleton, Config, Health API

**Files:**
- Create: `go.mod`
- Create: `cmd/aortd/main.go`
- Create: `internal/config/config.go`
- Create: `internal/api/server.go`
- Create: `configs/dev.yaml`
- Test: `internal/config/config_test.go`
- Test: `internal/api/server_test.go`

**Interfaces:**
- Produces: `config.Load(path string) (Config, error)`
- Produces: `api.NewServer(cfg config.Config) http.Handler`
- Produces: `GET /api/health -> {"status":"ok","mode":"mock"}`

- [ ] **Step 1: Write config test**

```go
package config

import "testing"

func TestLoadDevConfig(t *testing.T) {
    cfg, err := Load("../../configs/dev.yaml")
    if err != nil {
        t.Fatalf("Load returned error: %v", err)
    }
    if cfg.HTTPAddr != "127.0.0.1:8080" {
        t.Fatalf("HTTPAddr = %q", cfg.HTTPAddr)
    }
    if cfg.Mode != "mock" {
        t.Fatalf("Mode = %q", cfg.Mode)
    }
}
```

- [ ] **Step 2: Write health API test**

```go
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
```

- [ ] **Step 3: Run tests to confirm they fail before implementation**

Run: `go test ./internal/config ./internal/api`  
Expected: fail because packages and functions are not defined.

- [ ] **Step 4: Implement minimal config and API**

Create `configs/dev.yaml` with:

```yaml
http_addr: 127.0.0.1:8080
mode: mock
data_dir: .aort-dev
```

Create `internal/config/config.go` with a small YAML loader using `gopkg.in/yaml.v3`.

Create `internal/api/server.go` with `GET /api/health`.

- [ ] **Step 5: Run tests**

Run: `go test ./internal/config ./internal/api`  
Expected: pass.

- [ ] **Step 6: Manual run**

Run: `go run ./cmd/aortd --config configs/dev.yaml`  
Expected: logs include `aortd listening on 127.0.0.1:8080`.

- [ ] **Step 7: Commit**

```bash
git add go.mod go.sum cmd/aortd internal/config internal/api configs/dev.yaml
git commit -m "feat: add aortd skeleton and health api"
```

### Task 2: AVP Lifecycle and DAG Manager

**Files:**
- Create: `internal/avp/types.go`
- Create: `internal/avp/manager.go`
- Create: `internal/dag/dag.go`
- Test: `internal/avp/manager_test.go`
- Test: `internal/dag/dag_test.go`

**Interfaces:**
- Consumes: `events.Event` from shared interfaces.
- Produces: `avp.Manager.Create(taskID, role string, deps []string) AVP`
- Produces: `avp.Manager.Transition(agentID string, next AgentState) error`
- Produces: `dag.Graph.Ready(completed map[string]bool) []string`

- [ ] **Step 1: Write AVP lifecycle tests**

```go
func TestAVPTransitionCreatedReadyRunningCompleted(t *testing.T) {
    mgr := NewManager()
    a := mgr.Create("task-1", "planner", nil)
    if a.State != StateCreated {
        t.Fatalf("initial state = %s", a.State)
    }
    for _, state := range []AgentState{StateReady, StateRunning, StateCompleted} {
        if err := mgr.Transition(a.AgentID, state); err != nil {
            t.Fatalf("transition to %s failed: %v", state, err)
        }
    }
    got, ok := mgr.Get(a.AgentID)
    if !ok || got.State != StateCompleted {
        t.Fatalf("final AVP = %#v ok=%v", got, ok)
    }
}

func TestAVPRejectsInvalidCompletedToRunning(t *testing.T) {
    mgr := NewManager()
    a := mgr.Create("task-1", "planner", nil)
    _ = mgr.Transition(a.AgentID, StateReady)
    _ = mgr.Transition(a.AgentID, StateRunning)
    _ = mgr.Transition(a.AgentID, StateCompleted)
    if err := mgr.Transition(a.AgentID, StateRunning); err == nil {
        t.Fatalf("expected invalid transition error")
    }
}
```

- [ ] **Step 2: Write DAG tests**

```go
func TestGraphReadyReturnsNodesWhoseDepsCompleted(t *testing.T) {
    g := NewGraph()
    g.AddNode("planner", nil)
    g.AddNode("coder", []string{"planner"})
    g.AddNode("tester", []string{"coder"})
    ready := g.Ready(map[string]bool{"planner": true})
    if len(ready) != 1 || ready[0] != "coder" {
        t.Fatalf("ready = %#v", ready)
    }
}
```

- [ ] **Step 3: Run tests to confirm failure**

Run: `go test ./internal/avp ./internal/dag`  
Expected: fail because managers are missing.

- [ ] **Step 4: Implement AVP and DAG**

Implement the shared `AVP` and `AgentState` types exactly as defined in Shared Interfaces. Implement valid transitions:

```text
CREATED -> READY
READY -> RUNNING
RUNNING -> WAITING_LLM
RUNNING -> WAITING_TOOL
RUNNING -> WAITING_IPC
WAITING_LLM -> READY
WAITING_TOOL -> READY
WAITING_IPC -> READY
RUNNING -> SUSPENDED
SUSPENDED -> READY
RUNNING -> COMPLETED
RUNNING -> FAILED
FAILED -> READY
FAILED -> KILLED
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/avp ./internal/dag`  
Expected: pass.

- [ ] **Step 6: Commit**

```bash
git add internal/avp internal/dag
git commit -m "feat: add avp lifecycle and dag manager"
```

### Task 3: Event Hub, Trace Recorder, and SSE

**Files:**
- Create: `internal/events/hub.go`
- Create: `internal/trace/recorder.go`
- Modify: `internal/api/server.go`
- Test: `internal/events/hub_test.go`
- Test: `internal/trace/recorder_test.go`
- Test: `internal/api/sse_test.go`

**Interfaces:**
- Produces: `events.Hub.Publish(Event)`
- Produces: `events.Hub.Subscribe() (<-chan Event, func())`
- Produces: `trace.Recorder.Append(events.Event) error`
- Produces: `GET /api/events` SSE stream.

- [ ] **Step 1: Write hub test**

```go
func TestHubPublishesToSubscriber(t *testing.T) {
    hub := NewHub(4)
    ch, cancel := hub.Subscribe()
    defer cancel()
    event := Event{ID: "e1", TaskID: "t1", Type: "task.updated", Source: "runtime", Timestamp: 1}
    hub.Publish(event)
    got := <-ch
    if got.ID != "e1" || got.Type != "task.updated" {
        t.Fatalf("event = %#v", got)
    }
}
```

- [ ] **Step 2: Write trace recorder test**

```go
func TestRecorderWritesJSONL(t *testing.T) {
    dir := t.TempDir()
    rec, err := NewRecorder(dir)
    if err != nil {
        t.Fatalf("NewRecorder: %v", err)
    }
    event := events.Event{ID: "e1", TaskID: "t1", Type: "agent.created", Source: "runtime", Timestamp: 1}
    if err := rec.Append(event); err != nil {
        t.Fatalf("Append: %v", err)
    }
    data, err := os.ReadFile(filepath.Join(dir, "t1.jsonl"))
    if err != nil {
        t.Fatalf("ReadFile: %v", err)
    }
    if !strings.Contains(string(data), `"agent.created"`) {
        t.Fatalf("trace data = %s", data)
    }
}
```

- [ ] **Step 3: Implement hub and recorder**

Use buffered channels and non-blocking publish. If a subscriber channel is full, drop the event for that subscriber and keep runtime execution moving.

- [ ] **Step 4: Add `/api/events`**

SSE format must be:

```text
event: <event.Type>
data: <json event>

```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/events ./internal/trace ./internal/api`  
Expected: pass.

- [ ] **Step 6: Commit**

```bash
git add internal/events internal/trace internal/api
git commit -m "feat: add event hub trace recorder and sse"
```

### Task 4: Mock Demo Runner and REST Task API

**Files:**
- Create: `internal/demo/software_demo.go`
- Modify: `internal/api/server.go`
- Test: `internal/demo/software_demo_test.go`
- Test: `internal/api/demo_api_test.go`

**Interfaces:**
- Produces: `demo.RunSoftwareDemo(ctx context.Context) (taskID string, err error)`
- Produces: `POST /api/demo/run`
- Produces: `GET /api/tasks`
- Produces: `GET /api/tasks/{task_id}/dag`

- [ ] **Step 1: Write demo test**

```go
func TestSoftwareDemoProducesExpectedRoles(t *testing.T) {
    runner := NewSoftwareDemoRunner()
    result, err := runner.Run(context.Background())
    if err != nil {
        t.Fatalf("Run: %v", err)
    }
    roles := result.Roles()
    want := []string{"planner", "coder-a", "coder-b", "tester", "reviewer", "fixer"}
    if !reflect.DeepEqual(roles, want) {
        t.Fatalf("roles = %#v", roles)
    }
}
```

- [ ] **Step 2: Write API test**

```go
func TestDemoRunEndpointCreatesTask(t *testing.T) {
    srv := NewServer(config.Config{HTTPAddr: "127.0.0.1:8080", Mode: "mock"})
    req := httptest.NewRequest(http.MethodPost, "/api/demo/run", nil)
    rec := httptest.NewRecorder()
    srv.ServeHTTP(rec, req)
    if rec.Code != http.StatusAccepted {
        t.Fatalf("status = %d body=%s", rec.Code, rec.Body.String())
    }
    if !strings.Contains(rec.Body.String(), "task_id") {
        t.Fatalf("body = %s", rec.Body.String())
    }
}
```

- [ ] **Step 3: Implement deterministic mock workflow**

Emit these events in order:

```text
task.created
agent.created planner
agent.state_changed planner READY
scheduler.selected planner
syscall.finished context.materialize
agent.state_changed planner COMPLETED
agent.created coder-a
agent.created coder-b
agent.state_changed coder-a COMPLETED
agent.state_changed coder-b COMPLETED
agent.created tester
syscall.finished tool.go_test exit_code=1
agent.created reviewer
agent.created fixer
agent.state_changed fixer COMPLETED
syscall.finished tool.go_test exit_code=0
task.completed
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/demo ./internal/api`  
Expected: pass.

- [ ] **Step 5: Manual test**

Run:

```bash
go run ./cmd/aortd --config configs/dev.yaml
curl -s -X POST http://127.0.0.1:8080/api/demo/run
curl -N http://127.0.0.1:8080/api/events
```

Expected: event stream includes `task.completed`.

- [ ] **Step 6: Commit**

```bash
git add internal/demo internal/api
git commit -m "feat: add mock software demo api"
```

### Task 5: V1 Dashboard and V1 Report

**Files:**
- Create: `dashboard/package.json`
- Create: `dashboard/src/main.ts`
- Create: `dashboard/src/App.vue`
- Create: `dashboard/src/api/client.ts`
- Create: `dashboard/src/stores/runtime.ts`
- Create: `dashboard/src/pages/Overview.vue`
- Create: `dashboard/src/pages/AvpCapsule.vue`
- Create: `dashboard/src/pages/ContextMemory.vue`
- Create: `dashboard/src/pages/Timeline.vue`
- Create: `dashboard/src/pages/Experiments.vue`
- Create: `docs/iteration-reports/v1-foundation.md`
- Test: `docs/testing/manual-test-guide.md`

**Interfaces:**
- Consumes: REST and SSE APIs from Tasks 1-4.
- Produces: Dashboard that can run with `npm run dev`.
- Produces: V1 report with Done, How to Test, Evidence, Known Risks, Next Version.

- [ ] **Step 1: Create dashboard scripts**

`dashboard/package.json` must include:

```json
{
  "scripts": {
    "dev": "vite --host 127.0.0.1",
    "build": "vite build",
    "test": "vue-tsc --noEmit"
  },
  "dependencies": {
    "@vitejs/plugin-vue": "^5.0.0",
    "echarts": "^5.5.0",
    "vite": "^5.0.0",
    "vue": "^3.4.0"
  },
  "devDependencies": {
    "typescript": "^5.4.0",
    "vue-tsc": "^2.0.0"
  }
}
```

- [ ] **Step 2: Implement pages with real API client**

The dashboard must show:

```text
Overview: task id, status, DAG role list
AVP & Capsule: Agent id, role, state, retry count
Context Memory: page id, kind, ref count
Timeline: event type, source, agent id, timestamp
Experiments: text saying V1 collects no experiment metrics yet and links to V2 plan
```

- [ ] **Step 3: Build frontend**

Run:

```bash
cd dashboard
npm install
npm run test
npm run build
```

Expected: TypeScript check and production build pass.

- [ ] **Step 4: Write V1 report**

Create `docs/iteration-reports/v1-foundation.md` with:

```markdown
# V1 Foundation Report

## Done

- aortd starts in mock mode.
- Health API returns ok.
- Demo API creates a deterministic software-engineering task.
- SSE streams runtime events.
- Dashboard displays task, AVP, context, timeline, and experiment entry page.

## How To Test

```bash
go test ./...
go run ./cmd/aortd --config configs/dev.yaml
curl -s http://127.0.0.1:8080/api/health
curl -s -X POST http://127.0.0.1:8080/api/demo/run
cd dashboard
npm install
npm run dev
```

## Expected Evidence

- Browser shows task DAG and AVP state changes.
- Terminal SSE stream contains `task.completed`.
- Trace file exists under `.aort-dev/traces/`.

## Known Risks

- V1 uses mock execution and does not yet create real cgroups.
- V1 does not yet isolate tool processes with overlayfs.

## Next Version

- V2 replaces mock execution with worker processes, cgroups, overlayfs, CVM, syscall gateway, scheduler, IPC, and Supervisor.
```

- [ ] **Step 5: Commit**

```bash
git add dashboard docs/iteration-reports/v1-foundation.md docs/testing/manual-test-guide.md
git commit -m "feat: add v1 dashboard and foundation report"
```

---

## Version 2 Tasks: Real OS Mechanisms and Core Innovation

### Task 6: Worker Process and Per-Agent cgroup Capsule

**Files:**
- Create: `cmd/aort-worker/main.go`
- Create: `internal/capsule/manager.go`
- Modify: `internal/avp/types.go`
- Modify: `internal/api/server.go`
- Test: `internal/capsule/manager_test.go`
- Create: `configs/openeuler-dev.yaml`

**Interfaces:**
- Produces: `capsule.Manager.StartWorker(ctx context.Context, avp avp.AVP) (capsule.Runtime, error)`
- Produces: `capsule.Manager.Freeze(agentID string) error`
- Produces: `capsule.Manager.Unfreeze(agentID string) error`
- Produces: `capsule.Manager.Kill(agentID string) error`
- Produces: `POST /api/agents/{id}/freeze|unfreeze|kill`

- [ ] **Step 1: Write cgroup path test with fake filesystem**

```go
func TestCapsuleCreatesCgroupFiles(t *testing.T) {
    root := t.TempDir()
    mgr := NewManager(Config{CgroupRoot: root, WorkerPath: "/bin/true"})
    runtime, err := mgr.Prepare("agent-1", Limits{MemoryMax: "256M", PidsMax: 64, CPUMax: "100000 100000"})
    if err != nil {
        t.Fatalf("Prepare: %v", err)
    }
    if !strings.Contains(runtime.CgroupPath, "agent-1") {
        t.Fatalf("cgroup path = %s", runtime.CgroupPath)
    }
    assertFileContains(t, filepath.Join(runtime.CgroupPath, "memory.max"), "256M")
    assertFileContains(t, filepath.Join(runtime.CgroupPath, "pids.max"), "64")
}

func assertFileContains(t *testing.T, path string, want string) {
    t.Helper()
    data, err := os.ReadFile(path)
    if err != nil {
        t.Fatalf("ReadFile(%s): %v", path, err)
    }
    if strings.TrimSpace(string(data)) != want {
        t.Fatalf("%s = %q want %q", path, strings.TrimSpace(string(data)), want)
    }
}
```

- [ ] **Step 2: Implement fakeable cgroup writer**

Write cgroup files through an interface:

```go
type FileSystem interface {
    MkdirAll(path string, perm os.FileMode) error
    WriteFile(path string, data []byte, perm os.FileMode) error
    ReadFile(path string) ([]byte, error)
}
```

- [ ] **Step 3: Implement real openEuler cgroup paths**

Default `CgroupRoot` is `/sys/fs/cgroup/aort.slice`. Write:

```text
cpu.max
memory.max
pids.max
cgroup.freeze
cgroup.procs
```

- [ ] **Step 4: Add API controls**

Freeze writes `1` to `cgroup.freeze`, unfreeze writes `0`, kill sends SIGKILL to the worker PID and records `agent.state_changed KILLED`.

- [ ] **Step 5: Run unit tests**

Run: `go test ./internal/capsule ./internal/api`  
Expected: pass with fake filesystem.

- [ ] **Step 6: Run openEuler manual test**

Run:

```bash
sudo go run ./cmd/aortd --config configs/openeuler-dev.yaml
curl -s -X POST http://127.0.0.1:8080/api/demo/run
curl -s http://127.0.0.1:8080/api/agents
```

Expected: each Agent includes non-zero `pid` and a cgroup path.

- [ ] **Step 7: Commit**

```bash
git add cmd/aort-worker internal/capsule internal/avp internal/api configs/openeuler-dev.yaml
git commit -m "feat: add worker process and cgroup capsule"
```

### Task 7: Overlay Workspace and Tool Sandbox

**Files:**
- Create: `internal/workspace/overlay.go`
- Create: `internal/tool/runner.go`
- Modify: `internal/capsule/manager.go`
- Test: `internal/workspace/overlay_test.go`
- Test: `internal/tool/runner_test.go`

**Interfaces:**
- Produces: `workspace.Manager.Create(agentID string) (Workspace, error)`
- Produces: `workspace.Manager.Commit(agentID string) (snapshotID string, error)`
- Produces: `workspace.Manager.Rollback(agentID string) error`
- Produces: `tool.Runner.Exec(ctx context.Context, req ExecRequest) (ExecResult, error)`

- [ ] **Step 1: Write overlay directory test**

```go
func TestWorkspaceCreateBuildsUpperWorkMerged(t *testing.T) {
    root := t.TempDir()
    mgr := NewManager(Config{Root: root})
    ws, err := mgr.Create("agent-1")
    if err != nil {
        t.Fatalf("Create: %v", err)
    }
    for _, dir := range []string{ws.UpperDir, ws.WorkDir, ws.MergedDir} {
        info, err := os.Stat(dir)
        if err != nil || !info.IsDir() {
            t.Fatalf("dir %s stat=%v err=%v", dir, info, err)
        }
    }
}
```

- [ ] **Step 2: Implement workspace directories**

Directory layout:

```text
<data_dir>/snapshots/base
<data_dir>/capsules/<agent_id>/upper
<data_dir>/capsules/<agent_id>/work
<data_dir>/capsules/<agent_id>/merged
```

- [ ] **Step 3: Implement tool runner with timeout**

`ExecRequest` fields:

```go
type ExecRequest struct {
    AgentID    string
    Command    []string
    WorkDir    string
    TimeoutMS  int
    Env        map[string]string
}
```

If timeout expires, kill the process group and return `ExecResult{ExitCode: -1, ErrorType: "TOOL_TIMEOUT"}`.

- [ ] **Step 4: Add manual overlay mount path**

On openEuler with root, mount overlay using:

```text
mount -t overlay overlay -o lowerdir=<snapshot>,upperdir=<upper>,workdir=<work> <merged>
```

In non-root tests, skip mount and use directory creation only.

- [ ] **Step 5: Run tests**

Run: `go test ./internal/workspace ./internal/tool`  
Expected: pass.

- [ ] **Step 6: Manual rollback test**

Run fault `rmrf` after Task 12 is available; expected result is main snapshot unchanged and Agent upper discarded.

- [ ] **Step 7: Commit**

```bash
git add internal/workspace internal/tool internal/capsule
git commit -m "feat: add overlay workspace and tool runner"
```

### Task 8: CVM Page Store and Materialization

**Files:**
- Create: `internal/cvm/page.go`
- Create: `internal/cvm/store.go`
- Modify: `internal/api/server.go`
- Test: `internal/cvm/store_test.go`

**Interfaces:**
- Produces: `cvm.Store.Put(kind string, data []byte, meta map[string]string) (Page, error)`
- Produces: `cvm.Store.Mount(agentID, pageID string) error`
- Produces: `cvm.Store.Materialize(agentID string) (string, []Page, error)`
- Produces: `GET /api/context/pages`
- Produces: `GET /api/context/agents/{agent_id}/pagetable`

- [ ] **Step 1: Write content addressing test**

```go
func TestPutSameBytesReturnsSamePageID(t *testing.T) {
    store := NewMemoryStore()
    p1, err := store.Put("project", []byte("same content"), nil)
    if err != nil {
        t.Fatalf("Put p1: %v", err)
    }
    p2, err := store.Put("project", []byte("same content"), nil)
    if err != nil {
        t.Fatalf("Put p2: %v", err)
    }
    if p1.ID != p2.ID {
        t.Fatalf("ids differ: %s %s", p1.ID, p2.ID)
    }
}
```

- [ ] **Step 2: Write materialize order test**

```go
func TestMaterializeUsesStableOrder(t *testing.T) {
    store := NewMemoryStore()
    sys, _ := store.Put("system", []byte("system\n"), nil)
    proj, _ := store.Put("project", []byte("project\n"), nil)
    task, _ := store.Put("task", []byte("task\n"), nil)
    delta, _ := store.Put("delta", []byte("delta\n"), nil)
    _ = store.Mount("agent-1", sys.ID)
    _ = store.Mount("agent-1", proj.ID)
    _ = store.Mount("agent-1", task.ID)
    _ = store.Mount("agent-1", delta.ID)
    got, _, err := store.Materialize("agent-1")
    if err != nil {
        t.Fatalf("Materialize: %v", err)
    }
    if got != "system\nproject\ntask\ndelta\n" {
        t.Fatalf("materialized = %q", got)
    }
}
```

- [ ] **Step 3: Implement sha256 page ids**

Use lowercase hex sha256 of page bytes as `Page.ID`.

- [ ] **Step 4: Implement page ref count**

Increment `RefCount` when a page is mounted into a new Agent page table. Do not increment when the same Agent mounts the same page twice.

- [ ] **Step 5: Add context API**

Return JSON pages without `Bytes`, but include `id`, `kind`, `tokens`, `ref_count`, and `meta`.

- [ ] **Step 6: Run tests**

Run: `go test ./internal/cvm ./internal/api`  
Expected: pass.

- [ ] **Step 7: Commit**

```bash
git add internal/cvm internal/api
git commit -m "feat: add cvm page store and materialization"
```

### Task 9: Syscall Gateway and Audit

**Files:**
- Create: `internal/syscall/gateway.go`
- Modify: `cmd/aort-worker/main.go`
- Modify: `internal/trace/recorder.go`
- Test: `internal/syscall/gateway_test.go`

**Interfaces:**
- Produces: `syscall.Gateway.Handle(ctx context.Context, req Request) (Response, error)`
- Produces: syscall names `context.materialize`, `context.write_delta`, `llm.call`, `tool.exec`, `ipc.publish`, `ipc.poll`, `agent.spawn`, `agent.report`

- [ ] **Step 1: Write capability rejection test**

```go
func TestGatewayRejectsMissingCapability(t *testing.T) {
    gw := NewGateway(Config{}, Dependencies{})
    req := Request{AgentID: "agent-1", Name: "tool.exec", Capabilities: []string{"context.materialize"}}
    _, err := gw.Handle(context.Background(), req)
    if err == nil || !strings.Contains(err.Error(), "capability denied") {
        t.Fatalf("err = %v", err)
    }
}
```

- [ ] **Step 2: Write audit success test**

```go
func TestGatewayRecordsAuditEvent(t *testing.T) {
    sink := events.NewHub(8)
    ch, cancel := sink.Subscribe()
    defer cancel()
    gw := NewGateway(Config{}, Dependencies{Events: sink})
    req := Request{AgentID: "agent-1", TaskID: "task-1", Name: "context.materialize", Capabilities: []string{"context.materialize"}}
    _, err := gw.Handle(context.Background(), req)
    if err != nil {
        t.Fatalf("Handle: %v", err)
    }
    deadline := time.After(time.Second)
    for {
        select {
        case got := <-ch:
            if got.Type == "syscall.finished" {
                return
            }
        case <-deadline:
            t.Fatalf("did not receive syscall.finished")
        }
    }
}
```

- [ ] **Step 3: Implement gateway pipeline**

Pipeline order:

```text
capability check
quota check
timeout context
audit start event
execute dispatcher
fault classification
audit finish event
response
```

- [ ] **Step 4: Implement UDS server skeleton**

Socket path is `/run/aort/aortd.sock` in openEuler config and `.aort-dev/aortd.sock` in dev config.

- [ ] **Step 5: Run tests**

Run: `go test ./internal/syscall ./cmd/aort-worker`  
Expected: pass.

- [ ] **Step 6: Commit**

```bash
git add internal/syscall cmd/aort-worker internal/trace
git commit -m "feat: add syscall gateway and audit"
```

### Task 10: token-CFS Scheduler and Prefix Affinity

**Files:**
- Create: `internal/scheduler/scheduler.go`
- Modify: `internal/demo/software_demo.go`
- Modify: `internal/api/server.go`
- Test: `internal/scheduler/scheduler_test.go`

**Interfaces:**
- Produces: `scheduler.NewTokenCFS(prefixWindow uint64) *Scheduler`
- Produces: `Scheduler.Select(ready []avp.AVP, pageTables map[string][]string) Decision`
- Produces: `Scheduler.ReportUsage(agentID string, tokens int)`

- [ ] **Step 1: Write token-CFS fairness test**

```go
func TestTokenCFSSelectsLowestVRuntime(t *testing.T) {
    s := NewTokenCFS(2000)
    ready := []avp.AVP{
        {AgentID: "a", Weight: 100, VRuntime: 5000},
        {AgentID: "b", Weight: 100, VRuntime: 1000},
    }
    decision := s.Select(ready, nil)
    if decision.AgentID != "b" {
        t.Fatalf("selected = %s", decision.AgentID)
    }
}
```

- [ ] **Step 2: Write prefix affinity test**

```go
func TestPrefixAffinityCanSelectSamePrefixWithinWindow(t *testing.T) {
    s := NewTokenCFS(2000)
    s.SetLastPrefixGroup("sys/project")
    ready := []avp.AVP{
        {AgentID: "a", Weight: 100, VRuntime: 1000},
        {AgentID: "b", Weight: 100, VRuntime: 2500},
    }
    pageTables := map[string][]string{
        "a": []string{"sys", "other"},
        "b": []string{"sys", "project", "task"},
    }
    decision := s.Select(ready, pageTables)
    if decision.AgentID != "b" {
        t.Fatalf("selected = %s reason=%s", decision.AgentID, decision.DecisionReason)
    }
}
```

- [ ] **Step 3: Implement strategy names**

Use:

```text
fifo
token-cfs
token-cfs-prefix
```

- [ ] **Step 4: Emit decision events**

Event type is `scheduler.selected`; payload includes `strategy`, `vruntime`, `prefix_group`, `decision_reason`.

- [ ] **Step 5: Run tests**

Run: `go test ./internal/scheduler ./internal/demo`  
Expected: pass.

- [ ] **Step 6: Commit**

```bash
git add internal/scheduler internal/demo internal/api
git commit -m "feat: add token cfs prefix scheduler"
```

### Task 11: Page Reference IPC Blackboard

**Files:**
- Create: `internal/ipc/blackboard.go`
- Modify: `internal/syscall/gateway.go`
- Modify: `internal/api/server.go`
- Test: `internal/ipc/blackboard_test.go`

**Interfaces:**
- Produces: `ipc.Blackboard.Publish(topic, pageID string, sizeBytes int) Metric`
- Produces: `ipc.Blackboard.Poll(topic string, subscriber string) ([]string, Metric)`
- Produces: metric `avoided_copy_bytes`

- [ ] **Step 1: Write publish/poll test**

```go
func TestBlackboardPublishesPageReferences(t *testing.T) {
    bb := NewBlackboard()
    metric := bb.Publish("review", "page-1", 1024)
    if metric.Messages != 1 {
        t.Fatalf("publish metric = %#v", metric)
    }
    pages, metric := bb.Poll("review", "agent-2")
    if len(pages) != 1 || pages[0] != "page-1" {
        t.Fatalf("pages = %#v", pages)
    }
    if metric.AvoidedCopyBytes != 1024 {
        t.Fatalf("avoided = %d", metric.AvoidedCopyBytes)
    }
}
```

- [ ] **Step 2: Implement topic queues**

Store messages by topic with stable insertion order. A subscriber receives each page id once.

- [ ] **Step 3: Wire syscalls**

`ipc.publish` receives `topic` and `page_id`; `ipc.poll` receives `topic` and mounts returned page ids into the caller page table.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/ipc ./internal/syscall`  
Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add internal/ipc internal/syscall internal/api
git commit -m "feat: add page reference ipc blackboard"
```

### Task 12: Supervisor and Fault Injection

**Files:**
- Create: `internal/supervisor/supervisor.go`
- Modify: `internal/demo/software_demo.go`
- Modify: `internal/api/server.go`
- Create: `docs/iteration-reports/v2-os-core.md`
- Test: `internal/supervisor/supervisor_test.go`

**Interfaces:**
- Produces: `supervisor.HandleFault(ctx context.Context, fault Fault) Action`
- Produces: `POST /api/demo/fault/forkbomb`
- Produces: `POST /api/demo/fault/oom`
- Produces: `POST /api/demo/fault/rmrf`
- Produces: `POST /api/demo/fault/conflict`

- [ ] **Step 1: Write Supervisor retry test**

```go
func TestSupervisorRetriesOOMWithNewCapsule(t *testing.T) {
    sup := NewSupervisor(Config{MaxRetries: 2})
    action := sup.HandleFault(context.Background(), Fault{AgentID: "agent-1", Type: "CAPSULE_OOM", RetryCount: 0})
    if action.Type != ActionRetryNewCapsule {
        t.Fatalf("action = %#v", action)
    }
}
```

- [ ] **Step 2: Write retry limit test**

```go
func TestSupervisorKillsAfterRetryLimit(t *testing.T) {
    sup := NewSupervisor(Config{MaxRetries: 2})
    action := sup.HandleFault(context.Background(), Fault{AgentID: "agent-1", Type: "CAPSULE_OOM", RetryCount: 2})
    if action.Type != ActionKillAgent {
        t.Fatalf("action = %#v", action)
    }
}
```

- [ ] **Step 3: Implement fault actions**

Actions:

```text
RETRY_NEW_CAPSULE
KILL_AGENT
SPAWN_FIXER
ROLLBACK_WORKSPACE
OPEN_CIRCUIT
```

- [ ] **Step 4: Implement demo fault endpoints**

`forkbomb` runs a tool command that repeatedly forks under `pids.max`.  
`oom` runs a tool command that allocates memory above `memory.max`.  
`rmrf` runs a tool command that deletes files inside merged workspace.  
`conflict` creates two upper layers modifying the same path and emits `workspace.conflict`.

- [ ] **Step 5: Write V2 report**

Create `docs/iteration-reports/v2-os-core.md` with:

```markdown
# V2 OS Core Report

## Done

- Worker processes run under per-Agent cgroups.
- Freeze, unfreeze, and kill operate through cgroup and process control.
- Tool calls execute through syscall gateway and are audited.
- Workspaces use overlay-style upper/work/merged directories.
- CVM stores content-addressed pages and stable page tables.
- token-CFS and prefix affinity emit decision logs.
- Page reference IPC records avoided copy bytes.
- Supervisor handles OOM, PID limit, rmrf rollback, and conflict-triggered Fixer spawn.

## How To Test

```bash
sudo ./scripts/check-openeuler-env.sh
go test ./...
sudo go run ./cmd/aortd --config configs/openeuler-dev.yaml
curl -s -X POST http://127.0.0.1:8080/api/demo/run
curl -s -X POST http://127.0.0.1:8080/api/demo/fault/forkbomb
curl -s -X POST http://127.0.0.1:8080/api/demo/fault/oom
curl -s -X POST http://127.0.0.1:8080/api/demo/fault/rmrf
```

## Expected Evidence

- `/api/agents` shows PID and cgroup path.
- Dashboard AVP page shows resource stats.
- Timeline shows `supervisor.retry` and `syscall.finished`.
- Context page shows ref count and IPC avoided bytes.

## Known Risks

- eBPF timeline is not part of V2.
- Checkpoint recovery is not part of V2.
- llama.cpp prefix-cache metrics are part of V3.

## Next Version

- V3 adds llama.cpp metrics, eBPF observer, checkpoint recovery, experiment automation, systemd deployment, and delivery artifacts.
```

- [ ] **Step 6: Run tests**

Run: `go test ./internal/supervisor ./internal/demo ./internal/api`  
Expected: pass.

- [ ] **Step 7: Commit**

```bash
git add internal/supervisor internal/demo internal/api docs/iteration-reports/v2-os-core.md
git commit -m "feat: add supervisor fault injection and v2 report"
```

---

## Version 3 Tasks: Full Innovation and Competition Package

### Task 13: LLM Router and llama.cpp Timing Metrics

**Files:**
- Create: `internal/llm/router.go`
- Create: `internal/llm/mock.go`
- Create: `internal/llm/deepseek_provider.go`
- Create: `internal/llm/llamacpp.go`
- Modify: `internal/syscall/gateway.go`
- Test: `internal/llm/router_test.go`

**Interfaces:**
- Produces: `llm.Provider.Complete(ctx context.Context, req Request) (Response, Usage, error)`
- Produces: `Usage{PromptTokens, CompletionTokens, CachedTokens, TTFTMS, PrefillMS int}`

- [ ] **Step 1: Write mock provider test**

```go
func TestMockProviderReturnsUsage(t *testing.T) {
    p := NewMockProvider("fixed response")
    resp, usage, err := p.Complete(context.Background(), Request{Prompt: "hello"})
    if err != nil {
        t.Fatalf("Complete: %v", err)
    }
    if resp.Text != "fixed response" {
        t.Fatalf("text = %q", resp.Text)
    }
    if usage.PromptTokens == 0 {
        t.Fatalf("usage = %#v", usage)
    }
}
```

- [ ] **Step 2: Write llama.cpp timing parser test**

```go
func TestParseLlamaTimings(t *testing.T) {
    body := `{"timings":{"prompt_n":128,"predicted_n":16,"prompt_ms":250,"predicted_ms":100},"content":"ok"}`
    usage, err := ParseLlamaUsage([]byte(body))
    if err != nil {
        t.Fatalf("ParseLlamaUsage: %v", err)
    }
    if usage.PromptTokens != 128 || usage.PrefillMS != 250 {
        t.Fatalf("usage = %#v", usage)
    }
}
```

- [ ] **Step 3: Implement provider router**

Roles default to:

```text
planner -> deepseek-relay
coder -> deepseek-relay
tester -> mock
reviewer -> deepseek-relay
fixer -> deepseek-relay
experiments -> llamacpp-local
```

If a provider fails twice, fallback to `mock` for demo stability.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/llm ./internal/syscall`  
Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add internal/llm internal/syscall
git commit -m "feat: add llm router and llama timing metrics"
```

### Task 14: eBPF Observer with Graceful Disabled Mode

**Files:**
- Create: `internal/ebpf/observer.go`
- Modify: `internal/trace/recorder.go`
- Modify: `internal/api/server.go`
- Test: `internal/ebpf/observer_test.go`

**Interfaces:**
- Produces: `ebpf.Observer.Start(ctx context.Context) error`
- Produces: kernel events with `Source: "kernel"` and `Type: "kernel.exec"` for execve.

- [ ] **Step 1: Write disabled mode test**

```go
func TestObserverDisabledReturnsNoError(t *testing.T) {
    obs := NewObserver(Config{Enabled: false})
    if err := obs.Start(context.Background()); err != nil {
        t.Fatalf("Start disabled: %v", err)
    }
}
```

- [ ] **Step 2: Write event mapping test**

```go
func TestMapKernelEventToTraceEvent(t *testing.T) {
    e := MapKernelEvent(KernelEvent{CgroupPath: "/sys/fs/cgroup/aort.slice/agent-1", Command: "go"})
    if e.Type != "kernel.exec" || e.Source != "kernel" || e.AgentID != "agent-1" {
        t.Fatalf("event = %#v", e)
    }
}
```

- [ ] **Step 3: Implement observer**

If bpffs or permissions are unavailable, emit one event:

```text
kernel.observer_disabled
```

and keep aortd running.

- [ ] **Step 4: Implement execve first**

Use `sched:sched_process_exec` as the first kernel event. Add `openat` and `connect` only after execve is visible on Timeline.

- [ ] **Step 5: Run tests**

Run: `go test ./internal/ebpf ./internal/trace ./internal/api`  
Expected: pass.

- [ ] **Step 6: Manual openEuler test**

Run:

```bash
sudo go run ./cmd/aortd --config configs/openeuler-dev.yaml
curl -s -X POST http://127.0.0.1:8080/api/demo/run
curl -s http://127.0.0.1:8080/api/traces/latest
```

Expected: trace includes `kernel.exec` or `kernel.observer_disabled`.

- [ ] **Step 7: Commit**

```bash
git add internal/ebpf internal/trace internal/api
git commit -m "feat: add ebpf observer timeline events"
```

### Task 15: Checkpoint, Recovery, and daemonkill Demo

**Files:**
- Create: `internal/checkpoint/checkpoint.go`
- Create: `scripts/demo-daemonkill.sh`
- Modify: `cmd/aortd/main.go`
- Modify: `internal/demo/software_demo.go`
- Test: `internal/checkpoint/checkpoint_test.go`

**Interfaces:**
- Produces: `checkpoint.Store.Save(snapshot Snapshot) error`
- Produces: `checkpoint.Store.LoadLatest(taskID string) (Snapshot, error)`
- Produces: `checkpoint.Recover(taskID string) error`

- [ ] **Step 1: Write save/load test**

```go
func TestCheckpointSaveLoadLatest(t *testing.T) {
    dir := t.TempDir()
    store := NewStore(dir)
    snap := Snapshot{TaskID: "task-1", CompletedAgents: []string{"planner"}, SchedulerVRuntime: map[string]uint64{"planner": 100}}
    if err := store.Save(snap); err != nil {
        t.Fatalf("Save: %v", err)
    }
    got, err := store.LoadLatest("task-1")
    if err != nil {
        t.Fatalf("LoadLatest: %v", err)
    }
    if got.TaskID != "task-1" || got.CompletedAgents[0] != "planner" {
        t.Fatalf("snapshot = %#v", got)
    }
}
```

- [ ] **Step 2: Save checkpoint after each completed Agent**

Snapshot fields:

```go
type Snapshot struct {
    TaskID            string            `json:"task_id"`
    CompletedAgents   []string          `json:"completed_agents"`
    PageTables        map[string][]string `json:"page_tables"`
    SchedulerVRuntime map[string]uint64 `json:"scheduler_vruntime"`
    TraceOffset       int64             `json:"trace_offset"`
    CreatedAtUnix     int64             `json:"created_at_unix"`
}
```

- [ ] **Step 3: Recover unfinished task on aortd startup**

When state contains a task without `task.completed`, load the latest checkpoint and recreate READY AVPs for remaining DAG nodes.

- [ ] **Step 4: Create daemonkill script**

`scripts/demo-daemonkill.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail
pid="$(pgrep -f 'aortd' | head -n 1)"
echo "killing aortd pid=${pid}"
sudo kill -9 "${pid}"
sleep 3
systemctl status aortd --no-pager
curl -s http://127.0.0.1:8080/api/tasks
```

- [ ] **Step 5: Run tests**

Run: `go test ./internal/checkpoint ./cmd/aortd ./internal/demo`  
Expected: pass.

- [ ] **Step 6: Manual recovery test**

Run:

```bash
sudo systemctl restart aortd
curl -s -X POST http://127.0.0.1:8080/api/demo/run
sudo ./scripts/demo-daemonkill.sh
```

Expected: task resumes and reaches `task.completed`.

- [ ] **Step 7: Commit**

```bash
git add internal/checkpoint scripts/demo-daemonkill.sh cmd/aortd internal/demo
git commit -m "feat: add checkpoint recovery and daemonkill demo"
```

### Task 16: Experiment Automation and Charts

**Files:**
- Create: `cmd/aort-experiment/main.go`
- Create: `internal/experiment/e1_scheduler.go`
- Create: `internal/experiment/e2_fault.go`
- Create: `internal/experiment/e3_context_ipc.go`
- Create: `internal/experiment/e4_observe_recover.go`
- Modify: `dashboard/src/pages/Experiments.vue`
- Test: `internal/experiment/experiment_test.go`

**Interfaces:**
- Produces: `go run ./cmd/aort-experiment --name e1-scheduler --runs 5`
- Produces: JSON files under `experiments/results/`

- [ ] **Step 1: Write experiment result test**

```go
func TestExperimentResultHasMeanAndStddev(t *testing.T) {
    result := Summarize("e1-scheduler", []float64{10, 20, 30})
    if result.Mean != 20 {
        t.Fatalf("mean = %f", result.Mean)
    }
    if result.Stddev <= 0 {
        t.Fatalf("stddev = %f", result.Stddev)
    }
}
```

- [ ] **Step 2: Implement E1 output schema**

```json
{
  "name": "e1-scheduler",
  "runs": 5,
  "metrics": {
    "fifo_wall_ms": {"mean": 0, "stddev": 0},
    "token_cfs_wall_ms": {"mean": 0, "stddev": 0},
    "prefix_affinity_wall_ms": {"mean": 0, "stddev": 0},
    "cached_tokens": {"mean": 0, "stddev": 0},
    "jain_index": {"mean": 0, "stddev": 0}
  }
}
```

Use real measured numbers when available. Use mock provider deterministic timing only when running on a laptop without llama.cpp.

- [ ] **Step 3: Implement E2/E3/E4 schemas**

E2 metrics:

```text
affected_sibling_agents
recovery_ms
final_success
oom_kill_events
pid_limit_events
```

E3 metrics:

```text
total_prompt_tokens
dedup_saved_tokens
page_ref_count
avoided_copy_bytes
materialize_ms
```

E4 metrics:

```text
trace_events
kernel_events
checkpoint_write_ms
recovery_ms
replayable_stages
```

- [ ] **Step 4: Wire dashboard charts**

Experiments page must load `/api/experiments/{name}` and render bar charts for means with labels for standard deviation.

- [ ] **Step 5: Run tests**

Run:

```bash
go test ./internal/experiment
go run ./cmd/aort-experiment --name e1-scheduler --runs 2
go run ./cmd/aort-experiment --name e2-fault --runs 2
go run ./cmd/aort-experiment --name e3-context-ipc --runs 2
```

Expected: JSON files exist under `experiments/results/`.

- [ ] **Step 6: Commit**

```bash
git add cmd/aort-experiment internal/experiment dashboard/src/pages/Experiments.vue
git commit -m "feat: add experiment automation and charts"
```

### Task 17: systemd Deployment, Environment Checks, and Delivery Docs

**Files:**
- Create: `scripts/check-openeuler-env.sh`
- Create: `scripts/install-openeuler.sh`
- Create: `deploy/aortd.service`
- Create: `docs/testing/manual-test-guide.md`
- Create: `docs/delivery/competition-checklist.md`
- Create: `docs/iteration-reports/v3-full-aort-r.md`

**Interfaces:**
- Produces: `sudo ./scripts/check-openeuler-env.sh`
- Produces: `sudo ./scripts/install-openeuler.sh`
- Produces: `systemctl status aortd --no-pager`

- [ ] **Step 1: Create environment check script**

Checks:

```text
running as root
go version exists
node version exists
/sys/fs/cgroup exists
cgroup v2 mounted
overlay module available or overlayfs listed in /proc/filesystems
/sys/kernel/btf/vmlinux exists or eBPF disabled mode is accepted
```

- [ ] **Step 2: Create systemd unit**

`deploy/aortd.service`:

```ini
[Unit]
Description=AORT-R Agent Runtime
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/aortd --config /etc/aort/config.yaml
Restart=always
RestartSec=2
User=root
Group=root

[Install]
WantedBy=multi-user.target
```

- [ ] **Step 3: Create install script**

Install script must:

```text
build aortd
copy binary to /usr/local/bin/aortd
copy config to /etc/aort/config.yaml
copy systemd unit to /etc/systemd/system/aortd.service
run systemctl daemon-reload
enable and start aortd
```

- [ ] **Step 4: Write final manual test guide**

`docs/testing/manual-test-guide.md` must include:

```text
V1 mock demo test
V2 cgroup and fault injection test
V3 daemonkill recovery test
E1 scheduler experiment
E2 fault isolation experiment
E3 context and IPC experiment
E4 observability recovery experiment
Dashboard screenshot checklist
```

- [ ] **Step 5: Write V3 report**

Create `docs/iteration-reports/v3-full-aort-r.md` with:

```markdown
# V3 Full AORT-R Report

## Done

- LLM router supports mock, DeepSeek relay, and llama.cpp local provider.
- Scheduler experiments record timing and cache-related metrics.
- Timeline includes application events, syscall audit, and kernel observer events.
- Checkpoint recovery resumes a demo after daemon kill.
- Experiment automation writes JSON result files.
- Dashboard renders experiment charts.
- systemd deployment is available for openEuler.

## How To Test

```bash
sudo ./scripts/install-openeuler.sh
systemctl status aortd --no-pager
curl -s -X POST http://127.0.0.1:8080/api/demo/run
sudo ./scripts/demo-daemonkill.sh
go run ./cmd/aort-experiment --name e1-scheduler --runs 5
go run ./cmd/aort-experiment --name e2-fault --runs 5
go run ./cmd/aort-experiment --name e3-context-ipc --runs 5
go run ./cmd/aort-experiment --name e4-observability-recovery --runs 3
```

## Expected Evidence

- aortd is managed by systemd.
- Dashboard Timeline has runtime, syscall, and kernel lanes.
- Experiments page renders E1/E2/E3/E4 charts.
- daemonkill demo resumes and completes.
- Competition checklist marks all artifacts as present.

## Remaining Risks

- DeepSeek relay cache fields depend on relay behavior.
- openat/connect eBPF events may be disabled if kernel headers or BTF support are missing.
```

- [ ] **Step 6: Write competition checklist**

Checklist entries:

```text
source code
openEuler install guide
demo video
PPT
experiment report
dashboard screenshots
E1 result JSON
E2 result JSON
E3 result JSON
E4 result JSON
systemd service
fault injection scripts
manual test guide
```

- [ ] **Step 7: Run final verification**

Run:

```bash
go test ./...
cd dashboard
npm run test
npm run build
cd ..
sudo ./scripts/check-openeuler-env.sh
```

Expected: all commands pass, or `check-openeuler-env.sh` prints an explicit warning for eBPF disabled mode while exiting 0.

- [ ] **Step 8: Commit**

```bash
git add scripts deploy docs
git commit -m "docs: add deployment checks and delivery package"
```

---

## Self-Review Checklist

### Spec Coverage

- AVP lifecycle is covered by Tasks 2 and 6.
- Agent Capsule is covered by Tasks 6 and 7.
- Syscall Gateway is covered by Task 9.
- CVM is covered by Task 8.
- token-CFS and prefix affinity are covered by Task 10.
- Page reference IPC is covered by Task 11.
- Supervisor and fault injection are covered by Task 12.
- LLM Router and llama.cpp metrics are covered by Task 13.
- eBPF Timeline is covered by Task 14.
- Checkpoint recovery is covered by Task 15.
- Experiments E1/E2/E3/E4 are covered by Task 16.
- systemd deployment and delivery docs are covered by Task 17.
- Dashboard is introduced in Task 5 and expanded in Tasks 16 and 17.

### Type Consistency

- Agent states use the same names as the final design document.
- Scheduler decision fields use `strategy`, `vruntime`, `prefix_group`, and `decision_reason`.
- CVM page fields use `id`, `kind`, `tokens`, `ref_count`, and `meta`.
- Event fields use `id`, `task_id`, `agent_id`, `type`, `source`, `timestamp`, and `payload`.

### Execution Rule

Implement tasks in order. At the end of each task:

```bash
go test ./...
git status --short
```

At the end of each version, produce the iteration report named in that version and run the version-specific manual test commands before moving to the next version.
