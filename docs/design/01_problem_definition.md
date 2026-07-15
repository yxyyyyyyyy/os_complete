# 01 Problem Definition

## 问题

AORT-R 本轮只回答两个可测问题。

1. 多 Agent 并发执行时，单个 Agent 的 CPU、内存、进程树或工作区故障会不会扩散到其他 Agent 与公共输入？
2. 多 Agent 读取相同公共上下文时，能否避免重复写入、重复传输和重复 materialization，并给出真实字节计数？

赛题给出的多 Agent 执行问题、Linux cgroup v2、OverlayFS、memfd/mmap 和 CFS 思想都不是项目原创。AORT-R 的贡献范围是：将这些机制组织成 Agent Runtime 的两个闭环，定义可审计的场景接口，采集统一指标，并在不支持的环境中给出明确 degraded 证据。

## 目标

- 资源闭环：AVP 运行时对象 -> cgroup v2 边界 -> 资源采样 -> resource-aware 调度 -> 工作区隔离 -> kill/destroy/recovery -> Timeline/证据。
- 上下文闭环：CVM 页面 -> 内容寻址与引用 -> memfd/mmap 或 page reference -> Prefix Affinity -> materialization -> 字节/时延/内存证据。
- 所有结论可追溯到 `internal/review`、`cmd/aortctl` 和 `experiments/results/review_remediation/`。

## 约束

- AVP 是 AORT-R 用户态运行时对象，不是 Linux 新增进程类型。
- Syscall Gateway 是运行时受控调用入口，不是新增内核系统调用。
- CVM 是上下文页复用原型，不等于模型内部 KV Cache。
- page-reference 和 memfd/mmap 不宣称 kernel zero-copy。
- cgroup + OverlayFS 不构成完整容器、VM、namespace/seccomp/MAC 安全沙箱。
- eBPF 是可选观测增强；当前提交的 eBPF evidence 为 degraded。

## 关键实现

- CLI: `cmd/aortctl/main.go`
- 场景与统一统计: `internal/review/metrics.go`, `resource_isolation.go`, `context_sharing.go`
- 支撑机制: `internal/capsule`, `internal/workspace`, `internal/cvm`, `internal/ipc`, `internal/scheduler`
- 证据汇总: `internal/review/review_final.go`

## 失败处理与边界

场景运行失败仍写 raw observation 和 failure reason。平台不允许 cgroup、OverlayFS、memfd/mmap 或 eBPF 时，命令保留可重复用户态路径并标记 degraded/unsupported，不将其计为真实 OS 机制成功。
