# PHASE 17 Real Cgroup V2 Report

## Status

Current evidence mode: `real-cgroup-v2`.

The latest openEuler smoke evidence is from openEuler 24.03 LTS with unified
cgroup v2 mounted. The cgroup filesystem check is:

```text
stat -fc %T /sys/fs/cgroup = cgroup2fs
```

Runtime capsule status is `capsule_mode=real` and `real_cgroup_v2=true`.

## Evidence

- `experiments/results/openeuler_smoke/`
- `experiments/results/openeuler_cgroupv2_multi/`
- `experiments/results/openeuler_cgroupv2_limits/`
- `experiments/results/openeuler_smoke/capsule_real.json`
- `experiments/results/openeuler_smoke/agent_summary.json`
- `experiments/results/openeuler_cgroupv2_multi/multi_agent_summary.json`
- `experiments/results/openeuler_cgroupv2_limits/limit_summary.json`

## Verified Behavior

- Multiple Agent capsules ran with real cgroup v2 paths under `/sys/fs/cgroup`.
- `memory.current`, `pids.current`, and `cpu.stat` were collected from real
  cgroup files.
- Freeze, unfreeze, and kill operations completed on real cgroup v2 capsules.
- Memory, pids, and CPU limit enforcement evidence is present in
  `openeuler_cgroupv2_limits`.

Historical degraded records remain useful as old-environment comparisons only.
They do not represent the current openEuler state.

## Phase-Boundary Gaps Superseded Later

- At this phase boundary, E1/E2 still needed real-runtime benchmarks; current
  evidence is now in `docs/phase_reports/PHASE_18_REAL_SCHEDULER_BENCHMARK.md`
  and `docs/phase_reports/PHASE_19_REAL_FAULT_ISOLATION.md`.
- At this phase boundary, the software demo still needed a fuller
  end-to-end real-runtime path; current evidence is now in
  `docs/phase_reports/PHASE_20_SOFTWARE_REAL_DEMO.md`.
- At this phase boundary, DeepSeek still needed provider integration; current
  code now includes `internal/llm/deepseek_provider.go` with environment-only
  API key loading, mock fallback, and a real API smoke summary in
  `experiments/results/deepseek_smoke/summary.json`.

For the latest delivery-wide reconciliation, see
`docs/phase_reports/PHASE_21_DELIVERY_FINAL_SYNC_REPORT.md`.
