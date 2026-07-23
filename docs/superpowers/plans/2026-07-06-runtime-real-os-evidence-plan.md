# Runtime Real OS Evidence Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make capsule kill, resource-aware scheduling, workspace execution, and pressure fault evidence use real OS paths where available.

**Architecture:** Keep existing package boundaries and add narrow interfaces. Capsule owns cgroup operations, a new resource sampler owns cgroup/PSI reads, syscall gateway accepts a workspace runtime interface, and experiments/CLI expose pressure and probe evidence as JSON.

**Tech Stack:** Go 1.22, standard library, existing `internal/capsule`, `internal/scheduler`, `internal/workspace`, `internal/syscall`, `internal/experiment`, and `cmd/aortctl`.

## Global Constraints

- Use TDD: write failing tests before production code.
- Do not fake `real-*` evidence; degraded paths must include `fallback_reason`.
- Do not present resource-aware as fastest; pressure experiments prove risk reduction.
- Keep existing CLI commands working.
- Verify locally and rerun real evidence on Linux/root/openEuler before final completion.

---

### Task 1: Capsule Kill Evidence

**Files:**
- Modify: `internal/capsule/manager.go`
- Modify: `internal/capsule/manager_test.go`
- Modify: `internal/api/server.go`
- Modify: `internal/experiment/real_benchmark.go`

**Interfaces:**
- Produces: `capsule.KillResult{AgentID, KillMethod, EvidenceMode, FallbackReason}`
- Produces: `(*capsule.Manager).Kill(agentID string) (capsule.KillResult, error)`

- [ ] Write tests requiring real capsules to write `cgroup.kill` and return `kill_method=cgroup.kill`.
- [ ] Write tests requiring fallback capsules to return `kill_method=pid-signal-fallback`.
- [ ] Run `go test ./internal/capsule` and confirm the new tests fail.
- [ ] Implement `KillResult` and cgroup.kill-first logic.
- [ ] Update API and experiment call sites to include `kill_method` evidence.
- [ ] Run capsule/API/experiment tests.

### Task 2: Resource Sampler

**Files:**
- Create: `internal/resource/sampler.go`
- Create: `internal/resource/sampler_test.go`
- Modify: `internal/api/server.go`
- Modify: `internal/experiment/real_benchmark.go`

**Interfaces:**
- Produces: `type ResourceSampler interface { Sample(agent avp.AVP) (scheduler.ResourcePressure, error) }`
- Produces: `resource.NewCgroupSampler(procPressureRoot string)`

- [ ] Write tests with fixture cgroup files for `memory.current`, `memory.max`, `pids.current`, `pids.max`, `cpu.stat`, and PSI.
- [ ] Run `go test ./internal/resource` and confirm failure.
- [ ] Implement sampler with degraded fallback reasons on missing files.
- [ ] Use sampler in API pressure sampling and pressure experiments.
- [ ] Run resource/API/experiment tests.

### Task 3: E1 Pressure Experiment

**Files:**
- Modify: `internal/experiment/real_benchmark.go`
- Modify: `internal/experiment/resource_aware_test.go`
- Modify: `cmd/aortctl/main.go`
- Modify: `cmd/aortctl/main_test.go`

**Interfaces:**
- Produces: `RunE1PressureBenchmark(runs int, outDir string) (E1PressureReport, error)`
- CLI: `go run ./cmd/aortctl experiment e1-pressure --runs 5 --out experiments/results/e1_pressure`

- [ ] Write tests for metrics `selected_high_pressure_agent_count`, `avoided_high_pressure_agent_count`, and `resource_aware_reduced_risk`.
- [ ] Run targeted tests and confirm failure.
- [ ] Implement benchmark and CLI.
- [ ] Run targeted tests and command.

### Task 4: Tool Exec Workspace Runtime

**Files:**
- Modify: `internal/syscall/gateway.go`
- Modify: `internal/syscall/gateway_test.go`
- Modify: `internal/api/server.go`

**Interfaces:**
- Produces: `syscallgw.WorkspaceRuntime` with `WorkspaceDir`, `Commit`, `Rollback`, and `Destroy`.
- `tool.exec` uses workspace manager cwd, commits on `OK`, rolls back on `ERROR` or `TIMEOUT`.

- [ ] Write gateway tests with a fake workspace runtime tracking commit/rollback.
- [ ] Run tests and confirm failure.
- [ ] Implement workspace runtime injection and API adapter.
- [ ] Run syscall/API tests.

### Task 5: Workspace Probe

**Files:**
- Modify: `internal/workspace/manager.go`
- Modify: `internal/workspace/workspace_isolation_test.go`
- Modify: `cmd/aortctl/main.go`
- Modify: `cmd/aortctl/main_test.go`

**Interfaces:**
- Produces: `workspace.ProbeOverlay(root string) WorkspaceProbe`
- CLI: `go run ./cmd/aortctl workspace probe --out experiments/results/workspace_probe.json`

- [ ] Write tests for degraded probe fields on non-Linux/non-root fixtures.
- [ ] Run tests and confirm failure.
- [ ] Implement probe and CLI.
- [ ] Run targeted tests and command.

### Task 6: E2 Pressure Fault

**Files:**
- Modify: `internal/experiment/real_benchmark.go`
- Modify: `internal/experiment/resource_aware_test.go`
- Modify: `cmd/aortctl/main.go`
- Modify: `cmd/aortctl/main_test.go`

**Interfaces:**
- Produces: `RunE2PressureFault(runs int, outDir string) (E2PressureFaultReport, error)`
- CLI: `go run ./cmd/aortctl experiment e2-pressure-fault --runs 5 --out experiments/results/e2_pressure_fault`

- [ ] Write tests requiring `cascade_failure=false`, `recovery_success=true`, and `unaffected_agents_continued=true`.
- [ ] Run tests and confirm failure.
- [ ] Implement combined pressure/workspace fault scenario.
- [ ] Run targeted tests and command.

### Task 7: Verification and Server Evidence

**Files:**
- Modify docs/README/evidence outputs as needed.
- Modify `scripts/competition_verify.sh` only if new evidence should be included in final index.

- [ ] Run `gofmt`.
- [ ] Run `go test ./...`.
- [ ] Run new CLI commands locally.
- [ ] Sync to openEuler/root server and rerun real evidence commands.
- [ ] Pull back evidence, verify JSON, commit, push.
