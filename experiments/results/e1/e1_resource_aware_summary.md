# E1 Resource-Aware Scheduler Summary

| policy | evidence_mode | wall_time_ms | p95_task_latency_ms | decisions | memory_peak_bytes | pids_peak |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| fifo | real-runtime | 1635 | 42 | 40 | 166723584 | 43 |
| token-cfs | real-runtime | 1275 | 33 | 40 | 205520896 | 46 |
| token-cfs-prefix-affinity | real-runtime | 915 | 24 | 40 | 244318208 | 49 |
| token-cfs-prefix-affinity-resource-aware | degraded | 1228 | 33 | 40 | 244318208 | 49 |

Resource-aware decisions use cgroup/PSI data when available and degraded fallback metadata when local files cannot be read.
