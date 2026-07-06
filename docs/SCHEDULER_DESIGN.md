# Scheduler Design

AORT-R supports four scheduler policies:

| policy | behavior |
| --- | --- |
| `fifo` | Selects the earliest ready Agent. |
| `token-cfs` | Selects the ready Agent with the lowest token vruntime. |
| `token-cfs-prefix-affinity` | Starts from token-CFS, then prefers a candidate with higher shared context pages when it is within the vruntime threshold. |
| `token-cfs-prefix-affinity-resource-aware` | Adds resource pressure penalties to vruntime and context-affinity scoring. |

The resource-aware policy considers:

- token vruntime;
- context affinity score from shared CVM pages;
- memory pressure from cgroup metrics or candidate runtime stats;
- pids pressure;
- CPU throttle pressure from `cpu.stat`;
- PSI pressure from `/proc/pressure/{cpu,memory,io}` when available;
- dependency readiness;
- dynamic spawn priority through the Agent priority field.

Every scheduler decision records:

- `decision_id` and `timestamp`;
- `policy`;
- `candidates` and per-candidate score details;
- `selected_agent` and `selected_task`;
- `vruntime_score`, `context_affinity_score`, memory/pids/CPU/PSI pressure, dependency readiness, final score;
- `evidence_mode`;
- `fallback_reason`.

API:

- `GET /api/scheduler/decisions`
- `GET /api/scheduler/policies`
- `GET /api/scheduler/resource-pressure`
- `POST /api/scheduler/policy`

CLI:

```bash
go run ./cmd/aortctl experiment e1 --policy resource-aware --runs 5 --out experiments/results/e1
go run ./cmd/aortctl experiment e1 --policy all --runs 5 --out experiments/results/e1
```

In the current portable benchmark, `token-cfs-prefix-affinity` has the lowest
wall time. `token-cfs-prefix-affinity-resource-aware` improves over FIFO and
token-CFS while adding memory/pids/cpu/PSI pressure-aware safety decisions. Do
not present the resource-aware policy as the fastest policy unless a future
fresh benchmark actually shows that ordering.

Outputs:

- `experiments/results/e1/e1_resource_aware.json`
- `experiments/results/e1/e1_resource_aware.csv`
- `experiments/results/e1/e1_resource_aware_decisions.json`
- `experiments/results/e1/e1_resource_aware_summary.md`

If cgroup or PSI files cannot be read, the scheduler keeps running and marks
the decision evidence as `degraded` with a `fallback_reason`.
