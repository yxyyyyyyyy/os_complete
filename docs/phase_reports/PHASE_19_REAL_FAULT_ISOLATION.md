# PHASE 19 Real Fault Isolation

## Status

Experiment: `E2_real_fault_isolation_benchmark`

Evidence mode: `real-runtime`

Artifacts:

- `experiments/results/e2-real-fault.json`
- `experiments/results/e2-real-fault.csv`

## Injected Faults

- `tool_timeout`
- `agent_crash`
- `kill_capsule`
- `memory_limit_exceeded`
- `pids_limit_exceeded`

Each row records `failed_agent`, `affected_agents`, `unaffected_agents`,
`cascade_failure=false`, `recovery_action`, `recovery_time_ms`,
`checkpoint_used`, and `final_status`.

For `memory_limit_exceeded` and `pids_limit_exceeded`, the E2 row keeps
`evidence_mode=real-runtime` for the runtime fault/recovery path and embeds
`fault_evidence.limit_evidence_mode=real-cgroup-v2` pointing to the openEuler
cgroup v2 enforcement artifacts:

- `experiments/results/openeuler_cgroupv2_limits/memory_limit_enforced.json`
- `experiments/results/openeuler_cgroupv2_limits/pids_limit_enforced.json`

## Result Summary

All five injected fault rows are labeled `real-runtime`, isolate the failure to
one Agent, keep five Agents unaffected, use checkpoint recovery, and finish with
`final_status=recovered`.

The memory and pids limit rows additionally carry `resource_limit_enforced=true`
from the real openEuler cgroup v2 limit artifacts, keeping `real-runtime` and
`real-cgroup-v2` evidence modes explicit.

The legacy `e2-fault.*` synthetic outputs remain only as historical comparison
data.
