# PHASE_16_DELIVERY_SYNC_REPORT

## 1. 本次修复目标

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
| 当前 capsule_mode | `degraded` |
| 当前 cgroup_path | `degraded://task-1783176060469225965-reviewer` |
| 当前 real cgroup v2 | 未完成 |
| 当前结论 | degraded smoke passed; real cgroup v2 pending |

当前服务器不能作为 real cgroup v2 满血证据机，因为
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

## 5. 当前 real / degraded / simulation 状态

### real

- Go test 通过。
- API smoke 跑通。
- syscall gateway 可响应。
- CVM API 可响应。
- scheduler API 可响应。
- fault API 可响应。

### degraded

- cgroup capsule 当前 degraded。
- `freeze/unfreeze` 因 degraded capsule 返回 `409`。
- workspace 仍 degraded-copy。
- kernel observer 仍 degraded-proxy。
- checkpoint 是 checkpoint-light。

### simulation/mock

- LLM provider 默认 mock。
- 部分实验仍 degraded/simulation。

## 6. 下一步只剩什么

1. 找一台 unified cgroup v2 的 openEuler 机器。
2. 确认：

   ```bash
   stat -fc %T /sys/fs/cgroup
   ```

   输出 `cgroup2fs`。
3. 重新运行：

   ```bash
   bash scripts/check_openeuler_env.sh
   bash scripts/smoke_openeuler.sh
   ```

4. 目标是让：

   ```json
   "capsule_mode": "real"
   ```

5. 然后再考虑 overlayfs real 和 eBPF real。

## 7. 本次验证命令

```bash
mkdir -p .cache/go-build
GOCACHE="$PWD/.cache/go-build" go test ./...
bash -n scripts/*.sh
```

当前结果：通过。
