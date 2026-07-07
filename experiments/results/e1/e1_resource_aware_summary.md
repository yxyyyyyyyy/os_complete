# E1 Resource-Aware Scheduler Summary

| policy | evidence_mode | wall_time_ms | p95_task_latency_ms | decisions | memory_peak_bytes | pids_peak |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| fifo | real-runtime | 978 | 42 | 24 | 149946368 | 27 |
| token-cfs | real-runtime | 762 | 33 | 24 | 188743680 | 30 |
| token-cfs-prefix-affinity | real-runtime | 546 | 24 | 24 | 227540992 | 33 |
| token-cfs-prefix-affinity-resource-aware | degraded | 697 | 31 | 24 | 227540992 | 33 |

Resource-aware decisions use cgroup/PSI data when available and degraded fallback metadata when local files cannot be read.
