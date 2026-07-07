# COMPETITION_EVIDENCE_MAP

本文件把赛题四个核心问题映射到 AORT-R 的实现模块、API 和证据文件。

## 1. 多 Agent 执行缺乏统一调度

对应能力：

| 项目能力 | 证据 |
| --- | --- |
| AVP 抽象 | `internal/avp/`, `/api/agents` |
| Worker process | `/api/agents` 中的真实 `pid` |
| token-CFS-prefix-affinity scheduler | `internal/scheduler/`, `/api/scheduler/decisions` |
| resource-aware scheduler | `internal/scheduler/`, `/api/scheduler/resource-pressure`, `experiments/results/e1/e1_resource_aware.json` |
| pressure risk scheduler evidence | `go run ./cmd/aortctl experiment e1-pressure`, `experiments/results/e1_pressure/e1_pressure.json` |
| cgroup v2 capsule | `/api/capsules`, `capsule_real.json` |
| scheduler trace | `experiments/results/e1-real-scheduler.json` |

当前证据模式：worker process 为 `real-runtime`；openEuler cgroup v2 capsule 为
`real-cgroup-v2`；资源指标不可读时 scheduler decision 标记 `degraded` 并写
`fallback_reason`。

当前 portable E1 benchmark 中 `token-cfs-prefix-affinity` wall time 最低；
resource-aware policy 应表述为资源压力感知和安全调度机制，而不是最快策略。

## 2. Context / KV Cache 冗余拷贝

对应能力：

| 项目能力 | 证据 |
| --- | --- |
| CVM page store | `internal/cvm/`, `/api/context/pages` |
| Page table | `/api/context/agents/:id/pagetable` |
| Page-reference IPC | `internal/ipc/`, `/api/ipc/metrics` |
| memfd/mmap shared-memory IPC | `internal/ipc/shm/`, `experiments/results/ipc_shm/ipc_shm_smoke.json` |
| CVM memory management | `experiments/results/cvm_memory/cvm_memory_smoke.json` |
| saved bytes / tokens | `/api/context/stats`, `experiments/results/e3-real-context.json` |

当前证据模式：Runtime page store、page table、saved bytes/tokens 为
`real-partial`。CVM memory smoke 覆盖 hot page、LRU eviction、compression、
pin/refcount 和 materialize correctness。真实模型 KV cache 内存映射未接入，
答辩时应描述为上下文页级复用，不说真实 KV Cache 共享。

## 3. 模型调用、工具调用与系统资源缺乏统一抽象

对应能力：

| 统一 syscall | 证据 |
| --- | --- |
| `llm.call` | `/api/syscalls`，默认 provider 为 `mock` |
| `tool.exec` | `/api/syscalls`, `fault_tool_timeout.json` |
| `context.materialize` | `/api/syscalls`, `/api/context/stats` |
| `ipc.publish` / `ipc.poll` | `/api/syscalls`, `/api/ipc/metrics`; mode 可为 `page-reference` 或 `memfd-mmap` |
| `agent.spawn` | `/api/syscalls`, worker registry |
| cgroup resource control | `/api/capsules`, real cgroup v2 files |

当前证据模式：syscall gateway、tool、agent control 为 `real-runtime`；CVM 和
IPC 为 `real-partial`；LLM provider 默认 `mock`，不能宣称真实 DeepSeek/llama.cpp。

## 4. 复杂任务缺乏系统级可观测性与控制能力

对应能力：

| 项目能力 | 证据 |
| --- | --- |
| REST API | `/api/health`, `/api/evidence`, `/api/agents`, `/api/capsules` |
| SSE timeline | `/api/events` |
| Dashboard | `dashboard/src/` |
| Trace recorder | `.aort-dev/traces/` |
| Replay trace | `experiments/results/software_real_demo/trace.json`, `experiments/results/replay/replay_result.json` |
| eBPF smoke | `experiments/results/ebpf_smoke/ebpf_smoke.json` |
| Checkpoint | `/api/checkpoints`, `/api/recovery/status` |
| cgroup stats | `memory.current`, `pids.current`, `cpu.stat`, `cgroup.events` |
| Runtime control | `freeze`, `unfreeze`, `kill` API |
| Workspace isolation | `/api/workspaces`, `POST /api/demo/fault/workspace-rmrf`, `workspace_probe.json`, `workspace_isolation_evidence.json` |
| Final verification | `scripts/competition_verify.sh`, `experiments/results/final/FINAL_EVIDENCE_INDEX.json` |

当前证据模式：openEuler real cgroup v2 smoke 已验证 freeze/unfreeze/kill 为
`200`，且其他 Agent 未被级联影响的验证由
`scripts/smoke_cgroupv2_multi_agent.sh` 生成。

## Evidence Mode 边界

| 模块 | 当前状态 |
| --- | --- |
| Cgroup Capsule | `real-cgroup-v2` 或 `degraded` |
| Worker Process | `real-runtime` |
| Syscall Gateway | `real-runtime` |
| Scheduler | `real-runtime`；resource pressure 不可读时 decision 为 `degraded` |
| CVM | `real-partial` |
| Page-reference IPC | `real-partial` |
| memfd/mmap IPC | `real-shm-ipc` 或 `degraded` |
| Workspace Isolation | `real-overlayfs` 或 `degraded-copy` |
| Kernel Observer | `degraded` |
| PSI Monitor | `real-cgroup-v2` 相关 host 可读时参与调度；否则 `degraded` |
| eBPF Observer | `real-ebpf` 或 `degraded` |
| Replay | `real-runtime` |
| LLM Provider | `mock` |

eBPF observer experimental path implemented; current submitted evidence is
degraded unless openEuler/Linux smoke reports real-ebpf.

当前 workspace evidence 以 `workspace_probe.json` 和
`workspace_isolation_evidence.json` 为准；只有 Linux/root 主机成功 mount
overlayfs 且 probe 证明 merged 是 mountpoint 后才可保持 `real-overlayfs`，
其他环境复跑必须写 `degraded-copy`。

## P0/P1/P2 Evidence Files

| requirement | artifact |
| --- | --- |
| P0 one-command verification | `scripts/competition_verify.sh` |
| P0 final index | `experiments/results/final/FINAL_EVIDENCE_INDEX.json` |
| P0 final summary | `experiments/results/final/FINAL_SUMMARY.md` |
| P1 resource-aware E1 | `experiments/results/e1/e1_resource_aware.json` |
| P1 pressure-risk E1 | `experiments/results/e1_pressure/e1_pressure.json` |
| P2 pressure + workspace fault | `experiments/results/e2_pressure_fault/e2_pressure_fault.json` |
| P2 workspace overlay probe | `experiments/results/workspace_probe.json` |
| P2 workspace rmrf fault | `experiments/results/workspace_isolation_evidence.json` |
| eBPF observer smoke | `experiments/results/ebpf_smoke/ebpf_smoke.json` |
| IPC shm smoke | `experiments/results/ipc_shm/ipc_shm_smoke.json` |
| CVM memory smoke | `experiments/results/cvm_memory/cvm_memory_smoke.json` |
| Replay evidence | `experiments/results/replay/replay_result.json` |
