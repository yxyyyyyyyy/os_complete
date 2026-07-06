# AORT-R Final Evidence Summary

- timestamp: 2026-07-06T07:26:14.389374+00:00
- evidence_mode: real-runtime
- go_test: passed
- smoke: passed
- e1_scheduler: passed
- e1_pressure: passed
- e2_fault_isolation: passed
- e2_pressure_fault: passed
- software_real_demo: passed
- workspace_probe: passed
- workspace_isolation: passed

## Generated Files
- `experiments/results/final/step_status.tsv`
- `experiments/results/final/env_check.json`
- `experiments/results/e1/e1_resource_aware.json`
- `experiments/results/e1/e1_resource_aware.csv`
- `experiments/results/e1/e1_resource_aware_decisions.json`
- `experiments/results/e1/e1_resource_aware_summary.md`
- `experiments/results/e1_pressure/e1_pressure.json`
- `experiments/results/e2-real-fault.json`
- `experiments/results/e2-real-fault.csv`
- `experiments/results/e2_pressure_fault/e2_pressure_fault.json`
- `experiments/results/software_real_demo/result.json`
- `experiments/results/workspace_probe.json`
- `experiments/results/workspace_isolation_evidence.json`

## Missing Files
- none

## Evidence Mode Summary
- cgroup_capsule: real-cgroup-v2
- worker_process: real-runtime
- cvm: real-partial
- ipc: real-partial
- llm: mock
- ebpf: planned
- overlayfs: real-overlayfs

## Known Limits
- resource-aware pressure sampler degraded: resource pressure sampler not configured or local cgroup pressure files unavailable
