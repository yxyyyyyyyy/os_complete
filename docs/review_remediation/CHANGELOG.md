# 评审整改变更日志

## P1 仓库审计与计划

- 提交：`a40971d` `docs: audit review feedback and remediation plan`
- 新增仓库能力审计、分阶段计划和老师意见到任务/证据映射。
- 将 real、partial、mock、degraded 和历史证据边界落实到具体文件、命令和字段。

## P2 资源隔离与故障控制

- 提交：`0229a9c` `feat: add resource isolation scenario benchmark`
- 新增 `aortctl scenario resource-isolation`，串联 6 Agent、四类故障、三模式、临时工作区和统一 raw/summary/CSV/report 输出。
- 增加安全删除根校验、失败证据保留和场景测试。

## P3 上下文复用与通信量化

- 提交：`26ce32a` `feat: add context sharing benchmark`
- 新增 `aortctl scenario context-sharing`，覆盖三模式和 0/25/50/75% 共享比例。
- 分别采集 logical、physical、transferred、materialized、saved、page 和 Prefix Affinity 指标。

## P4/P5 Demo、统计与总证据

- 提交：`24de060` `feat: add real agent availability demo`
- 新增 mock/DeepSeek provider 分离的 6-Agent Demo，包含 Gateway、Timeline、LLM、工具故障和后续继续执行。
- 提交：`424e204` `feat: add repeatable review evidence reports`
- 新增通用统计、measurement kind、CSV/Markdown 报告和 `aortctl evidence review-final`；缺失/失败场景会生成失败索引并返回错误。

## P6/P7 概要设计与能力边界

- 提交：`405a064` `docs: rewrite report as problem-driven architecture design`
- 新增 `docs/design/01..10`、设计索引和威胁模型；技术叙事收敛为资源隔离/故障控制、上下文复用/通信优化两条主线。
- README、历史方案和技术报告增加版本/能力边界，避免把运行时原型写成内核原生机制、完整 KV Cache、零拷贝或完整沙箱。

## P8 答辩材料

- 提交：`4d43fc3` `docs: add defense materials and expert review response`
- 新增固定 10 页 PPT 源稿、8 分钟讲稿、4 分钟 Demo、逐条专家回复、Q&A、数据源和可说/谨慎说/不能说边界。

## P9 最终验证与证据收口

- 提交：包含本文件的 `test: complete final review-driven verification` 提交；最终哈希以 `git rev-parse HEAD` 为准。
- 新增 60 个资源 measured runs、240 个上下文 measured runs、1 次 mock Demo 的可追溯证据，以及整改总索引。
- 新增 `FINAL_CHECKLIST.md`、`FINAL_REPORT.md`，记录通过项、环境降级、未运行项、历史 real-only 证据和用户工作树排除项。
- 旧 `experiments/results/final/` 保持只读；兼容性重生成写入 `/private/tmp`，没有覆盖 2026-07-07 的 openEuler 历史索引。

## 兼容性

- 原有 CLI group 和 `scripts/competition_verify.sh`/`scripts/competition_verify_real.sh` 保留。
- 原有 `FINAL_EVIDENCE_INDEX.json`、`FINAL_SUMMARY.md` 和 real-only raw evidence 未删除、未改格式。
- 新输出限定在 `experiments/results/review_remediation/` 与 `experiments/results/review_final/`。
- Dashboard 的 4 个用户修改文件及其他无关未跟踪内容未纳入整改提交。
