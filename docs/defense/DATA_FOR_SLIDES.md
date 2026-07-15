# Data for Slides

所有数字来自当前 summary，不从文案估算。

## Resource slide

- 样本：3 modes x 20 measured runs = 60；warmup=3/mode。
- success_rate：baseline=1.0, isolation-only=1.0, aort-r=1.0。
- normal completion P50：2/2/2 ms。
- normal completion P95：3.8/3.15/3.62 ms。
- aort-r lowerdir unchanged mean=1；cross-agent contamination mean=0。
- 来源：`experiments/results/review_remediation/resource_isolation/summary.json`。
- 限制：本机 evidence_mode=degraded；不要把时延差解释为 cgroup 性能收益。

## Context slide

| shared ratio | full-copy transferred | shared-ipc transferred | aort-r transferred | aort-r saved | aort-r page hit |
|---:|---:|---:|---:|---:|---:|
| 0% | 24576 | 24576 | 24576 | 0 | 0 |
| 25% | 24576 | 19456 | 18496 | 6080 | 0.8333 |
| 50% | 24576 | 14336 | 12352 | 12224 | 0.8333 |
| 75% | 24576 | 9216 | 6208 | 18368 | 0.8333 |

来源：`experiments/results/review_remediation/context_sharing/summary.json`。shared-ipc 的数值需以 summary 为准；本机模式 degraded。50% 时 aort-r Prefix Affinity=5 hits/run。

## Demo slide

- 6 roles，1 `llm.call`，5 `tool.exec`。
- fault type=`tool_process_failure`；contained=true；continued=true。
- provider requested/actual=`mock`；evidence_mode=`mock`。
- 当前 `DEEPSEEK_API_KEY` unset，真实 API 本轮未执行。
- 历史 final：openEuler=true、real_cgroup_v2=true、real_overlayfs=true、deepseek_real_api=true、eBPF=degraded。

## Visual guidance

资源页画三列模式对比和 fault boundary，不画“性能提升百分比”。上下文页画共享比例到 transferred bytes 的折线/柱状图，y 轴单位 bytes。Demo 页画 Timeline 事件序列并标注 mock/real-api 边界。
