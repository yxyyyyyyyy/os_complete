# AORT-R Review-Driven Design Index

本目录以两个问题组织 AORT-R 当前实现，不再把 AVP、CVM、Gateway、Timeline、DeepSeek 和 eBPF 并列包装成同等级创新。

1. [问题定义](01_problem_definition.md)
2. [场景与需求](02_scenarios_and_requirements.md)
3. [总体架构](03_architecture_overview.md)
4. [资源隔离与故障控制](04_resource_isolation_design.md)
5. [上下文复用与通信](05_context_sharing_design.md)
6. [安全与能力边界](06_security_and_boundaries.md)
7. [实验方法](07_experiment_methodology.md)
8. [当前结果](08_results.md)
9. [真实 Agent 可用性 Demo](09_real_agent_demo.md)
10. [限制与后续工作](10_limitations_and_future_work.md)

补充材料：[威胁模型](THREAT_MODEL.md)、[资源基准](../experiments/RESOURCE_ISOLATION_BENCHMARK.md)、[上下文基准](../experiments/CONTEXT_SHARING_BENCHMARK.md)。

事实优先级为：当前代码与新生成的 raw evidence > `experiments/results/final/FINAL_EVIDENCE_INDEX.json` 中的历史 openEuler 实证 > 设计文档。旧版 V1/V2/V3 和 `docs/superpowers/specs/2026-07-04-aort-r-final-design.md` 是历史方案，不作为当前能力证明。
