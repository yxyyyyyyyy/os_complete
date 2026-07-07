# AORT-R Runtime Upgrade Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the DOCX upgrade plan without changing `main`: eBPF observer evidence, memfd/mmap shared-memory IPC, enhanced CVM memory management, deterministic replay, CLI integration, tests, and final evidence indexing.

**Architecture:** Add focused packages for each runtime feature and register them through `cmd/aortctl` and `internal/experiment`. Keep platform-specific OS behavior behind Linux/non-Linux files, and write degraded evidence with explicit `fallback_reason` whenever real support is unavailable. Preserve page-reference IPC and CVM as runtime context memory only; do not introduce local model execution or KV-cache claims.

**Tech Stack:** Go 1.22, standard library, platform build tags, existing `internal/experiment.WriteJSON` evidence conventions.

## Global Constraints

- Do not connect a local large model.
- Do not implement or claim real model KV Cache.
- CVM means Context Virtual Memory: page-level context reuse, materialization, compression, eviction, pin, and refcount.
- eBPF may report `real-ebpf` only after a Linux-capable observer path attaches and observes process runtime events; otherwise it must be `degraded`.
- page-reference IPC remains available; memfd/mmap is an additional shared-memory IPC mode.
- Do not call page-reference IPC kernel zero-copy.
- Every new smoke command writes JSON evidence.

---

### Task 1: eBPF Observer Smoke

**Files:**
- Create: `internal/observer/ebpf/observer.go`
- Create: `internal/observer/ebpf/observer_linux.go`
- Create: `internal/observer/ebpf/observer_other.go`
- Create: `internal/observer/ebpf/observer_test.go`
- Modify: `internal/experiment/experiment.go`
- Modify: `cmd/aortctl/main.go`

**Interfaces:**
- Produces: `ebpf.RunSmoke(ctx context.Context, outDir string) (ebpf.SmokeResult, error)`
- Produces evidence path: `<outDir>/ebpf_smoke.json`

- [ ] Write degraded fallback and event parser tests first.
- [ ] Implement non-Linux and unsupported Linux fallback with clear reasons.
- [ ] Implement a minimal Linux process-event probe path and cleanup reporting.
- [ ] Add `aortctl observer ebpf-smoke --out ...`.

### Task 2: memfd/mmap Shared-Memory IPC

**Files:**
- Create: `internal/ipc/shm/shm.go`
- Create: `internal/ipc/shm/shm_linux.go`
- Create: `internal/ipc/shm/shm_other.go`
- Create: `internal/ipc/shm/shm_test.go`
- Modify: `internal/experiment/experiment.go`
- Modify: `cmd/aortctl/main.go`

**Interfaces:**
- Produces: `shm.RunSmoke(outDir string) (shm.SmokeResult, error)`
- Produces evidence path: `<outDir>/ipc_shm_smoke.json`

- [ ] Write fallback, mmap data integrity, and cleanup tests first.
- [ ] Implement Linux memfd/mmap/fd-passing smoke.
- [ ] Preserve page-reference IPC as the fallback mode.
- [ ] Add `aortctl ipc shm-smoke --out ...`.

### Task 3: CVM Memory Manager

**Files:**
- Modify: `internal/cvm/store.go`
- Modify: `internal/cvm/store_test.go`
- Modify: `internal/experiment/experiment.go`
- Modify: `cmd/aortctl/main.go`

**Interfaces:**
- Produces: `cvm.MemoryConfig`, `(*Store).CompressColdPages`, `(*Store).EvictColdPages`, `(*Store).PinPage`, and enhanced `Stats`.
- Produces smoke evidence path: `<outDir>/cvm_memory_smoke.json`

- [ ] Write tests for refcount, pinned eviction protection, LRU eviction, compression/decompression, materialization correctness, and dedup saved bytes.
- [ ] Extend page metadata without breaking existing JSON fields.
- [ ] Add cold compression and LRU eviction over non-pinned/refcount-zero pages.
- [ ] Add `aortctl cvm memory-smoke --out ...`.

### Task 4: Replay Trace

**Files:**
- Modify: `internal/trace/recorder.go`
- Modify: `internal/trace/recorder_test.go`
- Create: `internal/replay/replay.go`
- Create: `internal/replay/replay_test.go`
- Modify: `internal/experiment/experiment.go`
- Modify: `cmd/aortctl/main.go`

**Interfaces:**
- Produces: `trace.TraceEvent`, `trace.WriteTrace`, `trace.ReadTrace`.
- Produces: `replay.Run(tracePath, outDir string) (replay.Result, error)`
- Produces evidence path: `<outDir>/replay_result.json`

- [ ] Write trace read/write, replay success, divergence, and missing trace tests first.
- [ ] Implement deterministic mock replay over scheduler/admission/runtime state events.
- [ ] Add `aortctl replay --trace ... --out ...`.

### Task 5: experiment all and final evidence

**Files:**
- Modify: `internal/experiment/all_final.go`
- Modify: `cmd/aortctl/main_test.go`
- Modify: `internal/evidence/evidence.go`

**Interfaces:**
- Updates `experiment all` step list with eBPF, IPC shm, and CVM memory smoke.
- Updates `FINAL_EVIDENCE_INDEX.json` with `ebpf_observer`, `ipc_shm`, `cvm_memory`, and `replay`.

- [ ] Write CLI/evidence integration tests first.
- [ ] Register new smoke commands in `experiment all`.
- [ ] Add final evidence status and evidence mode summary fields.
- [ ] Run `go test ./...` and the requested smoke commands.
