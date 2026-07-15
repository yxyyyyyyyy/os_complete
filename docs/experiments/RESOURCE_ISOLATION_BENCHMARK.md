# Resource Isolation Benchmark

## Command

```bash
go run ./cmd/aortctl scenario resource-isolation \
  --mode all --warmup 3 --runs 20 --seed 20260713 --timeout 5s \
  --out experiments/results/review_remediation/resource_isolation
```

## Workload

Planner、Coder-A、Coder-B、Tester、Reviewer、Fault-Agent。Fault-Agent 轮换 2 MiB bounded memory touch、4 个 bounded child process、15 ms CPU loop 和 generated-root-only rm-rf。测试不会删除仓库根或系统路径。

## Modes and metrics

模式为 baseline、isolation-only、aort-r。raw 指标包括完成率、task success、containment scope、正常 Agent P50/P95、memory/pids peak、检测/清理/恢复时延、lowerdir hash 和污染标志。统计口径见 `internal/review/metrics.go`。

## Current result

60 measured observations 全部 success。aort-r lowerdir unchanged=1、cross-agent contamination=0。本机 evidence_mode=degraded，因此这些数据证明 portable scenario、安全路径与清理闭环；real cgroup/OverlayFS 结论必须引用 `experiments/results/final/FINAL_EVIDENCE_INDEX.json`。

## Artifacts

- `raw/*.json`: 每个 measured run
- `summary.json`: 版本、参数、per-run、统计
- `comparison.csv`: 稳定列和 measurement kind
- `report.md`: 自动汇总
