# Real DeepSeek Large-Codebase DAG Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build and prove a real-only AORT-R DAG that uses `deepseek-v4-flash` to complete the three review-remediation changes on the 20,000-plus-line AORT-R Go repository on Huawei openEuler.

**Architecture:** A strict DeepSeek client and immutable run store feed a validated DAG executor. Each LLM node is represented by a stopped child worker that is attached to a real cgroup before execution, receives shared CVM context through cross-process memfd/SCM_RIGHTS, operates in a real OverlayFS workspace, and reports through the existing Unix-socket Gateway. Runner-owned acceptance scripts, full Go tests, artifact hashes, and a final evidence validator are authoritative; model prose cannot override a failed machine gate.

**Tech Stack:** Go 1.22 standard library, DeepSeek OpenAI-compatible HTTP API, cgroup v2, PSI, OverlayFS, Unix domain sockets, memfd/mmap, SCM_RIGHTS, Git, Bash, Python 3 only for evidence assertions already used by repository scripts, Huawei Cloud openEuler 24.03 LTS.

## Global Constraints

- The real run provider is only `deepseek`; the exact requested and returned model is `deepseek-v4-flash`.
- The run must contain at least seven successful real API calls and zero mock, fallback, skipped, planned, simulation, or degraded LLM calls.
- A single run permits at most ten API calls: seven required calls, up to two fixer calls, and one schema-repair retry.
- `DEEPSEEK_API_KEY` is read only from the process environment and never appears in source, commands, logs, JSON, Markdown, patches, or Git history.
- Both tracked physical Go lines and tracked nonblank Go lines in the clean workload must be at least 20,000 before any completion call.
- Real evidence requires openEuler 24.03 LTS, UID 0, `cgroup2fs`, writable nested cgroups, real OverlayFS, and real cross-process memfd/mmap plus FD passing.
- Every result uses a new run-ID directory opened with create-exclusive semantics; no command may overwrite an earlier run.
- Runner-owned acceptance scripts and their SHA-256 values are immutable and outside all model patch allowlists.
- The accepted experiment diff contains only recorded DeepSeek coder/fixer patches plus deterministic `gofmt`; direct human functional edits invalidate that run.
- Dangerous cleanup accepts only run-owned absolute paths below the configured runtime root.
- Existing CLI commands and historical final evidence remain compatible and are not overwritten.
- Unit-test fakes are allowed for local TDD but are never accepted as experiment evidence.

## File Structure

The implementation adds one focused package and small extensions to existing modules:

```text
internal/codebasedag/
  types.go             versioned run, node, call, test, and evidence schemas
  manifest.go          tracked-file Git manifest and line/hash gates
  runstore.go          create-exclusive artifacts, JSONL, and final hashes
  model.go             strict DeepSeek wrapper and call-budget enforcement
  patch.go             schema decoding, patch allowlists, and attribution
  process.go           process-neutral worker/runtime interfaces
  process_linux.go     stopped-worker, cgroup, sampler, scheduler lifecycle
  process_other.go     explicit unsupported-platform errors
  acceptance.go        immutable runner-owned acceptance executor
  runner.go            DAG state machine and orchestration
  validate.go          strict final evidence validation
  prompts.go           role-specific, schema-constrained prompt builders
  acceptance/
    resource_real.sh
    context_real.sh
    review_final_strict.sh
cmd/aort-code-worker/main.go  one-node UDS/Gateway worker
```

Existing files change only where their APIs must be extended:

- `internal/llm/router.go`, `internal/llm/deepseek_provider.go`: actual model/request ID/total tokens, model listing, bounded request configuration.
- `internal/dag/dag.go`: missing-dependency and cycle validation.
- `internal/workspace/manager.go`: seeded full-tree snapshots and require-real OverlayFS.
- `internal/ipc/shm/*`: reusable cross-process memfd/SCM_RIGHTS transport with counters.
- `internal/capsule/manager.go`: per-agent limits without changing current defaults.
- `internal/checkpoint/checkpoint.go`: completed-node output hashes and LLM call IDs.
- `internal/worker/launcher.go`, `internal/worker/protocol.go`: extra files, stopped startup, and code-node reports.
- `cmd/aortctl/main.go`: `scenario codebase-dag` and `evidence codebase-dag` entry points.
- `scripts/competition_verify_real.sh`: final codebase-DAG verification step.

---

### Task 1: Strict DeepSeek Metadata and Model Gate

**Files:**
- Modify: `internal/llm/router.go`
- Modify: `internal/llm/deepseek_provider.go`
- Modify: `internal/llm/deepseek_provider_live_test.go`
- Test: `internal/llm/deepseek_provider_test.go`

**Interfaces:**
- Consumes: existing `llm.Provider`, `llm.Request`, and `DeepSeekConfig`.
- Produces: `DeepSeekProvider.ValidateModel(context.Context) error`; `Response.RequestID`; `Usage.TotalTokens`; `DeepSeekConfig.MaxTokens` and `Temperature`.

- [ ] **Step 1: Write failing HTTP contract tests**

Add tests that use `httptest.Server`, assert the request model/max-token/temperature fields, and return actual response metadata:

```go
func TestDeepSeekProviderRecordsActualModelRequestIDAndTotalTokens(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-secret" {
			t.Fatal("missing bearer header")
		}
		var body struct {
			Model       string  `json:"model"`
			MaxTokens   int     `json:"max_tokens"`
			Temperature float64 `json:"temperature"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.Model != "deepseek-v4-flash" || body.MaxTokens != 4096 || body.Temperature != 0 {
			t.Fatalf("request = %#v", body)
		}
		_, _ = io.WriteString(w, `{"id":"call-123","model":"deepseek-v4-flash","choices":[{"message":{"content":"{\"status\":\"ok\"}"}}],"usage":{"prompt_tokens":11,"completion_tokens":7,"total_tokens":18}}`)
	}))
	defer server.Close()

	provider := NewDeepSeekProvider(DeepSeekConfig{
		APIKey: "test-secret", BaseURL: server.URL, Model: "deepseek-v4-flash",
		MaxTokens: 4096, Temperature: 0,
	})
	resp, usage, err := provider.Complete(context.Background(), Request{Prompt: "return json"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.RequestID != "call-123" || resp.Model != "deepseek-v4-flash" || usage.TotalTokens != 18 {
		t.Fatalf("resp=%#v usage=%#v", resp, usage)
	}
}
```

Add a `/models` test where the required ID is present and a second test where it is absent; the latter must return `required model "deepseek-v4-flash" is unavailable`.

- [ ] **Step 2: Run the tests and verify RED**

Run: `GOCACHE=/private/tmp/aort-gocache-task1 go test ./internal/llm -run 'TestDeepSeekProviderRecords|TestDeepSeekProviderValidateModel' -count=1`

Expected: FAIL because response/request metadata fields and `ValidateModel` do not exist.

- [ ] **Step 3: Extend the public structs without changing old defaults**

Use these exact fields:

```go
type Response struct {
	RequestID         string `json:"request_id,omitempty"`
	Text              string `json:"text"`
	Provider          string `json:"provider"`
	Model             string `json:"model,omitempty"`
	RequestedProvider string `json:"requested_provider,omitempty"`
	Fallback          bool   `json:"fallback"`
	FallbackFrom      string `json:"fallback_from,omitempty"`
	FallbackReason    string `json:"fallback_reason,omitempty"`
	EvidenceMode      string `json:"evidence_mode"`
}

type Usage struct {
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
	TotalTokens      int    `json:"total_tokens"`
	CachedTokens     int    `json:"cached_tokens"`
	PromptMS         int64  `json:"prompt_ms"`
	TTFTMS           int64  `json:"ttft_ms"`
	TotalMS          int64  `json:"total_ms"`
	Mode             string `json:"mode"`
}

type DeepSeekConfig struct {
	APIKey      string
	BaseURL     string
	Model       string
	MaxTokens   int
	Temperature float64
	Client      *http.Client
}
```

Default `MaxTokens` to 4096. Decode top-level API `id` and `model`; do not replace the returned model with the configured model. Decode `total_tokens` and fail if ID, model, choices, prompt tokens, completion tokens, or total tokens are missing in strict real runs.

- [ ] **Step 4: Add the authenticated model-list gate**

Implement `ValidateModel` as a GET to `<baseURL>/models`, using the same bearer header and client timeout. Decode only `data[].id`, compare exact strings, limit the body to 8 MiB with `io.LimitReader`, and return an error without embedding response bodies or headers.

- [ ] **Step 5: Replace the live fallback test with a strict opt-in live test**

Rename the test to `TestDeepSeekProviderLiveFromEnv`. When the key is absent, keep `t.Skip`; when present, register only DeepSeek, set no fallback, require exact model, `real-api`, positive usage, and `Fallback == false`.

- [ ] **Step 6: Run focused and package tests**

Run: `GOCACHE=/private/tmp/aort-gocache-task1 go test ./internal/llm -count=1`

Expected: PASS; live test is SKIP when the environment key is absent.

- [ ] **Step 7: Commit**

```bash
git add internal/llm/router.go internal/llm/deepseek_provider.go internal/llm/deepseek_provider_test.go internal/llm/deepseek_provider_live_test.go
git commit -m "feat: enforce strict DeepSeek response evidence"
```

### Task 2: Immutable Run Store and 20,000-Line Source Manifest

**Files:**
- Create: `internal/codebasedag/types.go`
- Create: `internal/codebasedag/manifest.go`
- Create: `internal/codebasedag/manifest_test.go`
- Create: `internal/codebasedag/runstore.go`
- Create: `internal/codebasedag/runstore_test.go`

**Interfaces:**
- Consumes: clean workload path and Git executable.
- Produces: `BuildSourceManifest(context.Context, string) (SourceManifest, []SeedFile, error)`; `NewRunStore(string, string) (*RunStore, error)`; create-exclusive JSON/JSONL/artifact hashing.

- [ ] **Step 1: Define versioned evidence types**

Start `types.go` with these exact contracts:

```go
package codebasedag

const SchemaVersion = "codebase-dag/v1"

type SeedFile struct {
	Path string      `json:"path"`
	Mode os.FileMode `json:"-"`
	Data []byte      `json:"-"`
}

type SourceFile struct {
	Path         string `json:"path"`
	SHA256       string `json:"sha256"`
	Bytes        int64  `json:"bytes"`
	PhysicalLines int   `json:"physical_lines"`
	NonblankLines int   `json:"nonblank_lines"`
}

type SourceManifest struct {
	SchemaVersion  string       `json:"schema_version"`
	GitCommit      string       `json:"git_commit"`
	GitDirty       bool         `json:"git_dirty"`
	TreeHash       string       `json:"tree_hash"`
	PhysicalLines  int          `json:"physical_go_lines"`
	NonblankLines  int          `json:"nonblank_go_lines"`
	TrackedGoFiles int          `json:"tracked_go_files"`
	Files          []SourceFile `json:"files"`
}
```

- [ ] **Step 2: Write failing manifest tests**

Create a temporary Git repository with two tracked `.go` files, one ignored untracked `.go` file, and one tracked Markdown file. Assert only tracked Go files count, paths are sorted, dirty state is false before adding the untracked file and true afterward, final lines without a newline count once, and each hash matches its bytes.

Also add:

```go
func TestSourceManifestLargeCodeGate(t *testing.T) {
	manifest := SourceManifest{PhysicalLines: 20000, NonblankLines: 19999}
	if err := manifest.ValidateLargeCodebase(); err == nil {
		t.Fatal("nonblank threshold must fail")
	}
	manifest.NonblankLines = 20000
	if err := manifest.ValidateLargeCodebase(); err != nil {
		t.Fatal(err)
	}
}
```

- [ ] **Step 3: Implement manifest construction with structured Git commands**

Run `git -C <dir> rev-parse HEAD`, `git -C <dir> status --porcelain`, and `git -C <dir> ls-files -z`. Do not invoke a shell. Reject absolute, parent-escaping, symlink, unreadable, non-regular, or duplicate paths. Count physical and nonblank lines with `bufio.Scanner`, raise its buffer to 1 MiB, and compute a tree hash over sorted `path:NUL:sha256` records. Return `SeedFile` values for all tracked regular files so the workspace task never copies `.env`, caches, or untracked secrets.

- [ ] **Step 4: Write failing run-store tests**

Test invalid run IDs (`""`, `"../escape"`, slash, backslash, whitespace), create the first store, verify the second `NewRunStore` call for the same ID fails with `run directory already exists`, verify `WriteJSON` uses create-exclusive mode, verify JSONL appends only inside the new run, and verify `FinalizeHashes` excludes its own hash index.

- [ ] **Step 5: Implement the run store**

Use this API:

```go
type RunStore struct {
	Root  string
	RunID string
	Dir   string
	mu    sync.Mutex
}

func NewRunStore(root, runID string) (*RunStore, error)
func (s *RunStore) WriteJSON(rel string, value any) error
func (s *RunStore) WriteBytes(rel string, data []byte, mode os.FileMode) error
func (s *RunStore) AppendJSONL(rel string, value any) error
func (s *RunStore) FinalizeHashes() (map[string]string, error)
```

Validate every relative path with `filepath.Rel`, reject symlinks, create parent directories below `s.Dir`, use `O_CREATE|O_EXCL` for immutable single artifacts, and serialize JSONL appends under `mu`.

- [ ] **Step 6: Run tests**

Run: `GOCACHE=/private/tmp/aort-gocache-task2 go test ./internal/codebasedag -run 'TestSourceManifest|TestRunStore' -count=1`

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/codebasedag/types.go internal/codebasedag/manifest.go internal/codebasedag/manifest_test.go internal/codebasedag/runstore.go internal/codebasedag/runstore_test.go
git commit -m "feat: add immutable large-codebase run store"
```

### Task 3: Validated DAG and Durable Node State

**Files:**
- Modify: `internal/dag/dag.go`
- Modify: `internal/dag/dag_test.go`
- Create: `internal/codebasedag/state.go`
- Create: `internal/codebasedag/state_test.go`

**Interfaces:**
- Consumes: existing `dag.Graph.Ready`.
- Produces: `Graph.Validate() error`, `Graph.Nodes() []string`, and `ExecutionState` with legal transitions.

- [ ] **Step 1: Write graph validation tests**

Cover duplicate node replacement, missing dependency, self-dependency, two-node cycle, stable sorted node IDs, and a valid fan-out/fan-in graph. The expected cycle error contains the participating node IDs.

- [ ] **Step 2: Run graph tests and verify RED**

Run: `GOCACHE=/private/tmp/aort-gocache-task3 go test ./internal/dag -count=1`

Expected: FAIL because `Validate` and `Nodes` do not exist.

- [ ] **Step 3: Implement deterministic validation**

Keep `AddNode` and `Ready` backward compatible. `Validate` first checks every dependency exists, then performs three-color DFS over sorted node IDs. `Nodes` returns a sorted copy. Do not expose the internal dependency map.

- [ ] **Step 4: Add the codebase-DAG state machine tests**

Use exact states and transitions:

```go
type NodeStatus string

const (
	NodePending   NodeStatus = "pending"
	NodeReady     NodeStatus = "ready"
	NodeRunning   NodeStatus = "running"
	NodeSucceeded NodeStatus = "succeeded"
	NodeFailed    NodeStatus = "failed"
	NodeReplaying NodeStatus = "replaying"
)
```

Allow `pending->ready->running->succeeded`, `running->failed`, and `failed->replaying->running`. Reject every terminal-to-nonterminal transition and record UTC timestamp, reason, output hash, and LLM call ID for each transition.

- [ ] **Step 5: Implement and test `ExecutionState`**

Provide `NewExecutionState(nodeIDs []string)`, `Transition(nodeID string, to NodeStatus, evidence TransitionEvidence) error`, `Completed() map[string]bool`, and `Snapshot() []NodeState`. Copy all returned maps/slices.

Run: `GOCACHE=/private/tmp/aort-gocache-task3 go test ./internal/dag ./internal/codebasedag -run 'TestGraph|TestExecutionState' -count=1`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/dag/dag.go internal/dag/dag_test.go internal/codebasedag/state.go internal/codebasedag/state_test.go
git commit -m "feat: validate and persist DAG node state"
```

### Task 4: Full-Tree Real OverlayFS Workspaces

**Files:**
- Modify: `internal/workspace/manager.go`
- Modify: `internal/workspace/manager_test.go`
- Create: `internal/workspace/manager_linux_test.go`
- Create: `internal/codebasedag/workspace.go`
- Create: `internal/codebasedag/workspace_test.go`

**Interfaces:**
- Consumes: `[]codebasedag.SeedFile` from Task 2.
- Produces: `workspace.Config.RequireOverlay`; `Manager.CreateBaseSnapshotFiles`; `codebasedag.WorkspaceRuntime` implementing the Gateway workspace interface.

- [ ] **Step 1: Write failing require-real and seeded-snapshot tests**

Add `RequireOverlay bool` to the expected config. Assert `ForceDegraded:true, RequireOverlay:true` returns an error and creates no registered workspace. Seed files must preserve executable bits, reject path escapes and symlinks, and never include a file absent from the manifest.

- [ ] **Step 2: Implement seeded snapshots**

Add a workspace-local seed type to avoid an import cycle:

```go
type SeedFile struct {
	Path string
	Mode os.FileMode
	Data []byte
}

func (m *Manager) CreateBaseSnapshotFiles(taskID string, files []SeedFile) (Snapshot, error)
```

Sort paths, reject duplicates, write only regular files below `tasks/<task>/base`, preserve `mode.Perm()`, and record the base path in `m.tasks`. Do not accept `.git`, `.env`, private-key suffixes, or symlinks. The codebase-DAG adapter converts its manifest `SeedFile` values to workspace values.

- [ ] **Step 3: Enforce `RequireOverlay` after mount attempt**

In `create`, when the mount probe or `mountOverlay` fails and `RequireOverlay` is true, remove the agent runtime root, do not register a degraded workspace, and return `real overlayfs required: <reason>`. Existing callers with the zero value retain degraded-copy behavior.

- [ ] **Step 4: Add Linux integration coverage**

In `manager_linux_test.go`, skip unless UID 0 and `/proc/filesystems` lists overlay. Create a seeded snapshot, prepare two agents, assert both are `ModeOverlayFS` and mountpoints, change one merged file, assert the other merged file and both lowerdirs remain unchanged, then destroy both and assert mountinfo no longer lists their merged paths.

- [ ] **Step 5: Implement the Gateway adapter**

`internal/codebasedag/workspace.go` wraps `workspace.Manager` and implements:

```go
func (r *WorkspaceRuntime) WorkspaceDir(agentID string) (string, error)
func (r *WorkspaceRuntime) Commit(agentID string) error
func (r *WorkspaceRuntime) Rollback(agentID string) error
func (r *WorkspaceRuntime) Destroy(agentID string) error
```

It rejects any status whose mode/evidence is not `overlayfs`/`real-overlayfs`.

- [ ] **Step 6: Run tests**

Run: `GOCACHE=/private/tmp/aort-gocache-task4 go test ./internal/workspace ./internal/codebasedag -run 'Test.*Workspace|Test.*Overlay' -count=1`

Expected locally: portable tests PASS; root-only test may SKIP. On openEuler it must PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/workspace/manager.go internal/workspace/manager_test.go internal/workspace/manager_linux_test.go internal/codebasedag/workspace.go internal/codebasedag/workspace_test.go
git commit -m "feat: require real OverlayFS for codebase workspaces"
```

### Task 5: Cross-Process memfd/mmap and SCM_RIGHTS Transport

**Files:**
- Create: `internal/ipc/shm/session.go`
- Create: `internal/ipc/shm/session_linux.go`
- Create: `internal/ipc/shm/session_other.go`
- Create: `internal/ipc/shm/session_linux_test.go`
- Modify: `internal/ipc/shm/shm.go`
- Modify: `internal/ipc/shm/shm_test.go`

**Interfaces:**
- Consumes: shared CVM page bytes and child `exec.Cmd`.
- Produces: reusable `Publisher`, one `Recipient` per worker, `Receive`, and measured transfer counters.

- [ ] **Step 1: Define transport evidence**

```go
type TransferStats struct {
	EvidenceMode       string `json:"evidence_mode"`
	PayloadSHA256      string `json:"payload_sha256"`
	PayloadBytesWritten int64 `json:"payload_bytes_written"`
	ControlBytesSent   int64  `json:"control_bytes_sent"`
	MappedBytes        int64  `json:"mapped_bytes"`
	Recipients         int    `json:"recipients"`
	MemfdCreated       bool   `json:"memfd_created"`
	FDPassingSucceeded bool   `json:"fd_passing_succeeded"`
	MmapSucceeded      bool   `json:"mmap_succeeded"`
	HashValidated      bool   `json:"hash_validated"`
	DurationMicros     int64  `json:"duration_micros"`
}

type Publisher interface {
	NewRecipient() (*os.File, string, error)
	Send(recipientID string) (TransferStats, error)
	Close() error
}

func NewPublisher(payload []byte) (Publisher, error)
func Receive(socketFD int, expectedSHA256 string) ([]byte, TransferStats, error)
```

- [ ] **Step 2: Write a real helper-process test**

Spawn the current test binary with `-test.run=TestSHMReceiverHelper`, pass the recipient socket through `cmd.ExtraFiles` as FD 3, call `Publisher.Send`, and have the helper call `Receive(3, expectedHash)`. The helper writes only its `TransferStats` JSON to stdout. Assert different PID, one payload write, successful FD passing/mmap/hash, and mapped bytes equal payload length.

- [ ] **Step 3: Run the test and verify RED**

Run on Linux: `GOCACHE=/private/tmp/aort-gocache-task5 go test ./internal/ipc/shm -run 'TestPublisherTransfersToChildProcess' -count=1`

Expected: FAIL because the session API does not exist.

- [ ] **Step 4: Implement the Linux publisher**

Create one memfd, `Ftruncate`, map writable once, copy payload once, `Msync` if available, unmap, and retain the FD. Each recipient gets a Unix `SOCK_SEQPACKET` socketpair. `Send` uses `Sendmsg` with one data byte plus `syscall.UnixRights(memfd)`. `Receive` parses exactly one received FD, maps read-only, hashes bytes, copies only when returning to the caller, and closes/unmaps every resource. Count actual payload writes, data/control bytes, mapped bytes, recipients, and microseconds. Any cleanup failure returns an error.

- [ ] **Step 5: Implement explicit unsupported behavior**

The non-Linux file returns `real cross-process memfd/mmap requires linux` and never emits degraded success. Keep the existing smoke API backward compatible, but implement its Linux path through the new session primitives so both paths share instrumentation.

- [ ] **Step 6: Run transport tests**

Run: `GOCACHE=/private/tmp/aort-gocache-task5 go test ./internal/ipc/shm -count=1`

Expected: PASS on Linux; helper test SKIP on non-Linux while unsupported-path tests PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/ipc/shm/session.go internal/ipc/shm/session_linux.go internal/ipc/shm/session_other.go internal/ipc/shm/session_linux_test.go internal/ipc/shm/shm.go internal/ipc/shm/shm_test.go
git commit -m "feat: add real cross-process shared context transport"
```

### Task 6: Stopped Workers, Per-Agent Capsules, Sampling, and Replay

**Files:**
- Modify: `internal/capsule/manager.go`
- Modify: `internal/capsule/manager_test.go`
- Modify: `internal/checkpoint/checkpoint.go`
- Modify: `internal/checkpoint/checkpoint_test.go`
- Modify: `internal/worker/launcher.go`
- Modify: `internal/worker/launcher_test.go`
- Modify: `internal/worker/protocol.go`
- Create: `internal/codebasedag/process.go`
- Create: `internal/codebasedag/process_linux.go`
- Create: `internal/codebasedag/process_other.go`
- Create: `internal/codebasedag/process_test.go`
- Create: `cmd/aort-code-worker/main.go`
- Create: `cmd/aort-code-worker/main_test.go`

**Interfaces:**
- Consumes: real workspace, shared-context recipient FD, capsule manager, sampler, scheduler, checkpoint store, and Gateway UDS.
- Produces: start-stopped worker lifecycle, per-agent limits, code-node reports, output-hash replay without duplicate API calls.

- [ ] **Step 1: Add failing per-agent limit tests**

Define:

```go
type Limits struct {
	MemoryMax string `json:"memory_max"`
	PidsMax   string `json:"pids_max"`
	CPUMax    string `json:"cpu_max"`
}

func (m *Manager) PrepareWithLimits(agentID string, pid int, limits Limits) (Runtime, error)
```

Test that explicit limits write the three control files and `Prepare` still uses config defaults. Reject empty/negative/malformed values before creating the child cgroup.

- [ ] **Step 2: Extend checkpoint schema compatibly**

Add:

```go
type NodeOutputCheckpoint struct {
	NodeID      string `json:"node_id"`
	SHA256      string `json:"sha256"`
	ArtifactPath string `json:"artifact_path"`
	LLMCallID   string `json:"llm_call_id"`
}

type Snapshot struct {
	// existing fields remain unchanged
	NodeOutputs map[string]NodeOutputCheckpoint `json:"node_outputs,omitempty"`
}
```

Test save/load preserves the map and old snapshot JSON without `node_outputs` still decodes.

- [ ] **Step 3: Extend worker launch without changing existing callers**

Add optional `Dir`, `Env`, `Args`, and `ExtraFiles` to `worker.Spec`. Preserve current generated flags first, append `Spec.Args`, use `os.Environ()` plus `Spec.Env`, and attach `ExtraFiles`. Add a `StartedPID` assertion in the launcher test.

- [ ] **Step 4: Define code-worker protocol reports**

Add `MessageCodeNodeResult = "code-node.result"`. Its payload contains `node_id`, `status`, `output_json`, `output_sha256`, `llm_call_id`, `provider`, `model`, `prompt_tokens`, `completion_tokens`, and `total_tokens`. The registry must retain the report through a callback without placing raw output in generic event logs.

- [ ] **Step 5: Implement `aort-code-worker`**

The worker performs this exact order:

1. parse required agent/task/node/role/socket/shared-FD/expected-hash/private-prompt flags;
2. call `SIGSTOP` on itself when `--self-stop` is set;
3. receive and hash-verify shared context from FD 3;
4. connect to the existing UDS and register;
5. call `llm.call` through the Gateway with provider `deepseek` and the combined schema-constrained prompt;
6. validate response metadata is real DeepSeek and exact model;
7. hash the output text and send one `code-node.result` report;
8. send `agent.report` with terminal state and exit.

The process never reads the API key because the Gateway owns the HTTP provider.

- [ ] **Step 6: Implement Linux start-stopped lifecycle**

`ProcessRuntime.StartPrepared` starts the worker, polls `/proc/<pid>/status` until state `T` with a two-second deadline, calls `PrepareWithLimits`, enriches the AVP with `CgroupSampler`, records pressure, prepares the shared-memory send, then sends `SIGCONT`. A process that runs or exits before capsule attachment fails the node.

For coder fan-out, start all three workers stopped, attach/sample all, build ready AVPs with CVM page tables, call the existing resource-aware scheduler, and continue at most two selected PIDs. Resume the third only after one slot completes.

- [ ] **Step 7: Implement fail-once replay without a second model call**

After the designated coder report is written and checkpointed, kill its capsule before acknowledgement. Save output artifact path/hash/call ID in `NodeOutputs`. Start a replacement worker with `--replay-output <artifact>`; it verifies the hash and reports the checkpointed output without invoking `llm.call`. Assert call count and call IDs are unchanged, then destroy both old and replacement cgroups.

- [ ] **Step 8: Run tests**

Run: `GOCACHE=/private/tmp/aort-gocache-task6 go test ./internal/capsule ./internal/checkpoint ./internal/worker ./internal/codebasedag ./cmd/aort-code-worker -count=1`

Expected locally: portable tests PASS; real stopped/cgroup process test SKIP outside Linux root. On openEuler it must PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/capsule/manager.go internal/capsule/manager_test.go internal/checkpoint/checkpoint.go internal/checkpoint/checkpoint_test.go internal/worker/launcher.go internal/worker/launcher_test.go internal/worker/protocol.go internal/codebasedag/process.go internal/codebasedag/process_linux.go internal/codebasedag/process_other.go internal/codebasedag/process_test.go cmd/aort-code-worker/main.go cmd/aort-code-worker/main_test.go
git commit -m "feat: run code DAG workers in sampled cgroup capsules"
```

### Task 7: Schema-Constrained Prompts and Safe Patch Attribution

**Files:**
- Create: `internal/codebasedag/prompts.go`
- Create: `internal/codebasedag/prompts_test.go`
- Create: `internal/codebasedag/model.go`
- Create: `internal/codebasedag/model_test.go`
- Create: `internal/codebasedag/patch.go`
- Create: `internal/codebasedag/patch_test.go`

**Interfaces:**
- Consumes: code-node output text and exact DeepSeek metadata from Task 1/6.
- Produces: typed role outputs, call ledger with hard budget, patch allowlist validation, and deterministic attribution.

- [ ] **Step 1: Define node output schemas**

Add these types to `types.go`:

```go
type NodeKind string

const (
	KindPlanner   NodeKind = "planner"
	KindCoder     NodeKind = "coder"
	KindTester    NodeKind = "tester"
	KindReviewer  NodeKind = "reviewer"
	KindFixer     NodeKind = "fixer"
	KindFinalizer NodeKind = "finalizer"
)

type PlanOutput struct {
	SchemaVersion string     `json:"schema_version"`
	NodeID        string     `json:"node_id"`
	Tasks         []PlanTask `json:"tasks"`
	Risks         []string   `json:"risks"`
	Commands      [][]string `json:"commands"`
}

type CoderOutput struct {
	SchemaVersion string   `json:"schema_version"`
	NodeID        string   `json:"node_id"`
	Summary       string   `json:"summary"`
	Patch         string   `json:"patch"`
	ChangedFiles  []string `json:"changed_files"`
	Tests         [][]string `json:"tests"`
}

type ReviewOutput struct {
	SchemaVersion string   `json:"schema_version"`
	NodeID        string   `json:"node_id"`
	Verdict       string   `json:"verdict"`
	Blocking      []string `json:"blocking_findings"`
	NonBlocking   []string `json:"non_blocking_findings"`
}

type FinalOutput struct {
	SchemaVersion string   `json:"schema_version"`
	NodeID        string   `json:"node_id"`
	Status        string   `json:"status"`
	Summary       string   `json:"summary"`
	Limitations   []string `json:"limitations"`
}
```

`PlanTask` contains `id`, `owner`, `dependencies`, `files`, and `acceptance`. Tester uses `ReviewOutput` with verdict `pass|fix`; fixer uses `CoderOutput`.

- [ ] **Step 2: Write strict-decoder tests**

Test valid JSON, Markdown fences, trailing data, unknown fields, wrong schema version, wrong node ID, empty patch, duplicate changed files, and a patch file not listed in `changed_files`. Only one bare JSON object is accepted; fences and trailing text fail.

- [ ] **Step 3: Implement role-specific prompt builders**

Each prompt starts with the node ID, exact model-independent task, owned-file allowlist, immutable acceptance paths, shared context hash, private source excerpts, and a complete JSON shape. End every prompt with:

```text
Return exactly one JSON object. Do not use Markdown fences. Do not include secrets,
authorization headers, environment variables, binary patches, absolute paths, or
files outside the allowlist. A textual claim cannot override command evidence.
```

Tests assert the prompt contains the ticket, schema, exact allowlist, context hash, and no API key value supplied through a test environment.

- [ ] **Step 4: Implement the call ledger**

```go
type CallRecord struct {
	CallID          string `json:"call_id"`
	NodeID          string `json:"node_id"`
	Role            string `json:"role"`
	Provider        string `json:"provider"`
	RequestedModel  string `json:"requested_model"`
	ActualModel     string `json:"actual_model"`
	EvidenceMode    string `json:"evidence_mode"`
	Fallback        bool   `json:"fallback"`
	PromptTokens    int    `json:"prompt_tokens"`
	CompletionTokens int   `json:"completion_tokens"`
	TotalTokens     int    `json:"total_tokens"`
	DurationMS      int64  `json:"duration_ms"`
	OutputSHA256    string `json:"output_sha256"`
	Status          string `json:"status"`
	Error           string `json:"error,omitempty"`
}

type CallLedger struct {
	RequiredModel string
	MaxCalls      int
	Records       []CallRecord
}
```

Use `Begin(nodeID, role string) (attemptID string, error)` to reserve and count
an attempt before the HTTP request, and `Finish(attemptID string, record
CallRecord) error` to close it exactly once. `Begin` rejects the eleventh
attempt. `Finish` rejects any provider other than `deepseek`, model mismatch,
evidence other than `real-api`, fallback, nonpositive usage, duplicate API call
ID, or missing output hash. `Fail(attemptID string, err error)` closes a failed
attempt with a sanitized error. Failed API attempts therefore consume the
ten-call budget without needing fabricated provider metadata.

- [ ] **Step 5: Define and test patch policies**

```go
type PatchPolicy struct {
	NodeID          string
	AllowedFiles    map[string]struct{}
	ImmutableFiles  map[string]string
	MaxBytes        int
}

type PatchRecord struct {
	NodeID       string   `json:"node_id"`
	SHA256       string   `json:"sha256"`
	Bytes        int      `json:"bytes"`
	ChangedFiles []string `json:"changed_files"`
	SourceCallID string   `json:"source_call_id"`
}
```

Reject absolute/parent paths, CRLF ambiguity, `GIT binary patch`, submodules, symlinks, mode-only changes, deletion/renaming of immutable tests, files outside the exact allowlist, patches over 256 KiB, empty hunks, and declared/actual file mismatches. Extract paths only from paired `--- a/path` and `+++ b/path` headers and require normalized slash paths.

- [ ] **Step 6: Run tests**

Run: `GOCACHE=/private/tmp/aort-gocache-task7 go test ./internal/codebasedag -run 'TestPrompt|TestDecode|TestCallLedger|TestPatch' -count=1`

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/codebasedag/types.go internal/codebasedag/prompts.go internal/codebasedag/prompts_test.go internal/codebasedag/model.go internal/codebasedag/model_test.go internal/codebasedag/patch.go internal/codebasedag/patch_test.go
git commit -m "feat: constrain and attribute DeepSeek DAG patches"
```

### Task 8: End-to-End Codebase DAG Runner

**Files:**
- Create: `internal/codebasedag/runner.go`
- Create: `internal/codebasedag/runner_test.go`
- Create: `internal/codebasedag/preflight.go`
- Create: `internal/codebasedag/preflight_test.go`
- Create: `internal/codebasedag/evidence.go`
- Test: `internal/codebasedag/evidence_test.go`

**Interfaces:**
- Consumes: all Task 1-7 interfaces.
- Produces: `Run(context.Context, Config, Dependencies) (Summary, error)` and immutable raw evidence.

- [ ] **Step 1: Define runner configuration and dependency boundary**

```go
type Config struct {
	WorkloadDir   string
	OutRoot       string
	RunID         string
	Ticket        string
	Provider      string
	Model         string
	WorkerCommand string
	MaxCalls      int
	MaxConcurrent int
	NodeTimeout   time.Duration
	RunTimeout    time.Duration
}

type Dependencies struct {
	WorkspaceFactory func(root string) (*WorkspaceRuntime, error)
	ProcessFactory   func(ProcessConfig) (ProcessController, error)
	CheckpointStore  *checkpoint.Store
	Clock            func() time.Time
}

func Run(ctx context.Context, cfg Config, deps Dependencies) (Summary, error)
```

Production defaults are provider `deepseek`, model `deepseek-v4-flash`, max calls 10, max concurrency 2, node timeout 180 seconds, and run timeout 45 minutes. Reject every other provider/model and values outside 7-10 calls or 1-2 concurrency.

- [ ] **Step 2: Write preflight table tests**

Cover wrong OS, non-root, wrong cgroup filesystem, absent key, dirty workload, either line threshold below 20,000, failed baseline tests, acceptance scripts unexpectedly passing baseline, unavailable exact model, non-real Overlay probe, and non-real shared-memory probe. Each case returns a named failed gate and writes `preflight.json` before returning.

- [ ] **Step 3: Implement production preflight**

Collect `/etc/os-release`, `statfs(/sys/fs/cgroup)`, UID, kernel, architecture, Go version, Git manifest, `DEEPSEEK_API_KEY` presence as a boolean only, DeepSeek model listing, `workspace.ProbeOverlay`, a cross-process shm helper, baseline `go test ./...`, and three immutable acceptance scripts expected to fail for their documented baseline reasons. A model availability GET does not increment completion-call count.

- [ ] **Step 4: Build the exact graph**

```go
graph.AddNode("preflight", nil)
graph.AddNode("planner", []string{"preflight"})
graph.AddNode("resource-coder", []string{"planner"})
graph.AddNode("context-coder", []string{"planner"})
graph.AddNode("evidence-coder", []string{"planner"})
graph.AddNode("integrate", []string{"resource-coder", "context-coder", "evidence-coder"})
graph.AddNode("tester", []string{"integrate"})
graph.AddNode("reviewer", []string{"tester"})
graph.AddNode("finalizer", []string{"reviewer"})
```

The fixer is a dynamically inserted dependency between reviewer and finalizer when tests fail or reviewer verdict is `fix`; allow at most two sequential fixer nodes.

- [ ] **Step 5: Implement context setup and worker execution**

Create and pin CVM pages for the ticket, source manifest, package list, acceptance contract, and review findings. Mount common pages plus private source pages per node. For each node, build the prompt, create a real Overlay workspace, create a shm recipient, start the worker stopped, attach/sample/schedule/continue it, receive exactly one report, validate metadata/output schema, append `llm_calls.jsonl`, and transition state.

Use ownership:

```text
resource-coder: internal/review/resource_isolation.go, its tests,
                internal/experiment/review_scenarios.go, cmd/aortctl/main.go
context-coder:  internal/review/context_sharing.go, its tests,
                internal/ipc/shm files, cmd/aortctl/main.go
evidence-coder: internal/review/review_final.go, its tests,
                internal/review/metrics.go, its tests, cmd/aortctl/main.go
```

Shared-file hunks are allowed but integration applies coder patches in stable node-ID order and sends any collision to a fixer; it never silently drops a hunk.

- [ ] **Step 6: Implement integration through the Gateway**

Prepare a fresh full-tree integration workspace. Through `tool.exec`, run `git init`, local fixed author config, `git add -A`, and a baseline commit with fixed author/committer date. Write each validated patch below the workspace, run `git apply --check`, then `git apply`. Run `gofmt` only on changed `.go` files, `git diff --check`, focused package tests, three acceptance scripts, and `go test ./...`. Preserve every stdout/stderr/exit code with secret redaction.

- [ ] **Step 7: Enforce tester/reviewer/fixer/finalizer gates**

Tester cannot pass if a command gate failed. Reviewer blocking findings force a fixer even when tests pass. Each fixer receives the accepted diff, failing command output, blocking findings, and the same allowlists. After each fixer patch, rerun all integration gates. Finalizer runs only after all gates pass and must return status `passed`; machine state remains authoritative.

- [ ] **Step 8: Write final evidence**

Write every artifact from the design, then `summary.json` last. `Summary` contains exact runner/workload commits, dirty flag, line counts, provider/model, completion call count, token totals, DAG states, patch records, changed files, test results, cgroup/workspace/shm/replay modes, cleanup result, human functional edit count fixed at zero, and `all_required_passed`. Call `FinalizeHashes` and write `ARTIFACT_SHA256.json` create-exclusively.

- [ ] **Step 9: Add orchestration tests with non-evidence fakes**

Use deterministic fake workers only to test dependency ordering, two-coder concurrency, third-coder scheduling, collision-to-fixer flow, failure short-circuit, ten-call bound, no finalizer on failed gates, checkpoint replay without duplicate call, cleanup on context cancellation, and immutable summary status. Mark summaries from fake dependencies `evidence_mode:test-only` and assert the strict validator rejects them.

- [ ] **Step 10: Run tests**

Run: `GOCACHE=/private/tmp/aort-gocache-task8 go test ./internal/codebasedag -count=1`

Expected: PASS.

- [ ] **Step 11: Commit**

```bash
git add internal/codebasedag/runner.go internal/codebasedag/runner_test.go internal/codebasedag/preflight.go internal/codebasedag/preflight_test.go internal/codebasedag/evidence.go internal/codebasedag/evidence_test.go
git commit -m "feat: orchestrate real DeepSeek codebase DAG"
```

### Task 9: Immutable Acceptance Scripts, Strict Validator, and CLI

**Files:**
- Create: `internal/codebasedag/acceptance.go`
- Create: `internal/codebasedag/acceptance_test.go`
- Create: `internal/codebasedag/acceptance/resource_real.sh`
- Create: `internal/codebasedag/acceptance/context_real.sh`
- Create: `internal/codebasedag/acceptance/review_final_strict.sh`
- Create: `internal/codebasedag/validate.go`
- Create: `internal/codebasedag/validate_test.go`
- Modify: `cmd/aortctl/main.go`
- Modify: `cmd/aortctl/main_test.go`
- Modify: `scripts/competition_verify_real.sh`
- Modify: `internal/verify/competition_script_test.go`

**Interfaces:**
- Consumes: workload CLI and completed run directory.
- Produces: embedded immutable acceptance scripts, `ValidateRun(string) error`, `scenario codebase-dag`, and `evidence codebase-dag`.

- [ ] **Step 1: Embed and hash acceptance scripts**

Use `//go:embed acceptance/*.sh`. `MaterializeAcceptance(dir)` writes each script with mode 0500, returns sorted path/hash records, and refuses an existing target. `VerifyAcceptance` rehashes before and after each run. Unit tests mutate one byte and require failure.

- [ ] **Step 2: Write the resource real-only script**

The script runs:

```bash
go run ./cmd/aortctl scenario resource-isolation \
  --mode aort-r --runs 1 --warmup 0 --timeout 90s --require-real \
  --out "$AORT_ACCEPT_OUT/resource"
```

Its Python assertion requires `success=true`, structured runtime evidence values `real-cgroup-v2`, `real-overlayfs`, and `real-runtime`, a real resource-aware scheduler decision, nonempty capsule paths beginning `/sys/fs/cgroup/aort.slice/`, positive PIDs, real samples, kill/destroy events, checkpoint/replay completion, unchanged lowerdir hash, no cross-agent contamination, and cleanup success. On the unremediated baseline, only unknown flag/missing structured evidence is an accepted failure reason.

- [ ] **Step 3: Write the context real-only script**

The script runs AORT-R mode at 50% shared ratio with six agents and requires `--require-real-shm`. Assert `real-shm-ipc`, memfd/mmap/FD/hash booleans, a recipient PID different from the runner PID, measured payload/control/mapped/materialized byte counters, derived saved bytes computed from raw counters, CVM shared/private pages, and positive Prefix Affinity hits. Reject a constant/unsupported measurement kind for required raw counters.

- [ ] **Step 4: Write the strict review-final script**

Create four fabricated `summary.json` files for resource isolation, context
sharing, real-agent-demo, and codebase-DAG containing only `scenario_id`,
`status:passed`, and degraded evidence; require `aortctl evidence review-final`
to exit nonzero. Then provide the real resource/context/real-agent-demo/
codebase-DAG summaries and require success only when expected modes, variants,
run counts, metrics, model calls, artifact hashes, and legacy final evidence
all validate. The script never edits source files.

- [ ] **Step 5: Implement strict final validation**

`ValidateRun` checks every requirement in design section 9: exact schema, exact model/provider per call, 7-10 successful real calls, zero fallback, both line counts, all required node states, three attributed coder patches, immutable acceptance hashes, test commands, actual cgroup/PID/path relationships, real workspace/shm evidence, one replay with unchanged hash/no extra call, zero human functional edits, cleanup, artifact existence/hash, and no status contradictions. Reject unknown evidence modes and any `passed` summary with failed steps.

- [ ] **Step 6: Add CLI parsing tests**

`scenario codebase-dag` accepts only provider `deepseek`, model
`deepseek-v4-flash`, required workload path, optional run ID, output root,
worker command, max calls 7-10, max concurrency 1-2, and timeouts. It never
exposes a key flag. `evidence codebase-dag --run <dir>` returns nonzero on
test-only evidence. Extend `real-agent-demo` with `--model` and
`--require-real`; in DeepSeek mode it registers no fallback, requires the exact
returned model and positive usage, and fails rather than skipping when
`--require-real` is present. Existing mock mode remains available only for the
portable command and is never invoked by this plan's real evidence path.

- [ ] **Step 7: Wire real verification**

Add a final required step to `competition_verify_real.sh` only when `AORT_CODEBASE_DAG_RUN` is set; invoke `go run ./cmd/aortctl evidence codebase-dag --run "$AORT_CODEBASE_DAG_RUN"`. Update the script structure test to require the command and environment gate.

- [ ] **Step 8: Run tests**

Run: `GOCACHE=/private/tmp/aort-gocache-task9 go test ./internal/codebasedag ./cmd/aortctl ./internal/verify -count=1`

Expected: PASS.

- [ ] **Step 9: Commit**

```bash
git add internal/codebasedag/acceptance.go internal/codebasedag/acceptance_test.go internal/codebasedag/acceptance internal/codebasedag/validate.go internal/codebasedag/validate_test.go cmd/aortctl/main.go cmd/aortctl/main_test.go scripts/competition_verify_real.sh internal/verify/competition_script_test.go
git commit -m "test: enforce real codebase DAG acceptance"
```

### Task 10: Local Integration Verification and Runner Release Commit

**Files:**
- No source edits are owned by this task. A failure returns to the owning Task
  1-9 and repeats that task's failing-test, implementation, and passing-test
  cycle before integration verification restarts.
- Create: `docs/review_remediation/CODEBASE_DAG_RUNBOOK.md`

**Interfaces:**
- Consumes: complete runner implementation.
- Produces: a clean, locally verified runner commit suitable for archive deployment.

- [ ] **Step 1: Run formatting and static diff checks**

Run:

```bash
gofmt -w internal/llm/router.go internal/llm/deepseek_provider.go \
  internal/llm/deepseek_provider_test.go internal/llm/deepseek_provider_live_test.go \
  internal/dag/dag.go internal/dag/dag_test.go internal/codebasedag/*.go \
  internal/workspace/manager.go internal/workspace/manager_test.go \
  internal/workspace/manager_linux_test.go internal/ipc/shm/*.go \
  internal/capsule/manager.go internal/capsule/manager_test.go \
  internal/checkpoint/checkpoint.go internal/checkpoint/checkpoint_test.go \
  internal/worker/launcher.go internal/worker/launcher_test.go \
  internal/worker/protocol.go cmd/aort-code-worker/*.go cmd/aortctl/*.go
git diff --check
```

Do not format generated workload copies or evidence.

Expected: no output from `git diff --check`.

- [ ] **Step 2: Run all Go tests with a writable cache**

Run: `GOCACHE=/private/tmp/aort-gocache-runner-all go test -count=1 ./...`

Expected: PASS; live DeepSeek and root-only Linux tests SKIP locally when prerequisites are absent.

- [ ] **Step 3: Run vet and portable competition verification**

Run: `GOCACHE=/private/tmp/aort-gocache-runner-vet go vet ./...`

Run: `bash scripts/competition_verify.sh`

Expected: both exit 0; portable results are not cited as real experiment evidence.

- [ ] **Step 4: Write the operator runbook**

Document exact prerequisites, safe runtime roots, key injection without shell-history exposure, clean workload materialization, runner command, evidence validator, cleanup checks, and the rule that an unknown `deepseek-v4-flash` model fails without substitution. Do not include an IP, key, password, or bearer example.

- [ ] **Step 5: Scan the tracked tree for secrets and generated binaries**

Run: `rg -n 'DEEPSEEK_API_KEY=.*sk-|Authorization: Bearer sk-|sk-[0-9a-f]{32,}|BEGIN (RSA |OPENSSH )?PRIVATE KEY' . --glob '!.git/**'`

Expected: no secret values. Environment variable names in source are allowed only without assigned secrets.

- [ ] **Step 6: Commit the verified runner**

```bash
git add docs/review_remediation/CODEBASE_DAG_RUNBOOK.md
git commit -m "docs: add real codebase DAG runbook"
```

Record `git rev-parse HEAD` as the runner commit in the deployment notes.

### Task 11: Huawei openEuler Real DeepSeek DAG Run

**Files:**
- Remote generated only: `/root/aort-r-runner`, `/root/aort-r-workload`, and run-owned `/root/aort-runtime-*` paths.
- Pull to: `experiments/results/codebase_dag/$RUN_ID/`

**Interfaces:**
- Consumes: committed runner archive and a user-provisioned runtime environment key.
- Produces: real DeepSeek patches and complete immutable Huawei evidence.

- [ ] **Step 1: Recheck the remote hard gates**

Run over SSH: OS release, `stat -fc %T /sys/fs/cgroup`, UID, Go version, CPU/RAM/disk, `/proc/filesystems` overlay entry, and DeepSeek network reachability. Expected: openEuler 24.03 LTS, `cgroup2fs`, UID 0, Go 1.22+, overlay present, and HTTP 401 without authorization.

- [ ] **Step 2: Ask the user to inject the key safely on the server**

The user runs in their own SSH terminal:

```bash
install -m 600 /dev/null /run/aort-deepseek.env
read -rsp 'DeepSeek API key: ' AORT_SECRET; printf '\n'
printf 'export DEEPSEEK_API_KEY=%q\n' "$AORT_SECRET" > /run/aort-deepseek.env
unset AORT_SECRET
```

Codex checks only `test -s /run/aort-deepseek.env` and mode `600`; it never prints or downloads the file.

- [ ] **Step 3: Deploy a clean committed runner archive**

Create `/private/tmp/aort-runner-$RUNNER_COMMIT.tgz` with `git archive`, upload it with `scp`, extract to a fresh `/root/aort-r-runner-$RUNNER_COMMIT`, and build `aortctl` plus `aort-code-worker`. No local dirty or untracked file enters the archive.

- [ ] **Step 4: Create the clean workload repository**

Extract the same archive into a fresh `/root/aort-r-workload-$RUNNER_COMMIT`, initialize Git, set repository-local test identity, add all files, and commit with fixed author/committer timestamp. Verify `git status --porcelain` is empty. The runner records this ephemeral workload commit and source tree hashes.

- [ ] **Step 5: Run the exact real DAG**

Generate a UTC run ID, source `/run/aort-deepseek.env` in the remote shell, set `DEEPSEEK_BASE_URL=https://api.deepseek.com` and `DEEPSEEK_MODEL=deepseek-v4-flash`, then run:

```bash
./bin/aortctl scenario codebase-dag \
  --provider deepseek \
  --model deepseek-v4-flash \
  --workload "/root/aort-r-workload-$RUNNER_COMMIT" \
  --ticket review-remediation \
  --worker-command ./bin/aort-code-worker \
  --max-calls 10 \
  --max-concurrent 2 \
  --node-timeout 180s \
  --run-timeout 45m \
  --run-id "$RUN_ID" \
  --out experiments/results/codebase_dag
```

Expected: exit 0 only if at least seven exact-model calls, three patches, tests, real OS evidence, replay, and cleanup all pass. Any other result remains a failed immutable run and is not renamed or overwritten.

- [ ] **Step 6: Validate remotely before pullback**

Run: `./bin/aortctl evidence codebase-dag --run "experiments/results/codebase_dag/$RUN_ID"`

Run cgroup/mount/process residue checks scoped to the run ID. Expected: validator exit 0 and no live residue.

- [ ] **Step 7: Pull the whole run directory and remove the temporary key file**

Use `scp -r` to the same local run-ID directory, then remotely delete only `/run/aort-deepseek.env`. Do not delete a failed run. Locally rerun the strict validator and compare `ARTIFACT_SHA256.json`.

- [ ] **Step 8: Commit only redacted evidence after validation**

Inspect every artifact for secrets and oversized/raw bodies. Stage the immutable run directory only after the secret scan returns no matches.

```bash
git add experiments/results/codebase_dag/$RUN_ID
git commit -m "evidence: add real DeepSeek large-codebase DAG run"
```

### Task 12: Apply DeepSeek Patches, Rerun Final Acceptance, Update Docs, and Push

**Files:**
- Modify from model patch: `internal/review/resource_isolation.go`, `internal/review/context_sharing.go`, `internal/review/review_final.go`, their tests, and any allowlisted integration files.
- Modify: `docs/review_remediation/FINAL_CHECKLIST.md`
- Modify: `docs/review_remediation/CHANGELOG.md`
- Modify: `docs/review_remediation/FINAL_REPORT.md`
- Modify: `docs/design/04_resource_isolation_design.md`
- Modify: `docs/design/05_context_sharing_design.md`
- Modify: `docs/design/08_results.md`
- Modify: `docs/design/09_real_agent_demo.md`
- Modify: `docs/defense/DATA_FOR_SLIDES.md`
- Modify: `docs/defense/DEMO_SCRIPT.md`

**Interfaces:**
- Consumes: validated `patches/accepted.diff` and immutable run evidence.
- Produces: final target commit, final openEuler evidence tied to that commit, corrected review documents, and pushed `codex/aort-r-upgrade`.

- [ ] **Step 1: Verify patch provenance before local apply**

Compare the accepted diff SHA-256 with `PatchRecord` and `ARTIFACT_SHA256.json`. Require `human_functional_edits=0`, three coder call IDs, exact model, and successful strict validator. Run `git apply --check` against the current branch. Any conflict is returned to a new real DeepSeek fixer run; do not hand-edit functional hunks.

- [ ] **Step 2: Apply and verify the model-produced diff**

Run `git apply`, `gofmt` on changed Go files, `git diff --check`, focused package tests, then `GOCACHE=/private/tmp/aort-gocache-final-model go test -count=1 ./...`. Expected: PASS. Confirm the local diff hash after deterministic formatting matches the recorded attribution rule.

- [ ] **Step 3: Perform code review without changing functional code**

Check the three original findings line-by-line: actual capsule/sampler/scheduler/kill/destroy/replay calls, actual shm transfer in AORT-R mode with measured counters, and strict evidence shape/mode/run/artifact validation. Any blocking issue is fed to a real fixer node and Task 11 is rerun with a new run ID.

- [ ] **Step 4: Commit the accepted model patch**

Stage only allowlisted model patch files and tests, inspect `git diff --cached --stat`, then commit:

```bash
git commit -m "feat: complete real review-remediation paths"
```

Record the final target commit hash.

- [ ] **Step 5: Redeploy the final target commit and rerun all real gates**

Deploy a new clean archive and run on Huawei openEuler:

```bash
go test -count=1 ./...
bash scripts/competition_verify_real.sh
go run ./cmd/aortctl scenario resource-isolation --mode all --warmup 3 --runs 20 --require-real --out experiments/results/review_remediation/resource_isolation/$FINAL_RUN_ID
go run ./cmd/aortctl scenario context-sharing --mode all --shared-ratio all --warmup 3 --runs 20 --require-real-shm --out experiments/results/review_remediation/context_sharing/$FINAL_RUN_ID
go run ./cmd/aortctl scenario real-agent-demo --provider deepseek --model deepseek-v4-flash --require-real --out experiments/results/review_remediation/real_agent_demo/$FINAL_RUN_ID
go run ./cmd/aortctl evidence final --out experiments/results/final/$FINAL_RUN_ID
go run ./cmd/aortctl evidence review-final --out experiments/results/review_final/$FINAL_RUN_ID
```

Set `AORT_CODEBASE_DAG_RUN` to the validated final codebase-DAG run when
invoking `competition_verify_real.sh`. For `resource-isolation --mode all`,
`--require-real` requires real cgroup/OverlayFS/sampler evidence for
`isolation-only` and `aort-r`; the deliberately unisolated baseline is labeled
`unisolated-baseline`, not `degraded`. For `context-sharing --mode all`,
`--require-real-shm` requires real shared-memory evidence for `shared-ipc` and
`aort-r`; `full-copy` is labeled `independent-copy-baseline`. Every required
command must exit 0; degraded or skipped outputs fail.

- [ ] **Step 6: Pull and validate final evidence**

Pull run-ID directories without overwriting history. Parse all JSON, verify CSV headers, report links, sample counts, modes, call/token counts, Git hashes, lowerdir hashes, capsule PID/path records, shm counters, and cleanup residue. Run the repository secret scan and require no secret values.

- [ ] **Step 7: Correct final documentation from actual fields only**

Replace current portable/degraded claims with the new real run IDs and exact measured fields. Preserve explicit boundaries: CVM is not model KV cache, memfd/mmap is not claimed as end-to-end zero-copy, cgroup plus OverlayFS is not a full VM/container sandbox, and single-host results do not imply distributed guarantees. Do not type performance percentages manually; generate them from the final summaries.

- [ ] **Step 8: Run final local verification**

Run `gofmt -l` over tracked Go files, `go test -count=1 ./...`, `go vet ./...`, portable verification, strict evidence validation, JSON parsing, CSV consistency, Markdown link checks, secret scan, and `git diff --check`. Record exact commands and exit codes in `FINAL_CHECKLIST.md`.

- [ ] **Step 9: Commit scoped final evidence and documents**

Stage only validated run evidence and the named review/design/defense documents. Preserve unrelated Dashboard and user-owned files. Inspect staged names and stats, then commit:

```bash
git commit -m "test: complete real DeepSeek openEuler verification"
```

- [ ] **Step 10: Push the requested branch**

Verify branch is `codex/aort-r-upgrade`, inspect final `git status --short`, and push:

```bash
git push origin codex/aort-r-upgrade
```

Report the runner commit, model-patch commit, final evidence commit, remote branch, all key command outcomes, 20,000-line counts, exact API call/token counts, and remaining capability boundaries. Mark the active goal complete only after the pushed commit and every strict real gate are independently verified.
