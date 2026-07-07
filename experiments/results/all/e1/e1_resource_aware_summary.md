# E1 Resource-Aware Scheduler Summary

| policy | evidence_mode | wall_time_ms | p95_task_latency_ms | decisions | memory_peak_bytes | pids_peak |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| fifo | real-runtime | 325 | 42 | 8 | 133169152 | 11 |
| token-cfs | real-runtime | 253 | 33 | 8 | 171966464 | 14 |
| token-cfs-prefix-affinity | real-runtime | 181 | 24 | 8 | 210763776 | 17 |
| token-cfs-prefix-affinity-resource-aware | degraded | 218 | 29 | 8 | 210763776 | 17 |

Resource-aware decisions use cgroup/PSI data when available and degraded fallback metadata when local files cannot be read.
