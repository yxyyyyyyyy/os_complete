# AORT-R Review-Driven Remediation Design

- **Date:** 2026-07-13
- **Scope:** P1-P9 from `AORT-R_评审整改AI执行提示词_独立版.docx`
- **Target branch:** `codex/aort-r-upgrade`

## Goal

把仓库从“已有运行时模块和分散实验”整理为两条可复现、可量化、可审计的闭环：

1. 资源隔离与故障控制：六个软件工程 Agent 在 baseline、isolation-only、aort-r 三种模式下执行，压力和工作区故障只影响受控临时范围。
2. 上下文复用与通信优化：在 0/25/50/75% 公共上下文比例下比较 full-copy、shared-ipc、aort-r，并由实测计数生成统计结论。

已有 AVP、cgroup、CVM、IPC、OverlayFS、checkpoint/replay、Timeline、DeepSeek 和 eBPF 代码继续作为支撑能力；不删除旧命令和历史证据。

## Architecture

新增 `internal/review` 包，分成四个职责清晰的部分：

- `metrics.go`: 可复用的运行记录、统计聚合、百分位数、稳定 CSV 和 Markdown 报告。
- `resource_isolation.go`: 临时目录、六 Agent 工作流、压力注入、工作区哈希、清理和三模式对照。
- `context_sharing.go`: 真实字节构造、CVM page dedup、memfd/mmap 传输、Prefix Affinity 选择和四档共享比例。
- `agent_demo.go`: mock/DeepSeek provider 适配、六角色事件流、工具调用、可控故障和脱敏输出。

`internal/experiment/review_scenarios.go` 只负责把场景结果写入既有证据约定，并保留旧实验实现。`cmd/aortctl` 增加 `scenario` 和 `evidence review-final` 子命令，不改变现有 `experiment`、`demo`、`workspace`、`observer`、`ipc`、`cvm`、`replay` 命令。

## Evidence contract

每个 scenario 运行目录都包含：

```text
<out>/<scenario>/<timestamp>/
  raw/run-*.json
  summary.json
  comparison.csv
  report.md
```

`summary.json` 使用版本化字段：`schema_version`、`scenario_id`、`run_id`、`timestamp`、`git_commit`、`git_dirty`、环境信息、`mode`、`parameters`、`seed`、`warmup`、`measured_runs`、`per_run`、`mean/stddev/min/max/p50/p95/success_rate`、失败原因、产物路径和限制。每个指标同时带 `measurement_kind`，值只能是 `measured`、`derived` 或 `unsupported`。

所有提升百分比在报告生成阶段由 baseline 与目标模式的实际数值计算；代码和静态文档不写固定百分比。API key 只从环境变量读取，任何 JSON、日志和 Markdown 都经过脱敏。

## Resource-isolation flow

每次运行在 `os.MkdirTemp` 创建的目录中建立 lowerdir 和六个 Agent 工作区。Planner、Coder-A、Coder-B、Tester、Reviewer 和 Fault-Agent 产生真实文件、哈希和耗时；Fault-Agent 依次支持 memory hog、pids hog、CPU hog、受控 rm-rf，并可选执行 worker crash/replay。aort-r 模式调用现有 workspace/capsule 能力；权限不足时保留 `degraded` 和 `fallback_reason`，不伪称真实 cgroup/OverlayFS。

清理只接受由本次运行创建且位于临时根目录下的路径，执行前进行绝对路径和前缀校验。每个 goroutine 都有 context timeout，`defer` 负责进程、挂载、文件和 cgroup 清理。报告记录正常 Agent 完成率、任务成功率、故障影响范围、P50/P95、资源峰值、检测/kill/destroy/恢复时延、lowerdir 前后哈希和跨 Agent 污染。

## Context-sharing flow

公共上下文和 Agent 私有上下文由固定 seed 生成，逻辑大小由参数决定。full-copy 使用独立字节副本；shared-ipc 使用现有 memfd/mmap smoke transport；aort-r 使用 CVM 内容寻址页、page reference 和 scheduler prefix affinity。`logical_context_bytes`、`physical_bytes_written`、`bytes_transferred`、`materialized_bytes`、`saved_bytes`、shared/private pages、命中率、IPC P50/P95、RSS、吞吐、等待时间和公平性均从运行过程计数或明确推导。

## Real-agent demo

mock 模式固定 seed 和响应，完全离线；deepseek 模式复用现有 provider/gateway，只在 `AORT_ENABLE_REAL_LLM=1` 且 `DEEPSEEK_API_KEY` 存在时发起请求。六个角色至少产生一次 `llm.call` 和三次 `tool.exec`，故障 Agent 失败后其余 Agent 继续，timeline/final_result/summary/report 均保留。网络耗时、模型耗时和 runtime overhead 分开记录。

## Boundaries

AVP 是 AORT-R 运行时执行对象，不是 Linux 新进程类型；Syscall Gateway 是运行时受控入口，不是新增内核系统调用；CVM 是上下文页复用原型，不等于真实推理引擎 KV Cache；memfd/mmap 结果不宣称 kernel zero-copy；cgroup + OverlayFS 不等于完整容器、VM、namespace/seccomp/MAC 安全沙箱；eBPF attach 失败必须是 degraded；主要真实环境范围是单机 openEuler。

## Acceptance

- P1: `AUDIT.md`、`PLAN.md`、`REVIEW_TO_TASK_MATRIX.md` 引用真实路径、命令和证据字段。
- P2/P3: 三模式/三模式和四比例命令输出 raw、summary、CSV、report，且可重复。
- P4: 统一聚合器覆盖奇偶样本、失败样本、除零和缺失值；`review-final` 输出总索引和摘要。
- P5: mock Demo 离线通过，真实 API 路径在无 key 时明确 skipped，在有 key 时只保留脱敏证据。
- P6-P8: 设计、威胁模型、能力边界、报告和答辩源稿只引用仓库实际实现。
- P9: gofmt、`go test ./...`、旧 smoke、real-only smoke、新场景和 evidence final/review-final 均有记录；历史 evidence 不被覆盖。
