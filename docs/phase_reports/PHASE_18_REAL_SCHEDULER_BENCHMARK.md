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
- `duplicate_tokens`
- `materialize_ms`
- `saved_ms`
- `syscall_count`

## Workload Update

The current E1 workload repeatedly materializes heavier shared system, project,
task, API, and test pages. FIFO is offered a cold-prefix candidate first,
token-CFS is offered the lowest-vruntime medium-sharing candidate, and
token-CFS-prefix-affinity can choose a hot-prefix candidate within the
affinity threshold when it shares more pages with the previous prefix group.

The current result ordering is:

- `fifo`: `materialize_ms=17350`, `p95_latency_ms=388`
- `token-cfs`: `materialize_ms=14453`, `p95_latency_ms=328`
- `token-cfs-prefix-affinity`: `materialize_ms=10472`, `p95_latency_ms=236`

Prefix affinity reduces materialize cost by about 27.5% versus token-CFS while
still recording real scheduler decisions and CVM syscalls.

The result files are labeled `real-runtime`, not `simulation` or
`degraded-simulation`. The legacy `e1-scheduler.*` simulation outputs remain
only as historical comparison data.
