# PHASE 18 Real Scheduler Benchmark

## Status

Experiment: `E1_real_scheduler_benchmark`

Evidence mode: `real-runtime`

Artifacts:

- `experiments/results/e1-real-scheduler.json`
- `experiments/results/e1-real-scheduler.csv`

## Compared Policies

- `fifo`
- `token-cfs`
- `token-cfs-prefix-affinity`

## Metrics

The benchmark records live runtime work across multiple Agents. Each policy
executes repeated CVM `context.write_delta` and `context.materialize` syscalls,
then records scheduler decisions from the FIFO/token-CFS/token-CFS-prefix
affinity scheduler rather than using `policyAdjustedTime()` ratios.

It emits:

- `wall_time_ms`
- `p50_latency_ms`
- `p95_latency_ms`
- `throughput_tasks_per_sec`
- `scheduler_decision_count`
- `context_saved_tokens`
- `context_reuse_rate`
- `syscall_count`

The result files are labeled `real-runtime`, not `simulation` or
`degraded-simulation`. The legacy `e1-scheduler.*` simulation outputs remain
only as historical comparison data.
