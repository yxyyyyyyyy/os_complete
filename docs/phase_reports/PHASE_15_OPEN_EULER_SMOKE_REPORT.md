# PHASE_15_OPEN_EULER_SMOKE_REPORT

mode=degraded-real

> Historical note: this PHASE_15 report is a before/after record from the
> legacy cgroup environment. The latest PHASE_16 real report supersedes it:
> openEuler unified cgroup v2 has passed with `capsule_mode=real` and
> `real_cgroup_v2=true`.

本报告记录 AORT-R 在一台公网 openEuler 24.03 LTS / 类 openEuler 环境服务器上的
真实验证结果。
本次没有伪造 cgroup v2 证据：服务器当前不是统一 cgroup v2 挂载，因此正式
`scripts/smoke_openeuler.sh` 在环境门槛处失败；随后执行了手动 degraded API
smoke，用于证明 Runtime 可启动、可运行 demo、可输出调度/CVM/syscall/fault
证据。

当时结论：已经完成一次 degraded-real smoke；`go test` 通过，API smoke 通过，
但当时尚未获得 `capsule_mode=real` 的 cgroup v2 满血证据。

## 执行环境

| 项目 | 结果 |
| --- | --- |
| 服务器目录 | `/root/aort-r-smoke-git` |
| Git commit | degraded smoke 运行于 `383ae93`，证据已同步至后续交付提交 |
| openEuler 版本 | `openEuler 24.03 (LTS)` |
| kernel 版本 | `6.6.0-112.0.0.104.oe2403.x86_64` |
| 是否 root | 是，`uid=0(root)` |
| Go 工具链 | `go version go1.22.12 linux/amd64` |
| 证据目录 | `experiments/results/openeuler_smoke/` |

## 脚本结果

| 命令 | 结果 | 证据 |
| --- | --- | --- |
| `bash scripts/check_openeuler_env.sh` | 失败，`failures=3 warnings=4` | `experiments/results/openeuler_smoke/env_check.txt` |
| `go test ./...` | 通过 | `experiments/results/openeuler_smoke/go_test.txt` |
| `bash scripts/smoke_openeuler.sh` | 失败，停在 cgroup v2 环境检查 | `experiments/results/openeuler_smoke/smoke_openeuler.log` |
| 手动 degraded API smoke | 完成 | `experiments/results/openeuler_smoke/manual_smoke_summary.json` |

## cgroup / OS 证据

| 项目 | 结果 |
| --- | --- |
| `/sys/fs/cgroup` 类型 | `tmpfs` |
| 期望类型 | `cgroup2fs` |
| `/sys/fs/cgroup` 是否可写 | 否 |
| `/sys/fs/cgroup/aort.slice` | 创建失败 |
| cgroup v2 是否 real | 否 |
| cgroup_path 示例 | `degraded://task-1783176060469225965-reviewer` |
| memory.current 示例 | 无 real cgroup 文件；degraded 汇总为 `0` |
| pids.current 示例 | 无 real cgroup 文件；degraded 汇总为 `0` |

因此，本次不能宣称 real cgroup v2 isolation 已在该服务器通过。当前能证明的是：
Runtime 在 openEuler/root/Go 1.22.12 上可编译测试通过，并能在 cgroup v2 缺失
时进入 degraded capsule 运行。

## API Smoke 结果

| API / 行为 | HTTP / 结果 | 证据 |
| --- | --- | --- |
| `/api/health` | `200` | `manual_smoke_summary.json` |
| `/api/demo/run` | `202` | `manual_smoke_summary.json` |
| `/api/agents` | `200`，存在真实 worker PID | `agents.json`, `agent_summary.json` |
| `/api/context/stats` | `200` | `context_stats.json` |
| `/api/syscalls` | `200`，共 21 条 syscall record | `syscalls.json` |
| `/api/scheduler/decisions` | `200`，共 4 条调度决策 | `scheduler_decisions.json` |
| `freeze` | `409`，degraded capsule 不支持 cgroup freeze | `freeze.json` |
| `unfreeze` | `409`，degraded capsule 不支持 cgroup unfreeze | `unfreeze.json` |
| `kill` | `200` | `kill.json` |
| `/api/demo/fault/tool-timeout` | `202`，`TOOL_TIMEOUT` recovered | `fault_tool_timeout.json` |

`health.json` 与 `demo_run.json` 在当前证据集中是 placeholder，内容明确标记为
`unavailable_from_manual_run`；对应 HTTP 状态以 `manual_smoke_summary.json`
为准，不补写伪造 API body。

## 关键样例

### Agent

```json
{
  "mode": "degraded-real",
  "real_cgroup_v2": false,
  "agent_id": "task-1783176060469225965-reviewer",
  "pid": 7619,
  "capsule_mode": "degraded",
  "cgroup_path": "degraded://task-1783176060469225965-reviewer",
  "memory_current": 0,
  "pids_current": 0,
  "freeze": "409",
  "unfreeze": "409"
}
```

### CVM

```json
{
  "total_pages": 12,
  "shared_pages": 12,
  "saved_bytes": 968,
  "saved_tokens": 237
}
```

### Scheduler

```json
{
  "selected_agent": "task-1783176060469225965-coder",
  "policy": "token-cfs-prefix-affinity",
  "reason": "lowest vruntime; no affinity candidate within threshold",
  "shared_pages": {
    "task-1783176060469225965-coder": 3,
    "task-1783176060469225965-reviewer": 3,
    "task-1783176060469225965-tester": 3
  }
}
```

### Fault

```json
{
  "type": "TOOL_TIMEOUT",
  "status": "RECOVERED",
  "recovery_action": "tool process killed by timeout context"
}
```

## 历史 degraded 项

| 模块 | 当前状态 | 说明 |
| --- | --- | --- |
| capsule / cgroup isolation | `degraded` | 服务器未提供 `/sys/fs/cgroup` 的 `cgroup2fs` 统一挂载。 |
| freeze / unfreeze | `degraded` | 因无 cgroup v2，接口返回 `409`，错误原因已记录。 |
| workspace isolation | `degraded-copy` | 当前 smoke 只验证复制回滚证据，不实现真实 overlayfs mount/commit。 |
| kernel observer | `degraded-proxy` | 当前通过 syscall gateway 的 exec observation 形成 `kernel.exec` 证据，不做 eBPF attach。 |
| LLM provider | `mock` | 当前默认使用 mock provider，不把外部 LLM API key 写入仓库。 |
| checkpoint recovery | `checkpoint-light` | 当前恢复 AVP 表、scheduler vruntime 和 CVM page references；page content durable backing 仍是后续工作。 |

## 下一步 real smoke 要求

1. 准备真正启用 unified cgroup v2 的 openEuler 24.03 环境，要求
   `stat -fc %T /sys/fs/cgroup` 输出 `cgroup2fs`，并且 root 可写
   `/sys/fs/cgroup/aort.slice`。
2. 在该环境先执行 `bash scripts/check_openeuler_env.sh`，确认没有 cgroup v2
   failure。
3. 再执行 `bash scripts/smoke_openeuler.sh`，目标是生成 real
   `memory.current`、`pids.current`、`freeze/unfreeze=2xx` 证据。
4. 保留本 degraded 证据作为历史对照组；最新 cgroup v2 real 证据以 PHASE_16 为准
   区分“环境不满足导致降级”和“Runtime 真实 OS 控制能力”。

必须满足：

```bash
stat -fc %T /sys/fs/cgroup
```

输出：

```text
cgroup2fs
```
