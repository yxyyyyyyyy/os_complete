# Review Response

## 1. 设计思想没有讲透

老师意见：需要说明 Agent 间资源干扰、隔离、通信、安全、环境和测试方法。

整改：对外收敛为资源隔离/故障控制和上下文复用/通信优化两条主线；新增 `docs/design/01..10`、`THREAT_MODEL.md` 和三个 `scenario` 命令。资源场景记录 lowerdir hash、污染、资源和恢复；上下文场景区分 logical/physical/transferred/materialized。

证据：`internal/review/resource_isolation.go`, `context_sharing.go`, `experiments/results/review_remediation/`。边界：portable run 为 degraded，真实 OS 机制另引 openEuler final。

## 2. 技术路线讲得过快

老师意见：报告缺少关键问题、设计思想和概要设计结构。

整改：文档统一按问题、目标、约束、方案、数据流、接口、失败处理、实验、结论、边界组织；总体架构、AVP 生命周期、故障序列和上下文数据流均提供 Mermaid 图。

证据：`docs/design/README.md` 及 10 章索引。

## 3. 通信机制缺少效率量化

老师意见：需要可比较、可重复的数据。

整改：新增 full-copy/shared-ipc/aort-r 三模式和 0/25/50/75% 四档。默认 warmup=3、runs=20；输出 mean/std/min/max/P50/P95/success rate 和 measurement kind。

证据：240 个 measured observations；`context_sharing/summary.json` 与 `comparison.csv`。限制：数据是 Runtime 上下文计数，不是模型 KV Cache。

## 4. 真实服务可用性和 Demo 不足

老师意见：加强真实服务、Demo 和实际效果。

整改：新增 6-Agent `real-agent-demo`，复用 CVM、Router、Gateway 和工具执行，受控故障后继续；mock 离线重复，DeepSeek env-only 可选。

证据：本轮 mock passed；历史 final 记录 DeepSeek real-api。当前环境无 Key，所以没有把历史结果冒充本轮调用。

## 5. 应聚焦 1-2 个关键问题

老师意见：不要把所有功能平均包装成创新。

整改：AVP、Gateway、Timeline、DeepSeek、eBPF 统一降为两条闭环的支撑模块；PPT 固定 10 页围绕资源和上下文。CLAIMS_BOUNDARY 明确可说/谨慎说/不能说。

验收：`experiments/results/review_final/REVIEW_EVIDENCE_INDEX.json` 对三场景给出 status/evidence_mode/source，缺失时命令失败。
