# PHASE 8 REPORT

## 1. 阶段目标

实现 Agent 间高效通信机制，通信传递 CVM page ID，而不是复制大文本。

## 2. 本阶段实际完成内容

- 新增 `internal/ipc` Blackboard。
- 支持 `ipc.publish(topic, page_id)` 与 `ipc.poll(topic)`。
- 统计 `total_messages`, `topic_depth`, `avoided_copy_bytes`。
- Consumer poll 后自动 mount page ID 到自己的 CVM page table。
- 新增 `/api/ipc/metrics` 与 `/api/ipc/topics`。
- Dashboard Context Memory 页展示 IPC 指标和 topic/page ref 表。

## 3. 修改文件清单

| 文件 | 类型 | 说明 |
|---|---|---|
| internal/ipc/blackboard.go | 新增 | IPC Blackboard |
| internal/ipc/blackboard_test.go | 新增 | publish/poll/避免复制测试 |
| internal/syscall/gateway.go | 修改 | 新增 `ipc.publish`, `ipc.poll` |
| internal/api/server.go | 修改 | 新增 IPC API 与 demo 证据 |
| dashboard/src/pages/ContextMemory.vue | 修改 | 展示 IPC 指标 |
| dashboard/src/api/client.ts | 修改 | 新增 IPC 类型和 API client |

## 4. 核心实现说明

Worker 通过 syscall gateway 发布 topic 和 page ID。Blackboard 只保存 page reference，不保存正文。Fixer poll 后，gateway 将返回的 page IDs mount 到 Fixer 的 CVM page table。`avoided_copy_bytes` 以发布 page 的字节数累计，表示没有通过 IPC 复制的上下文字节。

## 5. 新增或修改的 API

| 方法 | 路径 | 作用 | 返回关键字段 |
|---|---|---|---|
| GET | `/api/ipc/metrics` | IPC 汇总指标 | `total_messages`, `avoided_copy_bytes` |
| GET | `/api/ipc/topics` | IPC topic 消息 | `topic`, `publisher`, `page_id` |
| syscall | `ipc.publish` | 发布 page ref | `topic`, `page_id` |
| syscall | `ipc.poll` | 轮询 page ref | `page_ids`, `delivered_messages` |

## 6. 验证命令

```bash
GOCACHE="$PWD/.cache/go-build" go test ./internal/ipc ./internal/syscall ./internal/api
npm run test
```

## 7. 验证结果

- IPC、syscall、api 包测试通过。
- `vue-tsc --noEmit` 通过。

## 8. 当前风险和遗留问题

- 当前 Blackboard 为单机内存实现；后续可持久化到 bbolt。

## 9. 下一阶段建议

将 IPC avoided bytes 纳入 E3 实验结果，并在 Dashboard Experiments 页展示。
