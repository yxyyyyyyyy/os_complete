# PHASE_15_OPEN_EULER_SMOKE_REPORT

本报告用于记录 AORT-R 在 openEuler 24.03 LTS / Linux root / cgroup v2 环境下的 smoke test 证据。本阶段只验证当前 Runtime 的真实 OS 证据，不新增 overlayfs、eBPF、UI 或复杂功能。

## 运行命令

```bash
bash scripts/check_openeuler_env.sh
bash scripts/smoke_openeuler.sh
```

证据目录：

```text
experiments/results/openeuler_smoke/
```

## 环境证据

| 项目 | 结果 | 证据来源 |
| --- | --- | --- |
| openEuler 版本 | 待 openEuler 实机运行后填写 | `env_check.txt` 中 `/etc/os-release` |
| kernel 版本 | 待 openEuler 实机运行后填写 | `env_check.txt` 中 `uname -a` |
| 是否 root | 待 openEuler 实机运行后填写 | `env_check.txt` 中 `id` |
| cgroup v2 是否 real | 待 openEuler 实机运行后填写 | `env_check.txt` 中 `stat -fc %T /sys/fs/cgroup`，期望 `cgroup2fs` |

## Runtime OS 证据

| 项目 | 结果 | 证据来源 |
| --- | --- | --- |
| cgroup_path 示例 | 待 openEuler 实机运行后填写 | `agent_summary.json` 的 `cgroup_path` |
| memory.current 示例 | 待 openEuler 实机运行后填写 | `cgroup_memory_current.txt` 或 `agent_summary.json` 的 `memory_current` |
| pids.current 示例 | 待 openEuler 实机运行后填写 | `cgroup_pids_current.txt` 或 `agent_summary.json` 的 `pids_current` |
| freeze 是否成功 | 待 openEuler 实机运行后填写 | `freeze.status` 与 `freeze.json` |
| unfreeze 是否成功 | 待 openEuler 实机运行后填写 | `unfreeze.status` 与 `unfreeze.json` |
| kill 是否成功 | 待 openEuler 实机运行后填写 | `kill.status` 与 `kill.json` |
| demo/run 是否启动真实 worker | 待 openEuler 实机运行后填写 | `agents.json` 中至少一个 Agent 的 `pid` 非 0 |
| syscall 是否真实记录 | 待 openEuler 实机运行后填写 | `syscalls.json` |
| CVM stats 示例 | 待 openEuler 实机运行后填写 | `context_stats.json` |
| scheduler decision 示例 | 待 openEuler 实机运行后填写 | `scheduler_decisions.json` |
| tool-timeout fault 示例 | 待 openEuler 实机运行后填写 | `fault_tool_timeout.status` 与 `fault_tool_timeout.json` |

## 当前仍 degraded 的模块

| 模块 | 当前状态 | 说明 |
| --- | --- | --- |
| workspace isolation | `degraded-copy` | 当前 smoke 只验证复制回滚证据，不实现真实 overlayfs mount/commit。 |
| kernel observer | `degraded-proxy` | 当前通过 syscall gateway 的 exec observation 形成 `kernel.exec` 证据，不做 eBPF attach。 |
| LLM provider | `mock` | 当前默认使用 mock provider，不接入真实外部 LLM 凭据。 |
| checkpoint recovery | `checkpoint-light` | 当前恢复 AVP 表、scheduler vruntime 和 CVM page references；page content durable backing 仍是后续工作。 |

## 下一步建议

1. 在 openEuler root/cgroup v2 环境运行 smoke，并将 `experiments/results/openeuler_smoke/` 中的实际证据填入本报告。
2. 若 `capsule_mode=degraded`，优先检查 `/sys/fs/cgroup` 是否为 `cgroup2fs`、是否 root、`/sys/fs/cgroup/aort.slice` 是否可写。
3. smoke 证据稳定后，再进入后续阶段评估真实 overlayfs 和 eBPF；本阶段不实现这些能力。
