# 08 Current Results

## 运行范围

当前 review evidence 由本机 portable run 生成：resource 3 模式 x 20 measured runs = 60；context 3 模式 x 4 比例 x 20 = 240；mock Demo 1 次。resource/context evidence_mode 为 degraded，原因是本机不提供安全的 openEuler cgroup/OverlayFS/memfd 完整条件。

## 资源结果

| 模式 | success rate | normal P50 ms | normal P95 ms |
|---|---:|---:|---:|
| baseline | 1.0 | 2.0 | 3.8 |
| isolation-only | 1.0 | 2.0 | 3.15 |
| aort-r | 1.0 | 2.0 | 3.62 |

aort-r 的 lowerdir hash unchanged mean=1，cross-agent contamination mean=0。该场景证明 portable 工作区与清理逻辑，没有证明本机执行了真实 cgroup 限额。真实 openEuler final 显示 `real_cgroup_v2=true`, `real_overlayfs=true`, `real_resource_sampler=true`。

## 上下文结果

| 公共比例 | full-copy transferred bytes | aort-r transferred bytes | derived saved bytes | page hit ratio |
|---:|---:|---:|---:|---:|
| 0% | 24576 | 24576 | 0 | 0 |
| 25% | 24576 | 18496 | 6080 | 0.8333 |
| 50% | 24576 | 12352 | 12224 | 0.8333 |
| 75% | 24576 | 6208 | 18368 | 0.8333 |

50% 时每轮 Prefix Affinity 命中 5 次，对应首个 Agent 后的其余 5 个 Agent。数据只说明 AORT-R Runtime 的上下文页和传输计数，不说明模型内部缓存命中。

## Demo 与既有实证

mock Demo：6 Agent、1 次 `llm.call`、5 次 `tool.exec`，受控 `false` 工具失败被隔离，后续 Tester/Reviewer 继续，status=passed。当前环境没有 `DEEPSEEK_API_KEY`，因此本轮未发真实 API；历史 final evidence 的 `deepseek_real_api=true` 继续作为已有 openEuler 可用性证据，但不冒充本轮运行。

## 证据

- `experiments/results/review_remediation/resource_isolation/summary.json`
- `experiments/results/review_remediation/context_sharing/summary.json`
- `experiments/results/review_remediation/real_agent_demo/summary.json`
- `experiments/results/review_final/REVIEW_EVIDENCE_INDEX.json`
- `experiments/results/final/FINAL_EVIDENCE_INDEX.json`
