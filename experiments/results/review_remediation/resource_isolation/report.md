# resource-isolation

- schema_version: `review/v1`
- run_id: `resource-isolation-1784128300414800000`
- evidence_mode: `degraded`
- seed: `20260713`
- warmup: 3
- measured_runs: 20

| mode | metric | mean | stddev | p50 | p95 | success_rate | kind |
|---|---|---:|---:|---:|---:|---:|---|
| aort-r | cleanup_ms | 1.000 | 0.000 | 1.000 | 1.000 | 1.000 | measured |
| aort-r | cross_agent_contamination | 0.000 | 0.000 | 0.000 | 0.000 | 1.000 | measured |
| aort-r | fault_containment_scope | 1.000 | 0.000 | 1.000 | 1.000 | 1.000 | measured |
| aort-r | fault_detection_ms | 5.900 | 5.744 | 3.500 | 15.000 | 1.000 | measured |
| aort-r | lowerdir_hash_unchanged | 1.000 | 0.000 | 1.000 | 1.000 | 1.000 | measured |
| aort-r | memory_peak_bytes | 1858380.000 | 1039280.146 | 2577424.000 | 2861278.800 | 1.000 | measured |
| aort-r | normal_agent_completion_rate | 1.000 | 0.000 | 1.000 | 1.000 | 1.000 | measured |
| aort-r | normal_completion_p50_ms | 2.350 | 0.654 | 2.000 | 3.050 | 1.000 | measured |
| aort-r | normal_completion_p95_ms | 2.790 | 0.531 | 3.000 | 3.620 | 1.000 | measured |
| aort-r | pids_peak | 6.000 | 0.000 | 6.000 | 6.000 | 1.000 | measured |
| aort-r | recovery_ms | 1.000 | 0.000 | 1.000 | 1.000 | 1.000 | measured |
| aort-r | resource_sampler_mode | 18.000 | 0.000 | 18.000 | 18.000 | 1.000 | derived |
| aort-r | task_success | 1.000 | 0.000 | 1.000 | 1.000 | 1.000 | measured |
| baseline | cleanup_ms | 1.000 | 0.000 | 1.000 | 1.000 | 1.000 | measured |
| baseline | cross_agent_contamination | 0.000 | 0.000 | 0.000 | 0.000 | 1.000 | measured |
| baseline | fault_containment_scope | 1.000 | 0.000 | 1.000 | 1.000 | 1.000 | measured |
| baseline | fault_detection_ms | 5.650 | 5.730 | 3.000 | 15.000 | 1.000 | measured |
| baseline | lowerdir_hash_unchanged | 1.000 | 0.000 | 1.000 | 1.000 | 1.000 | measured |
| baseline | memory_peak_bytes | 2656105.200 | 145193.967 | 2667192.000 | 2852160.800 | 1.000 | measured |
| baseline | normal_agent_completion_rate | 1.000 | 0.000 | 1.000 | 1.000 | 1.000 | measured |
| baseline | normal_completion_p50_ms | 2.100 | 0.436 | 2.000 | 3.000 | 1.000 | measured |
| baseline | normal_completion_p95_ms | 2.700 | 0.539 | 2.800 | 3.800 | 1.000 | measured |
| baseline | pids_peak | 6.000 | 0.000 | 6.000 | 6.000 | 1.000 | measured |
| baseline | recovery_ms | 1.000 | 0.000 | 1.000 | 1.000 | 1.000 | measured |
| baseline | resource_sampler_mode | 8.000 | 0.000 | 8.000 | 8.000 | 1.000 | derived |
| baseline | task_success | 1.000 | 0.000 | 1.000 | 1.000 | 1.000 | measured |
| isolation-only | cleanup_ms | 1.000 | 0.000 | 1.000 | 1.000 | 1.000 | measured |
| isolation-only | cross_agent_contamination | 0.000 | 0.000 | 0.000 | 0.000 | 1.000 | measured |
| isolation-only | fault_containment_scope | 1.000 | 0.000 | 1.000 | 1.000 | 1.000 | measured |
| isolation-only | fault_detection_ms | 5.500 | 5.723 | 3.000 | 15.000 | 1.000 | measured |
| isolation-only | lowerdir_hash_unchanged | 1.000 | 0.000 | 1.000 | 1.000 | 1.000 | measured |
| isolation-only | memory_peak_bytes | 2716179.600 | 137566.863 | 2736964.000 | 2891588.000 | 1.000 | measured |
| isolation-only | normal_agent_completion_rate | 1.000 | 0.000 | 1.000 | 1.000 | 1.000 | measured |
| isolation-only | normal_completion_p50_ms | 2.050 | 0.384 | 2.000 | 3.000 | 1.000 | measured |
| isolation-only | normal_completion_p95_ms | 2.430 | 0.911 | 2.000 | 3.150 | 1.000 | measured |
| isolation-only | pids_peak | 6.000 | 0.000 | 6.000 | 6.000 | 1.000 | measured |
| isolation-only | recovery_ms | 1.000 | 0.000 | 1.000 | 1.000 | 1.000 | measured |
| isolation-only | resource_sampler_mode | 8.000 | 0.000 | 8.000 | 8.000 | 1.000 | derived |
| isolation-only | task_success | 1.000 | 0.000 | 1.000 | 1.000 | 1.000 | measured |

## Limitations

- Portable runs use bounded process/memory counters and the existing workspace capability probe; cgroup evidence is degraded when the host cannot provide a safe nested cgroup.
