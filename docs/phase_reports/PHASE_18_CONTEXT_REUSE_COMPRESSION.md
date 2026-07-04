# PHASE_18_CONTEXT_REUSE_COMPRESSION

mode=real-runtime

本阶段补齐 E3 real context reuse benchmark。实验通过 CVM page store、page table、
shared pages、delta pages 和 summary pages 计算 baseline / CVM / CVM+summary
三组指标。这里的 summary page 是 Runtime 层上下文压缩页，不是模型内部真实 KV Cache。

## 运行命令

```bash
mkdir -p .cache/go-build
GOCACHE="$PWD/.cache/go-build" go run ./cmd/aort-experiment --name e3-real-context --runs 5 --out experiments/results
```

总实验命令：

```bash
GOCACHE="$PWD/.cache/go-build" go run ./cmd/aort-experiment --name all --runs 5 --out experiments/results
```

## 证据文件

```text
experiments/results/e3-real-context.json
experiments/results/e3-real-context.csv
```

## 实验结果

| mode | baseline_tokens | actual_materialized_tokens | saved_tokens | shared_pages | summary_pages | reuse_rate |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| baseline | 36000 | 36000 | 0 | 0 | 0 | 0.0000 |
| cvm | 36000 | 9608 | 6475 | 43 | 0 | 0.1799 |
| cvm-summary | 36000 | 7760 | 10955 | 11 | 8 | 0.3043 |

## 结论

- CVM page sharing 将实际 materialized tokens 从 36000 降到 9608。
- CVM+summary 进一步降到 7760，并生成 8 个 summary pages。
- 当前 benchmark 使用 Runtime 层 summary page 建模压缩收益；通用
  `POST /api/context/compress` API 和 cold delta audit log 仍属于后续 P1 工作。

## 验证

```bash
GOCACHE="$PWD/.cache/go-build" go test ./internal/cvm ./internal/experiment
```
