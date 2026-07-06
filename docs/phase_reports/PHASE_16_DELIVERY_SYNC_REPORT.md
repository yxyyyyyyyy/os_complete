# PHASE_16_DELIVERY_SYNC_REPORT

## 1. 本次修复目标

本报告是历史交付清理记录，描述的是当时 degraded-real smoke 证据同步状态。
最新 openEuler unified cgroup v2 已经在 PHASE_16 real 报告中跑通：
`capsule_mode=real`、`real_cgroup_v2=true`。本报告中的 degraded 结论只作为
旧环境对照，不代表当前状态。

本次不是新增功能，而是修复交付一致性：同步 degraded-real smoke 证据、
PHASE_15 状态、README 状态说明、competition checklist 和 smoke 证据文件结构。

本阶段未新增 eBPF、overlayfs、真实 LLM provider、UI 美化或复杂 checkpoint。

## 2. go.mod 修复结果

| 项目 | 结果 |
| --- | --- |
| `go.mod` 标准化 | 已是标准 Go module 格式：`module aort-r` 空行后 `go 1.22` |
| 验证命令 | `GOCACHE="$PWD/.cache/go-build" go test ./...` |
| 当前结果 | 通过 |

## 3. PHASE_15 报告同步结果

| 项目 | 结果 |
| --- | --- |
| PHASE_15 模式 | 已同步为 `mode=degraded-real` |
| 当时 capsule_mode | `degraded` |
| 当时 cgroup_path | `degraded://task-1783176060469225965-reviewer` |
| 当时 real cgroup v2 | 未完成 |
| 当前最新状态 | PHASE_16 real 已通过，`capsule_mode=real` |

当时服务器不能作为 real cgroup v2 满血证据机，因为
`stat -fc %T /sys/fs/cgroup` 的结果是 `tmpfs`，不是 `cgroup2fs`。

## 4. Smoke 证据文件

| 文件 | 状态 | 说明 |
| -- | -- | -- |
| `manual_smoke_summary.json` | real summary | 记录 health/demo/agents/syscalls/CVM/scheduler/fault/kill/freeze/unfreeze 的 HTTP 状态。 |
| `agent_summary.json` | real summary | 从手动 smoke 结果提取，明确 `mode=degraded-real`、`real_cgroup_v2=false`。 |
| `env_check.txt` | real output | openEuler/root/Go 1.22.12 可用；cgroup v2 检查失败。 |
| `go_test.txt` | real output | openEuler 上 `go test ./...` 通过。 |
| `health.json` | placeholder | `unavailable_from_manual_run`；HTTP 200 见 `manual_smoke_summary.json`。 |
| `demo_run.json` | placeholder | `unavailable_from_manual_run`；HTTP 202 见 `manual_smoke_summary.json`。 |
| `agents.json` | real output | 包含真实 worker PID 与 degraded cgroup path。 |
| `syscalls.json` | real output | syscall gateway 记录 21 条调用。 |
| `context_stats.json` | real output | CVM API 返回 page/shared/saved metrics。 |
| `scheduler_decisions.json` | real output | scheduler decision API 返回 4 条决策。 |
| `fault_tool_timeout.json` | real output | tool-timeout fault 返回 recovered。 |
| `kill.json` | real output | Agent kill API 返回 200。 |
| `README.md` | explanatory | 说明本目录证据来源、placeholder 语义和下一次 real cgroup v2 要求。 |

## 5. 历史 real / degraded / simulation 状态

### real

- Go test 通过。
- API smoke 跑通。
- syscall gateway 可响应。
- CVM API 可响应。
- scheduler API 可响应。
- fault API 可响应。

### degraded

- cgroup capsule 当时 degraded；当前最新 PHASE_16 real 已通过。
- `freeze/unfreeze` 因 degraded capsule 返回 `409`。
- workspace 仍 degraded-copy。
- kernel observer 仍为 `degraded`。
- checkpoint 是 checkpoint-light。

### simulation/mock

- LLM provider 默认 mock。
- 部分实验仍 degraded/simulation。

## 6. 后续状态

以下目标已在最新 PHASE_16 real 报告中完成：

```bash
stat -fc %T /sys/fs/cgroup
bash scripts/check_openeuler_env.sh
bash scripts/smoke_openeuler.sh
```

输出和证据显示 `cgroup2fs`、`capsule_mode=real`。

## 7. 本次验证命令

```bash
mkdir -p .cache/go-build
GOCACHE="$PWD/.cache/go-build" go test ./...
bash -n scripts/*.sh
```

当前结果：通过。
