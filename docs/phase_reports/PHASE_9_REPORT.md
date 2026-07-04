# PHASE 9 REPORT

## 1. 阶段目标

补齐 OS 级故障隔离证据：Agent 执行 `rm -rf` 类破坏操作时不污染 base snapshot，并能 rollback。

## 2. 本阶段实际完成内容

- 新增 `internal/workspace`。
- 支持 task base snapshot。
- 支持 per-Agent workspace 准备。
- 支持 degraded-copy rollback。
- 新增 rmrf 故障注入：`POST /api/demo/fault/rmrf`。
- Timeline 记录 `workspace.created`、`workspace.rmrf`、`workspace.rollback`。
- Supervisor 记录 `WORKSPACE_ROLLBACK`。
- Dashboard SSE 订阅 workspace 事件。

## 3. 修改文件清单

| 文件 | 类型 | 说明 |
|---|---|---|
| internal/workspace/manager.go | 新增 | workspace snapshot、准备、rollback |
| internal/workspace/manager_test.go | 新增 | rollback 与 rmrf fault 测试 |
| internal/api/server.go | 修改 | 接入 rmrf fault endpoint |
| internal/api/fault_api_test.go | 修改 | 覆盖 rollback API |
| dashboard/src/stores/runtime.ts | 修改 | 订阅 workspace events |
| docs/testing/manual-test-guide.md | 修改 | 新增 rmrf 验收命令 |
| docs/delivery/competition-checklist.md | 修改 | 更新完成度 |

## 4. 核心实现说明

入口是 `POST /api/demo/fault/rmrf`。Runtime 先创建 base snapshot，再为故障 Agent 准备独立 workspace。故障注入删除 workspace 内全部内容，然后从 base snapshot 复制恢复。Supervisor 记录 `WORKSPACE_ROLLBACK`，细节包含 `workspace_mode`、`rollback_success`、`base_intact`、`removed_entries`。

当前实现是 `degraded-copy`，不伪装成真实 overlayfs；它证明了 rollback 语义。openEuler root VM 上的真实 overlay mount/commit 是下一增强项。

## 5. 新增或修改的 API

| 方法 | 路径 | 作用 | 返回关键字段 |
|---|---|---|---|
| POST | `/api/demo/fault/rmrf` | 注入 workspace 删除并 rollback | `WORKSPACE_ROLLBACK`, `rollback_success`, `base_intact` |

## 6. 验证命令

```bash
GOCACHE="$PWD/.cache/go-build" go test ./internal/workspace ./internal/api
curl -s -X POST http://127.0.0.1:8080/api/demo/fault/rmrf
```

## 7. 验证结果

- `internal/workspace` 测试通过。
- `internal/api` fault 测试通过。
- 本地模式为 `degraded-copy`，符合非 root/macOS 环境预期。

## 8. 当前风险和遗留问题

- 真实 overlayfs mount/commit 尚未实现。
- tool.exec 尚未自动绑定到 workspace manager 的 prepared workspace。

## 9. 下一阶段建议

在 openEuler root VM 上补真实 overlayfs mount，并将 tool.exec cwd 接入 workspace manager。
