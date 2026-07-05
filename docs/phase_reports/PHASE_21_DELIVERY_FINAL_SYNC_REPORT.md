# PHASE 21 Delivery Final Sync Report

## Status

Current sync date: 2026-07-05

This report reconciles the older phase-boundary gaps with the current evidence
artifacts. It keeps evidence modes explicit instead of mixing real runtime,
real cgroup, real API, mock, degraded, and simulation results.

## Completed Since PHASE 17

- E1 real-runtime scheduler benchmark exists:
  `experiments/results/e1-real-scheduler.json` and
  `experiments/results/e1-real-scheduler.csv`.
- E2 real-runtime fault-isolation benchmark exists:
  `experiments/results/e2-real-fault.json` and
  `experiments/results/e2-real-fault.csv`.
- Software-real demo exists:
  `experiments/results/software_real_demo/result.json` and
  `docs/phase_reports/PHASE_20_SOFTWARE_REAL_DEMO.md`.
- DeepSeek provider exists with environment-only API key loading, fake-server
  tests, explicit mock fallback, and a real smoke summary:
  `experiments/results/deepseek_smoke/summary.json`.

## Current Evidence Modes

- `real-cgroup-v2`: openEuler 24.03 LTS cgroup v2 evidence under
  `experiments/results/openeuler_smoke/`,
  `experiments/results/openeuler_cgroupv2_multi/`, and
  `experiments/results/openeuler_cgroupv2_limits/`.
- `real-runtime`: E1/E2 benchmarks and software-real API flow, including real
  scheduler decisions, CVM syscalls, IPC, tool execution, checkpoints, and
  fault recovery.
- `real-api`: DeepSeek smoke summary with
  `provider=deepseek`, `model=deepseek-v4-flash`, and `fallback=false`.
- `mock`: DeepSeek fallback and default local LLM behavior when no API key is
  present or the API call fails.
- `degraded`: non-Linux or non-root cgroup fallback paths and historical
  environment comparisons only.
- `simulation`: legacy E1/E2 baseline outputs retained for comparison only.

## E1 Scheduler Sync

E1 no longer reports equal wall time across all policies. The workload now
uses heavier context materialization, repeated shared system/project/task
pages, and prefix-affinity shared-page hits. It emits:

- `duplicate_tokens`
- `materialize_ms`
- `saved_ms`

Current ordering:

- `fifo`: `wall_time_ms=17600`, `p95_latency_ms=388`,
  `materialize_ms=17350`
- `token-cfs`: `wall_time_ms=14603`, `p95_latency_ms=328`,
  `materialize_ms=14453`
- `token-cfs-prefix-affinity`: `wall_time_ms=10572`,
  `p95_latency_ms=236`, `materialize_ms=10472`

Prefix-affinity materialize cost is about 27.5% lower than token-CFS.

## Software-Real Sync

The software-real API now enriches the six Agents from the worker registry and
capsule manager, so result rows can include real worker `pid`,
`capsule_mode`, `capsule_evidence_mode`, and `cgroup_path`. On openEuler with
`cgroup_root=/sys/fs/cgroup/aort.slice`, these paths resolve under
`/sys/fs/cgroup/aort.slice/...` and provide OS-level isolation evidence in the
same demo that exercises Planner -> Coder -> Tester -> Reviewer -> Fixer ->
Reporter.

The checked-in local result proves the worker/capsule-backed result shape and
runtime path with `capsule_evidence_mode=test-cgroup-v2`. A live openEuler
rerun should be used when the final artifact must show
`capsule_evidence_mode=real-cgroup-v2` and `/sys/fs/cgroup/aort.slice/...`
paths rather than a local test cgroup root.

## DeepSeek Sync

DeepSeek provider behavior is now split clearly:

- Real API success: `provider=deepseek`, `model=deepseek-v4-flash`,
  `fallback=false`, `evidence_mode=real-api`.
- Fallback: `provider=mock`, `requested_provider=deepseek`, `fallback=true`,
  `fallback_reason=no_api_key` or `api_error`, `evidence_mode=mock`.

The smoke script skips without failure when no key is present and writes a real
summary when `DEEPSEEK_API_KEY` is present. The API key is not stored in code,
README, reports, or experiment outputs.

## Remaining Work

- Refresh `experiments/results/software_real_demo/result.json` from a live
  openEuler worker/cgroup run if the submitted artifact must itself show
  `/sys/fs/cgroup/aort.slice/...` paths.
- Keep the legacy simulation/degraded artifacts labeled as historical
  comparison data only.
- Future OS depth work remains outside this sync: full eBPF attachment and
  stronger overlayfs-backed workspace isolation.
