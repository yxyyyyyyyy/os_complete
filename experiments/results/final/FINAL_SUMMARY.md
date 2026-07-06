# AORT-R Final Evidence Summary

- timestamp: 2026-07-06T03:47:21.751413+00:00
- evidence_mode: real-runtime
- go_test: passed
- smoke: passed
- e1_scheduler: passed
- e2_fault_isolation: passed
- software_real_demo: passed
- workspace_isolation: passed

## Generated Files
- `experiments/results/final/env_check.json`
- `experiments/results/e1/e1_resource_aware.json`
- `experiments/results/e1/e1_resource_aware.csv`
- `experiments/results/e1/e1_resource_aware_decisions.json`
- `experiments/results/e1/e1_resource_aware_summary.md`
- `experiments/results/e2-real-fault.json`
- `experiments/results/e2-real-fault.csv`
- `experiments/results/software_real_demo/result.json`
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
- overlayfs: degraded-copy

## Known Limits
- overlayfs mount was unavailable or not attempted; workspace isolation used degraded-copy fallback
