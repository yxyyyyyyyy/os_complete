# PHASE_17_REAL_SCHEDULER_BENCHMARK

mode=real-runtime

本阶段将旧的 E1 调度实验从 `degraded-simulation` 扩展为真实 Runtime 路径：
实验直接调用 `internal/scheduler`、`internal/cvm` 的 page table/materialize
机制，记录真实 scheduler decision count、page overlap context reuse、latency 与
throughput 指标。该实验不声明真实 OS cgroup 证据；cgroup v2 real 仍由
PHASE_16 单独证明。

## 运行命令

```bash
mkdir -p .cache/go-build
GOCACHE="$PWD/.cache/go-build" go run ./cmd/aort-experiment --name e1-real-scheduler --runs 5 --out experiments/results
```

也可通过总实验命令生成：

```bash
GOCACHE="$PWD/.cache/go-build" go run ./cmd/aort-experiment --name all --runs 5 --out experiments/results
```

## 证据文件

```text
experiments/results/e1-real-scheduler.json
experiments/results/e1-real-scheduler.csv
```

## 实验结果

| policy | wall_time_ms | p95_latency_ms | throughput | context_reuse_rate |
| --- | ---: | ---: | ---: | ---: |
| fifo | 50 | 5 | 1000.00 | 0.9800 |
| token-cfs | 50 | 5 | 1000.00 | 0.8600 |
| token-cfs-prefix-affinity | 50 | 6 | 1000.00 | 0.9800 |

## 结论

- 三种策略均真实经过 scheduler selection 和 CVM page table/materialize。
- `token-cfs-prefix-affinity` 在本轮 workload 中保持与 FIFO 相同的上下文页重叠率，
  但未在 wall-clock 上显著优于 FIFO；这是当前 workload 粒度较小、Go 本地执行过快
  导致的限制，报告不把它伪造成系统级性能胜出。
- 后续应在 openEuler real worker/cgroup 环境下扩大 task count、tool exec cost 和
  shared prefix 差异，再补更强的 wall-clock 证据。

## 验证

```bash
GOCACHE="$PWD/.cache/go-build" go test ./internal/experiment ./cmd/aort-experiment
```
