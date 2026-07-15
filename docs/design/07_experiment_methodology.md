# 07 Experiment Methodology

## 设计

每个模式默认 warmup=3、measured runs=20、固定 seed=20260713、单次 timeout。CLI 可覆盖这些值。warmup 不进入汇总；失败 measured run 保留 raw JSON 并计入 success rate。

## 统计

`internal/review/metrics.go` 对有效数值计算 population mean/stddev、min/max、线性插值 P50/P95。success rate 的分母是 measured observations；缺失 success 标记按失败处理。除零或非有限 baseline 不生成 improvement。

每项指标带：

- `measured`: 来自本轮执行计时或字节/事件计数。
- `derived`: 由 measured 字段计算，例如 saved bytes、吞吐和 Jain fairness。
- `unsupported`: 当前平台无法直接获取，例如无 `/proc` 时的 RSS。

## 可重复性

```bash
go run ./cmd/aortctl scenario resource-isolation --mode all --warmup 3 --runs 20 --seed 20260713 --out experiments/results/review_remediation/resource_isolation
go run ./cmd/aortctl scenario context-sharing --mode all --warmup 3 --runs 20 --seed 20260713 --agents 6 --context-size 4096 --out experiments/results/review_remediation/context_sharing
go run ./cmd/aortctl scenario real-agent-demo --provider mock --seed 20260713 --out experiments/results/review_remediation/real_agent_demo
go run ./cmd/aortctl evidence review-final --out experiments/results/review_final
```

## 环境分层

本地 portable run 用于测试场景逻辑、统计和安全清理，允许 degraded。real-only openEuler 证据使用 openEuler 24.03 LTS、cgroup2fs、root 条件，来源为 `experiments/results/final/FINAL_EVIDENCE_INDEX.json` 及其索引文件。两者不得混写为同一运行。

## 数据完整性

报告生成器只读 raw observations，不覆盖历史原始结果。`review-final` 引用旧 final，缺失或失败时仍写失败索引。最终检查解析所有 JSON、校验 CSV 列、扫描密钥模式，并核对 passed 与失败数组。
