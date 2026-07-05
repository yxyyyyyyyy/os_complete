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

## Remaining Gaps

- E1/E2 still needed real-runtime benchmarks at this phase boundary.
- The software demo still needed a fuller end-to-end real-runtime path.
- DeepSeek provider still needed real API integration.
