# PHASE_REPAIR_REPORT

## 本次修复范围

本次只处理格式与编译可信度问题：`go.mod`、Go 文件格式、README 本机路径、`.gitignore` 格式、全量测试记录和当前真实能力清单。没有新增 eBPF、checkpoint、LLM provider、UI 或其他功能。

## 命令执行结果

- `gofmt ./...`：未完成。原样执行结果为 `stat ./...: no such file or directory`，原因是 `gofmt` 不接受 `./...` 作为包通配参数。
- 等价全量格式化：已完成。实际执行 `rg --files -g '*.go' | xargs gofmt -w`，退出码为 0。
- `go test ./...`：原样执行未通过，失败原因是默认 Go 构建缓存目录不可写。

完整失败原因：

```text
FAIL	./... [setup failed]
# ./...
pattern ./...: open /Users/yxy/Library/Caches/go-build/eb/eb9dbe0d673e53070bda9ddc9c0ca534382bd0d82a0d4e8f9e4fea589d4865f8-d: operation not permitted
FAIL
go: failed to trim cache: open /Users/yxy/Library/Caches/go-build/trim.txt: operation not permitted
```

- 仓库内缓存复测：已通过。实际执行 `mkdir -p .cache/go-build` 和 `GOCACHE="$PWD/.cache/go-build" go test ./...`，全部 Go package 通过。

## 当前真实 syscall 列表

- `context.materialize`
- `context.write_delta`
- `ipc.publish`
- `ipc.poll`
- `llm.call`
- `tool.exec`
- `agent.spawn`
- `agent.report`

## 当前真实 API 列表

- `GET /api/health`
- `GET /api/events`
- `POST /api/demo/run`
- `POST /api/demo/fault/tool-timeout`
- `POST /api/demo/fault/timeout`
- `POST /api/demo/fault/rmrf`
- `POST /api/demo/fault/workspace-rmrf`
- `GET /api/faults`
- `GET /api/agents`
- `POST /api/agents/{agent_id}/freeze`
- `POST /api/agents/{agent_id}/unfreeze`
- `POST /api/agents/{agent_id}/kill`
- `GET /api/context/pages`
- `GET /api/context/stats`
- `GET /api/context/agents/{agent_id}/pagetable`
- `GET /api/ipc/metrics`
- `GET /api/ipc/topics`
- `GET /api/kernel/status`
- `GET /api/kernel/events`
- `GET /api/pressure/status`
- `GET /api/checkpoints`
- `GET /api/recovery/status`
- `GET /api/syscalls`
- `GET /api/scheduler/decisions`
- `POST /api/scheduler/policy`
- `GET /api/experiments/results`
- `GET /api/tasks`
- `GET /api/tasks/{task_id}/dag`

## 当前 real 模块

- AVP 生命周期与状态模型。
- worker 进程启动、UDS 注册、心跳与 syscall 转发。
- syscall gateway 记录、审计与 SSE 事件。
- CVM 内存页存储、page table、context materialize/write delta 与节省字节统计。
- IPC Blackboard 发布、轮询、topic 与 avoided-copy 指标。
- FIFO、token-CFS、token-CFS-prefix-affinity 调度策略与决策日志。
- Supervisor fault record 与 `tool.exec` timeout 注入。
- checkpoint JSON 存储、列表读取和启动恢复报告。
- 实验结果读取与 Go 侧实验 runner。
- REST/SSE API 服务。

## 当前 degraded 模块

- cgroup capsule：Linux cgroup v2/root 环境可走 real；macOS、非 Linux 或无权限环境进入 `capsule_mode=degraded`。
- workspace 隔离：当前是 `degraded-copy` 复制回滚，不是真实 overlayfs mount/commit。
- kernel observer：当前是 `degraded`，通过 syscall gateway 的 exec observation 形成 `kernel.exec` 证据；独立 eBPF smoke 已实现，但提交证据需等 openEuler/Linux 复跑证明 `real-ebpf`。
- pressure monitor：Linux PSI 可读时为 `psi`；无 `/proc/pressure` 或不可读时进入 degraded。
- checkpoint recovery：当前为 `checkpoint-light`，恢复 AVP 表、scheduler vruntime 与 CVM page references；page content 持久恢复仍依赖未来 durable page backing。

## 当前 simulation/mock 模块

- LLM Router 当前默认 provider 是 `mock`。
- 实验在本地不可用能力上会体现 degraded 或 simulation 证据。
- demo 路径在未启用真实 worker 配置时会走 runtime 内部演示流程。

## 下一步最该做的 3 件事

1. 保留最新 openEuler unified cgroup v2 real smoke 作为主证据，并补充多 Agent 隔离与 limit 生效脚本输出。
2. 将 workspace `degraded-copy` 升级为 openEuler 上的真实 overlayfs mount/commit/rollback，并保留 degraded fallback。
3. 在 openEuler/Linux root 或 capable VM 上复跑 eBPF smoke；只有 load/attach、worker PID observed 和 cleanup 成功时才把 eBPF 证据推进到 `real-ebpf`。
