# PHASE_16_OPEN_EULER_REAL_CGROUP_REPORT

mode=real-passed

本报告记录 AORT-R 在远程 openEuler 24.03 LTS 服务器上完成 unified cgroup v2
切换后的真实 OS 证据。本报告不伪造 degraded 或 simulation 结果。

## 1. 结论

PHASE_16 已拿到 real cgroup v2 capsule 证据。

| 验收项 | 真实结果 | 证据位置 |
| --- | --- | --- |
| Git commit | `7d939c2` | `go_test_cgroupv2_7d939c2.txt` |
| OS | `openEuler 24.03 (LTS)` | `env_check.json` |
| Kernel | `6.6.0-112.0.0.104.oe2403.x86_64` | `env_check.json` |
| root 用户 | `uid=0(root)` | `env_check.json` |
| `/sys/fs/cgroup` | `cgroup2fs` | `env_check.json` |
| `/sys/fs/cgroup` 可写 | `true` | `env_check.json` |
| `capsule_mode` | `real` | `capsule_real.json` |
| `cgroup_path` | `/sys/fs/cgroup/aort.slice/task-1783187945630244755-tester` | `capsule_real.json` |
| `memory.current` | `8192` | `cgroup_memory_current.txt` |
| `pids.current` | `5` | `cgroup_pids_current.txt` |
| freeze/unfreeze/kill | `200 / 200 / 200` | `capsule_real.json` |
| `go test ./...` | pass | `go_test_cgroupv2_7d939c2.txt` |
| smoke test | pass | `smoke_cgroupv2_7d939c2.log` |

证据包：

```text
experiments/results/openeuler_smoke/aort-r-openeuler-7d939c2-cgroupv2-real-evidence.tgz
```

## 2. 执行环境

远程服务器：

```text
root@116.204.94.247
/root/aort-r-smoke-7d939c2
```

环境摘要来自真实脚本输出：

```json
{
  "evidence_mode": "real",
  "cgroup": {
    "fs_type": "cgroup2fs",
    "is_cgroup2fs": true,
    "writable": true,
    "aort_slice": "exists"
  },
  "is_root": true,
  "failures": 0,
  "warnings": 4
}
```

仍为 warning/degraded 的环境项：

| 项 | 状态 | 影响 |
| --- | --- | --- |
| overlayfs | not listed in `/proc/filesystems` | 不影响本阶段 cgroup capsule 验收 |
| PSI | `/proc/pressure` unavailable | 压力观测仍不能宣传为 real PSI |
| Node/npm | not found | 前端 smoke 未在该服务器验证 |

## 3. 已执行命令

```bash
export GOTOOLCHAIN=go1.22.12
export GOPROXY=https://goproxy.cn,direct
cd /root/aort-r-smoke-7d939c2

GOCACHE="$PWD/.cache/go-build" go test ./...
bash scripts/check_openeuler_env.sh
bash scripts/smoke_openeuler.sh
```

远程 smoke 输出目录：

```text
/root/aort-r-smoke-7d939c2/experiments/results/openeuler_smoke/
```

本地同步后的证据目录：

```text
experiments/results/openeuler_smoke/
```

## 4. Capsule 真实证据

`capsule_real.json` 关键内容：

```json
{
  "evidence_mode": "real",
  "os": "openEuler 24.03 LTS",
  "cgroup_fs": "cgroup2fs",
  "capsule_mode": "real",
  "cgroup_path": "/sys/fs/cgroup/aort.slice/task-1783187945630244755-tester",
  "memory_current": 8192,
  "pids_current": 5,
  "freeze": "200",
  "unfreeze": "200",
  "kill": "200"
}
```

本次修复点：

```text
internal/capsule/manager.go
```

`Prepare` 在创建 agent 子 cgroup 前会读取父 cgroup 的 `cgroup.controllers`，
并对可用的 `cpu memory pids` 写入 `cgroup.subtree_control`。这是 unified cgroup v2
下子 cgroup 生成 `cpu.max`、`memory.max`、`pids.max` 的必要条件。

## 5. Runtime API 证据

Smoke test 调用并保存了以下接口：

| API | 状态 | 证据位置 |
| --- | --- | --- |
| `GET /api/health` | `200` | `health.json` |
| `POST /api/demo/run` | `202` | `demo_run.json` |
| `GET /api/agents` | `200` | `agents.json` |
| `GET /api/capsules` | `200` | `capsules.json` |
| `GET /api/context/stats` | `200` | `context_stats.json` |
| `GET /api/syscalls` | `200` | `syscalls.json` |
| `GET /api/scheduler/decisions` | `200` | `scheduler_decisions.json` |
| `POST /api/capsules/:id/freeze` | `200` | `freeze.json` |
| `POST /api/capsules/:id/unfreeze` | `200` | `unfreeze.json` |
| `POST /api/capsules/:id/kill` | `200` | `kill.json` |
| `POST /api/demo/fault/tool-timeout` | `202` | `fault_tool_timeout.json` |

## 6. CVM / Syscall / Scheduler / Fault 示例

CVM stats：

```json
{
  "total_pages": 12,
  "shared_pages": 12,
  "saved_bytes": 968,
  "saved_tokens": 237
}
```

syscall 记录包含真实 runtime API 输出，例如：

```json
{
  "name": "llm.call",
  "status": "OK",
  "duration_ms": 1,
  "input_size": 18,
  "output_size": 250
}
```

scheduler decision 示例：

```json
{
  "policy": "token-cfs-prefix-affinity",
  "selected_agent": "task-1783187945630244755-planner",
  "reason": "lowest vruntime; no previous prefix group"
}
```

tool-timeout fault 示例：

```json
{
  "type": "TOOL_TIMEOUT",
  "status": "RECOVERED",
  "recovery_action": "tool process killed by timeout context",
  "details": {
    "syscall": "tool.exec",
    "syscall_status": "TIMEOUT"
  }
}
```

## 7. 当前边界

当前已经可以宣传为 real 的模块：

| 模块 | evidence mode |
| --- | --- |
| openEuler 编译与 `go test ./...` | real |
| worker PID | real |
| cgroup v2 capsule | real |
| memory/pids/cpu cgroup counters | real |
| freeze/unfreeze/kill | real |
| syscall gateway record | real runtime output |
| CVM stats | real runtime output |
| scheduler decisions | real runtime output |
| tool-timeout fault recovery | real runtime output |

当前仍需诚实标注的模块：

| 模块 | evidence mode | 原因 |
| --- | --- | --- |
| PSI pressure | degraded | 该 openEuler 实例无 `/proc/pressure` |
| overlayfs workspace isolation | degraded/pending | 该实例 `/proc/filesystems` 未列出 overlay |
| LLM provider | simulation/mock | 当前 smoke 使用 mock/demo provider |
| Dashboard openEuler smoke | pending | 服务器未安装 Node/npm |
| eBPF syscall tracing | not implemented | 本阶段未做 eBPF |

## 8. 下一步建议

1. 在交付包中把 `PHASE_15` 作为 degraded 对照，把本报告作为 real cgroup v2 主证据。
2. 为答辩准备一张截图或录屏：展示 `env_check.json`、`capsule_real.json`、Dashboard evidence 页。
3. 下一阶段优先补 real PSI 或 overlayfs 二选一；不要再增加概念，先补能在 openEuler 上拿到的 OS 证据。
