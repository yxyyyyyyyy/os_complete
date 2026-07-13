# AORT-R Review Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the approved review-remediation specification as compatible scenario commands, measured evidence, boundary-correct documentation, and final review artifacts.

**Architecture:** Add a focused `internal/review` package for common metrics, resource isolation, context sharing, and the six-agent demo. Keep command wiring in `cmd/aortctl` and evidence adapters in `internal/experiment`; preserve all existing commands and historical results.

**Tech Stack:** Go 1.22+ standard library, existing AORT-R CVM/IPC/workspace/capsule/scheduler packages, JSON/CSV/Markdown evidence.

## Global Constraints

- Scenario outputs use `schema_version`, `scenario_id`, `run_id`, seed, environment, evidence mode, raw artifacts, and explicit measured/derived/unsupported labels.
- Defaults are warmup=3 and measured runs=20; CLI flags override them.
- Dangerous operations run only below a per-run `os.MkdirTemp` root and always have timeout/deferred cleanup.
- No fixed performance percentage, API key, password, or private network data is written to source or evidence.
- AVP, Gateway, CVM, IPC, cgroup, OverlayFS, and eBPF descriptions follow the capability boundaries in the design spec.
- Existing CLI commands, final evidence files, and historical results remain compatible and are never overwritten by review outputs.

### Task 1: P1 Audit and specifications

**Files:**
- Create: `docs/review_remediation/AUDIT.md`
- Create: `docs/review_remediation/PLAN.md`
- Create: `docs/review_remediation/REVIEW_TO_TASK_MATRIX.md`
- Create: `docs/superpowers/specs/2026-07-13-review-remediation-design.md`
- Create: `docs/superpowers/plans/2026-07-13-review-remediation-plan.md`

- [ ] Record current branch, dirty state, language tree, command surface, capability classification, evidence status, overclaims, and baseline `go test ./...` output.
- [ ] Map each of the five review comments to code paths, experiments, evidence, acceptance criteria, and a planned task.
- [ ] Commit with `docs: audit review feedback and remediation plan`.

### Task 2: Metrics model and resource-isolation scenario

**Files:**
- Create: `internal/review/metrics.go`
- Create: `internal/review/metrics_test.go`
- Create: `internal/review/resource_isolation.go`
- Create: `internal/review/resource_isolation_test.go`
- Create: `internal/experiment/review_scenarios.go`
- Modify: `cmd/aortctl/main.go`
- Modify: `cmd/aortctl/main_test.go`

**Interfaces:**
- `review.Aggregate([]float64, []bool) review.Stats`
- `review.RunResourceIsolation(context.Context, review.ResourceIsolationConfig) (review.ScenarioResult, error)`
- `experiment.RunResourceIsolation(review.ResourceIsolationConfig) (review.ScenarioResult, error)`

- [ ] Add failing tests for percentile/mean/stddev, failed samples, mode validation, temp-root safety, and required output files.
- [ ] Implement aggregation and safe six-agent execution with baseline, isolation-only, and aort-r feature switches.
- [ ] Measure fault types, completion/failure, hashes, timings, and resource counters; write `raw/`, `summary.json`, `comparison.csv`, and `report.md`.
- [ ] Wire `aortctl scenario resource-isolation --mode --runs --warmup --seed --timeout --out` with default `mode=all`.
- [ ] Run `gofmt` and focused Go tests, then commit `feat: add resource isolation scenario benchmark`.

### Task 3: Context-sharing scenario

**Files:**
- Create: `internal/review/context_sharing.go`
- Create: `internal/review/context_sharing_test.go`
- Modify: `internal/experiment/review_scenarios.go`
- Modify: `cmd/aortctl/main.go`
- Modify: `cmd/aortctl/main_test.go`

**Interfaces:**
- `review.RunContextSharing(context.Context, review.ContextSharingConfig) (review.ScenarioResult, error)`
- `experiment.RunContextSharing(review.ContextSharingConfig) (review.ScenarioResult, error)`

- [ ] Add failing tests for 0/25/50/75 ratios, full-copy/shared-ipc/aort-r metric semantics, unsupported labels, and no hard-coded percentage claims.
- [ ] Implement seeded public/private payloads, real CVM dedup/page references, existing memfd/mmap transport, and scheduler affinity accounting.
- [ ] Record logical/physical/transferred/materialized/saved bytes, page counts, IPC latency quantiles, RSS/throughput/wait/fairness and raw per-run observations.
- [ ] Wire `aortctl scenario context-sharing --mode --runs --warmup --seed --timeout --context-size --shared-ratio --out`.
- [ ] Run focused tests and commit `feat: add context sharing benchmark`.

### Task 4: Unified review evidence and real-agent demo

**Files:**
- Create: `internal/review/agent_demo.go`
- Create: `internal/review/agent_demo_test.go`
- Modify: `internal/experiment/review_scenarios.go`
- Modify: `internal/experiment/all_final.go`
- Modify: `cmd/aortctl/main.go`
- Modify: `cmd/aortctl/main_test.go`

- [ ] Add failing tests for six roles, at least one LLM call, three tool calls, fault continuation, API-key redaction, and mock determinism.
- [ ] Implement mock/deepseek provider selection through existing interfaces and write timeline/final result/summary/report artifacts.
- [ ] Add `aortctl scenario real-agent-demo --provider mock|deepseek --out`.
- [ ] Add `aortctl evidence review-final --out` that indexes scenario artifacts without mutating old final evidence.
- [ ] Run focused tests and commit `feat: add repeatable metrics and review evidence reports` and `feat: add real agent availability demo`.

### Task 5: Design, threat model, and defense materials

**Files:**
- Create: `docs/design/01_problem_definition.md` through `docs/design/10_limitations_and_future_work.md`
- Create: `docs/design/README.md`
- Create: `docs/design/THREAT_MODEL.md`
- Create: `docs/defense/PPT_OUTLINE.md`, `SPEECH_8MIN.md`, `DEMO_SCRIPT.md`, `REVIEW_RESPONSE.md`, `Q_AND_A.md`, `DATA_FOR_SLIDES.md`, `CLAIMS_BOUNDARY.md`
- Create: `docs/experiments/RESOURCE_ISOLATION_BENCHMARK.md`, `docs/experiments/CONTEXT_SHARING_BENCHMARK.md`

- [ ] Write every chapter with problem, goal, constraints, choices, flow/state, interfaces, failure handling, experiment, conclusion, and boundary sections.
- [ ] Cite concrete code paths, commands, evidence fields, sample counts, and current degraded states; remove claims unsupported by the repository.
- [ ] Keep the PPT and speech centered on resource isolation/fault control and context sharing/communication optimization.
- [ ] Check all relative links and commit `docs: add threat model and defense materials`.

### Task 6: Final verification and delivery

**Files:**
- Create: `docs/review_remediation/FINAL_CHECKLIST.md`
- Create: `docs/review_remediation/CHANGELOG.md`
- Create: `docs/review_remediation/FINAL_REPORT.md`
- Create: `experiments/results/review_final/REVIEW_EVIDENCE_INDEX.json`
- Create: `experiments/results/review_final/REVIEW_SUMMARY.md`
- Modify: `README.md`, only where command/claim corrections are required

- [ ] Run `gofmt -w` on changed Go files and `go test ./...`.
- [ ] Run old core smoke/real-only commands and the new resource, context, mock demo, final, and review-final commands in independent output directories.
- [ ] Validate every JSON/CSV/report link, absence of secrets, accurate degraded labels, and cleanup safety.
- [ ] Record exact commands, measured results, failures, limits, commit hash, and dirty state in the final report.
- [ ] Commit `test: complete final review-driven verification`, verify `git status`, and push `codex/aort-r-upgrade`.
