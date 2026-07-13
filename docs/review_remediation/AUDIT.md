# P1 Repository Audit

**Audit date:** 2026-07-13  
**Branch at audit:** `codex/aort-r-upgrade` (before remediation commit)  
**Baseline:** `go test ./...` passed for every package; `gofmt -l` returned no files.

## Repository and command surface

- Language/runtime: Go module `aort-r`, Go 1.25.3 on the audit host; Vue 3/Vite dashboard under `dashboard/`.
- Runtime entry points: `cmd/aortd`, `cmd/aort-worker`, `cmd/aortctl`.
- Existing CLI groups: `experiment`, `evidence`, `demo`, `workspace`, `observer`, `ipc`, `cvm`, `replay`.
- `aortctl scenario resource-isolation` currently returns `unknown command "scenario"`.
- Existing one-click scripts: `scripts/competition_verify.sh`, `scripts/competition_verify_real.sh`, `scripts/smoke_*.sh`.
- Existing historical evidence is under `experiments/results/`; `experiments/results/final/FINAL_EVIDENCE_INDEX.json` and `FINAL_SUMMARY.md` are generated artifacts and must remain compatible.

## Capability classification

| Capability | Classification | Evidence/code |
|---|---|---|
| AVP lifecycle and worker process | real-runtime | `internal/avp`, `internal/worker`, `internal/demo/software_demo.go` |
| cgroup v2 capsule | real on supported Linux/openEuler; degraded elsewhere | `internal/capsule/manager.go`, `experiments/results/openeuler_smoke/capsule_real.json` |
| resource-aware scheduler | real-runtime; pressure can degrade | `internal/scheduler`, `internal/resource`, `internal/pressure`, E1 results |
| CVM page reuse/materialization | real-partial | `internal/cvm/store.go`, `experiments/results/cvm_memory/cvm_memory_smoke.json` |
| memfd/mmap IPC | real-shm-ipc on Linux, fallback elsewhere | `internal/ipc/shm`, `experiments/results/ipc_shm/ipc_shm_smoke.json` |
| page-reference Blackboard IPC | real-runtime | `internal/ipc/blackboard.go` |
| OverlayFS workspace isolation | real-overlayfs on supported root Linux; degraded-copy fallback | `internal/workspace/manager.go`, workspace evidence |
| checkpoint/replay and Timeline | real-runtime | `internal/checkpoint`, `internal/replay`, `internal/trace`, `experiments/results/replay/replay_result.json` |
| DeepSeek provider | optional real-api; mock fallback | `internal/llm`, `internal/experiment/deepseek_real.go` |
| eBPF observer | implemented optional path, current local evidence degraded | `internal/observer/ebpf`, `experiments/results/ebpf_smoke/ebpf_smoke.json` (`fallback_reason=attach failed: invalid argument`) |

## Measurement gaps

1. No formal `scenario` command joins the existing resource, workspace, scheduler, and evidence paths.
2. Existing E1/E2/E3 benchmarks have separate schemas and do not provide the required three-mode resource comparison or four-ratio context comparison.
3. There is no common `mean/stddev/min/max/P50/P95/success_rate` model with failure reasons and measured/derived/unsupported labels.
4. Existing software-real demo records runtime events but has no standalone six-role CLI, three-tool-call contract, or provider-separated network/runtime timing.
5. No `docs/review_remediation` audit/plan/matrix, problem-driven `docs/design` set, threat model, or `docs/defense` source package exists at audit time.
6. `aortctl evidence final` exists; `evidence review-final` and a review evidence index do not.

## Boundary findings

- Older design material contains phrases such as “零拷贝”, “内核级机制隔离”, and “操作系统一等公民”. Current `README.md` and `docs/EVIDENCE_MODE.md` already narrow several claims, but `V2.0.md` and the 2026-07-04 design still need boundary corrections.
- CVM evidence explicitly says it is not model KV Cache; this wording must be retained.
- eBPF evidence is degraded and must never be promoted to complete observation.
- DeepSeek evidence currently uses environment-only credentials and redacted summaries; no key is present in the repository.

## Preserve / refactor / add / downgrade

**Keep:** old CLI groups, final evidence schema, historical results, cgroup/OverlayFS smoke scripts, existing tests, and real openEuler archives.  
**Refactor:** add adapters around existing CVM/IPC/workspace/scheduler code and normalize experiment statistics.  
**Add:** three scenario commands, unified reports, review-final index, six-agent mock/real demo, design/threat/defense documents.  
**Downgrade wording:** AVP/Gateway/CVM/IPC/eBPF and security claims to the explicit boundaries in the design spec whenever evidence is partial or degraded.
