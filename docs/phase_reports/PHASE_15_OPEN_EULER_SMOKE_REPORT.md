# PHASE_15_OPEN_EULER_SMOKE_REPORT

mode=pending

本报告用于记录 AORT-R 在 openEuler 24.03 LTS / Linux root / cgroup v2
环境下的 smoke test 证据。

当前仓库尚未包含真实 openEuler smoke 输出，因此本文件是待运行模板。
未运行，不得作为 real 证据。

## 脚本与输出位置

| 项目 | 路径 |
| --- | --- |
| 环境检查脚本 | `scripts/check_openeuler_env.sh` |
| smoke 脚本 | `scripts/smoke_openeuler.sh` |
| smoke 输出目录 | `experiments/results/openeuler_smoke/` |

运行命令：

```bash
bash scripts/check_openeuler_env.sh
bash scripts/smoke_openeuler.sh
```

## 当前状态

| 项目 | 状态 |
| --- | --- |
| openEuler 实机 smoke | 未运行 |
| 当前报告模式 | `pending` |
| 当前是否有 `experiments/results/openeuler_smoke/` 输出 | 无 |
| 当前未运行原因 | 当前工作环境不是 openEuler 24.03 LTS / Linux root / cgroup v2 实机 |
| 是否声明 cgroup real | 否，等待 openEuler root 证据 |
| 是否声明 eBPF real | 否，本阶段不做 eBPF |
| 是否声明 overlayfs real | 否，本阶段不做 overlayfs |

## 需要提交的 JSON/文本证据清单

| 证据 | 文件 |
| --- | --- |
| `env_check.txt` | `experiments/results/openeuler_smoke/env_check.txt` |
| `agents.json` | `experiments/results/openeuler_smoke/agents.json` |
| `agent_summary.json` | `experiments/results/openeuler_smoke/agent_summary.json` |
| `syscalls.json` | `experiments/results/openeuler_smoke/syscalls.json` |
| `context_stats.json` | `experiments/results/openeuler_smoke/context_stats.json` |
| `scheduler_decisions.json` | `experiments/results/openeuler_smoke/scheduler_decisions.json` |
| `fault_tool_timeout.json` | `experiments/results/openeuler_smoke/fault_tool_timeout.json` |
| HTTP health | `experiments/results/openeuler_smoke/health.json` |
| demo run 结果 | `experiments/results/openeuler_smoke/demo_run.json` |
| smoke 汇总 | `experiments/results/openeuler_smoke/smoke_summary.json` |
| aortd 日志 | `experiments/results/openeuler_smoke/aortd.log` |
| Go 测试输出 | `experiments/results/openeuler_smoke/go_test.txt` |

## openEuler 实机结果摘要

| 项目 | 结果 | 证据来源 |
| --- | --- | --- |
| openEuler 版本 | 待运行后填写 | `env_check.txt` 中 `/etc/os-release` |
| kernel 版本 | 待运行后填写 | `env_check.txt` 中 `uname -a` |
| 是否 root | 待运行后填写 | `env_check.txt` 中 `id` |
| cgroup v2 是否 real | 待运行后填写 | `env_check.txt` 中 `stat -fc %T /sys/fs/cgroup`，期望 `cgroup2fs` |
| cgroup_path 示例 | 待运行后填写 | `agent_summary.json` 的 `cgroup_path` |
| memory.current 示例 | 待运行后填写 | `cgroup_memory_current.txt` 或 `agent_summary.json` 的 `memory_current` |
| pids.current 示例 | 待运行后填写 | `cgroup_pids_current.txt` 或 `agent_summary.json` 的 `pids_current` |
| freeze/unfreeze/kill 是否成功 | 待运行后填写 | `freeze.status`、`unfreeze.status`、`kill.status` |
| demo/run 是否启动真实 worker | 待运行后填写 | `agents.json` 中至少一个 Agent 的 `pid` 非 0 |
| syscall 是否真实记录 | 待运行后填写 | `syscalls.json` |
| CVM stats 示例 | 待运行后填写 | `context_stats.json` |
| scheduler decision 示例 | 待运行后填写 | `scheduler_decisions.json` |
| tool-timeout fault 示例 | 待运行后填写 | `fault_tool_timeout.status` 与 `fault_tool_timeout.json` |

## 当前 degraded 项

| 模块 | 当前状态 | 说明 |
| --- | --- | --- |
| workspace isolation | `degraded-copy` | 当前 smoke 只验证复制回滚证据，不实现真实 overlayfs mount/commit。 |
| kernel observer | `degraded-proxy` | 当前通过 syscall gateway 的 exec observation 形成 `kernel.exec` 证据，不做 eBPF attach。 |
| LLM provider | `mock` | 当前默认使用 mock provider，不接入真实外部 LLM 凭据。 |
| checkpoint recovery | `checkpoint-light` | 当前恢复 AVP 表、scheduler vruntime 和 CVM page references；page content durable backing 仍是后续工作。 |

## 下一步如何在 openEuler 上运行

1. 在 openEuler 24.03 LTS VM/物理机中以 root 登录，确认 cgroup v2 已挂载。
2. 在仓库根目录执行 `bash scripts/check_openeuler_env.sh`，修复所有 `[FAIL]` 项。
3. 执行 `bash scripts/smoke_openeuler.sh`，确认输出目录生成上述证据文件。
4. 将 `agents.json`、`agent_summary.json`、`syscalls.json`、
   `context_stats.json`、`scheduler_decisions.json`、
   `fault_tool_timeout.json`、`env_check.txt` 的关键内容填入
   “openEuler 实机结果摘要”。
5. 若 `capsule_mode=degraded`，优先检查 `/sys/fs/cgroup` 是否为 `cgroup2fs`、是否 root、`/sys/fs/cgroup/aort.slice` 是否可写。
