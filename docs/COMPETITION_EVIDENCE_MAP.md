# COMPETITION_EVIDENCE_MAP

本文件把赛题四个核心问题映射到 AORT-R 的实现模块、API 和证据文件。

## 1. 多 Agent 执行缺乏统一调度

对应能力：

| 项目能力 | 证据 |
| --- | --- |
| AVP 抽象 | `internal/avp/`, `/api/agents` |
| Worker process | `/api/agents` 中的真实 `pid` |
| token-CFS-prefix-affinity scheduler | `internal/scheduler/`, `/api/scheduler/decisions` |
| cgroup v2 capsule | `/api/capsules`, `capsule_real.json` |
| scheduler trace | `experiments/results/e1-real-scheduler.json` |

当前证据模式：worker/capsule/scheduler 在 openEuler cgroup v2 smoke 中为 `real`。

## 2. Context / KV Cache 冗余拷贝

对应能力：

| 项目能力 | 证据 |
| --- | --- |
| CVM page store | `internal/cvm/`, `/api/context/pages` |
| Page table | `/api/context/agents/:id/pagetable` |
| Page-reference IPC | `internal/ipc/`, `/api/ipc/metrics` |
| saved bytes / tokens | `/api/context/stats`, `experiments/results/e3-real-context.json` |

当前证据模式：Runtime page store、page table、saved bytes/tokens 为 `real`；
真实模型 KV cache 内存映射未接入，答辩时应描述为 `real-partial`。

## 3. 模型调用、工具调用与系统资源缺乏统一抽象

对应能力：

| 统一 syscall | 证据 |
| --- | --- |
| `llm.call` | `/api/syscalls`，默认 provider 为 `mock` |
| `tool.exec` | `/api/syscalls`, `fault_tool_timeout.json` |
| `context.materialize` | `/api/syscalls`, `/api/context/stats` |
| `ipc.publish` / `ipc.poll` | `/api/syscalls`, `/api/ipc/metrics` |
| `agent.spawn` | `/api/syscalls`, worker registry |
| cgroup resource control | `/api/capsules`, real cgroup v2 files |

当前证据模式：syscall gateway、tool、context、IPC、agent control 为 `real`
Runtime output；LLM provider 默认 `mock`，不能宣称真实 DeepSeek/llama.cpp。

## 4. 复杂任务缺乏系统级可观测性与控制能力

对应能力：

| 项目能力 | 证据 |
| --- | --- |
| REST API | `/api/health`, `/api/evidence`, `/api/agents`, `/api/capsules` |
| SSE timeline | `/api/events` |
| Dashboard | `dashboard/src/` |
| Trace recorder | `.aort-dev/traces/` |
| Checkpoint | `/api/checkpoints`, `/api/recovery/status` |
| cgroup stats | `memory.current`, `pids.current`, `cpu.stat`, `cgroup.events` |
| Runtime control | `freeze`, `unfreeze`, `kill` API |

当前证据模式：openEuler real cgroup v2 smoke 已验证 freeze/unfreeze/kill 为
`200`，且其他 Agent 未被级联影响的验证由
`scripts/smoke_cgroupv2_multi_agent.sh` 生成。

## Evidence Mode 边界

| 模块 | 当前状态 |
| --- | --- |
| Cgroup Capsule | `real` |
| Worker Process | `real` |
| Syscall Gateway | `real` |
| Scheduler | `real` |
| CVM | `real-partial` |
| Page-reference IPC | `real-partial` |
| Workspace Isolation | `degraded-copy` |
| Kernel Observer | `degraded-proxy` |
| PSI Monitor | `unavailable/degraded` |
| eBPF Observer | `planned` |
| LLM Provider | `mock` |
