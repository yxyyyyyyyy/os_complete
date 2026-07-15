# Context Sharing Benchmark

## Command

```bash
go run ./cmd/aortctl scenario context-sharing \
  --mode all --warmup 3 --runs 20 --seed 20260713 \
  --agents 6 --context-size 4096 --timeout 5s \
  --out experiments/results/review_remediation/context_sharing
```

## Workload and modes

每个 Agent 逻辑上下文为 4096 bytes，公共比例 0/25/50/75%。比较 full-copy、shared-ipc、aort-r。公共和私有 payload 由 seed 生成，禁止用常量结果填充统计。

## Counters

`logical_context_bytes` 是每个 Agent 可见量；`physical_bytes_written` 是唯一写入量；`bytes_transferred` 是实际 payload/page-reference 计数；`materialized_bytes` 是实际组装量；`saved_bytes` 由 logical-transferred 推导。RSS 不可用时标 unsupported。

## Current result

240 measured observations。full-copy transferred 始终 24576 bytes；aort-r 在 0/25/50/75% 为 24576/18496/12352/6208 bytes，saved 为 0/6080/12224/18368。50% 时 Prefix Affinity=5 hits/run。该 benchmark 不测模型内部 KV Cache。

本机 shared-ipc 因平台条件 degraded；Linux/openEuler 可用性应重新运行 memfd/mmap 并检查 `evidence_mode=real-shm-ipc`。

## Artifacts

- `raw/*.json`
- `summary.json`
- `comparison.csv`
- `report.md`
