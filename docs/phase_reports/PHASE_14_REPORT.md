# PHASE 14 REPORT

## 1. 阶段目标

补齐 V3 高分项中的 systemd 守护进程部署与轻量 checkpoint 启动恢复，让 daemon crash 后的恢复能力有代码、API、脚本和文档证据。

## 2. 本阶段实际完成内容

- 新增 checkpoint 恢复报告模型：按 task 聚合最新 snapshot，输出 completed/ready Agent、页表引用数和恢复模式。
- `aortd` 启动时扫描 `data_dir/checkpoints`，恢复任务索引，并发布 `checkpoint.recovered`、`runtime.recovered` 事件。
- 新增 `GET /api/recovery/status`，用于展示启动恢复证据。
- Dashboard Overview 新增 Checkpoint Recovery 面板，支持中英文展示。
- 新增 systemd 单元 `deploy/systemd/aortd.service`。
- 新增 daemonkill 演示脚本 `scripts/demo-daemonkill.sh`。
- 更新 README、openEuler 部署指南、手工测试指南和竞赛清单。

## 3. 修改文件清单

| 文件 | 类型 | 说明 |
|---|---|---|
| internal/checkpoint/checkpoint.go | 修改 | 新增 `RecoverAll` 与恢复摘要 |
| internal/checkpoint/checkpoint_test.go | 修改 | 新增恢复摘要测试 |
| internal/api/server.go | 修改 | 启动恢复、恢复 API、恢复事件 |
| internal/api/demo_api_test.go | 修改 | 新增二次启动恢复测试 |
| dashboard/src/api/client.ts | 修改 | 新增 RecoveryStatus 类型与 API |
| dashboard/src/stores/runtime.ts | 修改 | 加载恢复状态，订阅恢复事件 |
| dashboard/src/pages/Overview.vue | 修改 | 新增恢复证据面板 |
| dashboard/src/stores/i18n.ts | 修改 | 新增恢复面板中英文文案 |
| dashboard/src/styles.css | 修改 | 新增恢复面板样式 |
| deploy/systemd/aortd.service | 新增 | openEuler/Linux systemd 参考单元 |
| configs/openeuler-dev.yaml | 新增 | openEuler/systemd 示例配置 |
| scripts/demo-daemonkill.sh | 新增 | 守护进程崩溃恢复演示 |
| docs/deployment_openeuler.md | 修改 | 新增 systemd 和 daemonkill 指南 |
| docs/testing/manual-test-guide.md | 修改 | 新增 checkpoint startup recovery 验收 |
| docs/delivery/competition-checklist.md | 修改 | 更新高分证据与剩余项 |
| README.md | 修改 | 更新快速验证和已实现机制 |

## 4. 核心实现说明

本阶段实现的是轻量恢复：恢复 AVP 表、scheduler vruntime、CVM 页表引用和任务索引。由于当前 CVM 页内容还未持久化，恢复报告明确标记 `degraded=true`，不假装已完成 durable KV/page-content checkpoint。这个边界对答辩更稳：现有能力可运行、可验证，后续增强路径清晰。

## 5. 新增或修改的 API

```text
GET /api/recovery/status
```

返回字段包括：

- `mode`: 当前为 `checkpoint-light`。
- `degraded`: 当前为 `true`，表示轻量恢复。
- `task_count`: 启动时恢复到 runtime index 的任务数。
- `recovered_tasks`: 每个任务的 sequence、status、agent_count、completed_agents、ready_agents、page_table_refs、scheduler_vruntime。

## 6. 验证命令

```bash
GOCACHE="$PWD/.cache/go-build" go test ./internal/checkpoint
GOCACHE="$PWD/.cache/go-build" go test ./internal/api -run TestServerStartupRecoversCheckpointState
cd dashboard && npm run test
```

全量提交前还应运行：

```bash
GOCACHE="$PWD/.cache/go-build" go test ./...
cd dashboard && npm run build
scripts/run_experiments.sh 5
```

## 7. 当前风险和遗留问题

- 需要在 openEuler VM 上用真实 systemd 复测 `scripts/demo-daemonkill.sh`。
- 当前 checkpoint 恢复不包含持久化 CVM page content 和 overlay upper 层快照。
- 真实 overlayfs mount/commit、eBPF execve observer、PSI 仍是后续冲刺项。
