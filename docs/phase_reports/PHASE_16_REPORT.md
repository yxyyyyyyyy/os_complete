# PHASE 16 REPORT

## 1. 阶段目标

补齐系统资源压力感知能力，让 AORT-R 不只按 token 和上下文局部性调度，也能展示 PSI / degraded 压力状态，并把压力快照纳入调度事件证据。

## 2. 本阶段实际完成内容

- 新增 `internal/pressure` PSI monitor。
- 支持解析 Linux PSI 行：`avg10`、`avg60`、`avg300`、`total`。
- 新增 `GET /api/pressure/status`。
- `scheduler.selected` 事件新增 `pressure_mode`、`pressure_throttle`、`pressure_cpu_avg10`、`pressure_memory_avg10`、`pressure_io_avg10` 等字段。
- 当 PSI 超过阈值时发布 `scheduler.pressure_throttle`。
- Dashboard Overview 新增 System Pressure 面板。
- `scripts/check_env.sh`、README、部署指南、手工测试指南、竞赛清单同步更新。

## 3. 实现边界

当前实现读取 Linux `/proc/pressure/*` 或同目录下 `*.pressure` 文件。macOS/无 PSI 环境返回：

```text
mode=degraded
degraded=true
```

并给出明确 reason，不伪造 Linux PSI 数据。

## 4. 修改文件清单

| 文件 | 类型 | 说明 |
|---|---|---|
| internal/pressure/monitor.go | 新增 | PSI parser 与 pressure monitor |
| internal/pressure/monitor_test.go | 新增 | parser、throttle、degraded 测试 |
| internal/api/server.go | 修改 | 接入 pressure monitor 和 API |
| internal/api/pressure_api_test.go | 新增 | pressure API 与 scheduler pressure event 测试 |
| dashboard/src/api/client.ts | 修改 | 新增 PressureStatus 类型与 API |
| dashboard/src/stores/runtime.ts | 修改 | 加载 pressure 状态和事件刷新 |
| dashboard/src/pages/Overview.vue | 修改 | 新增 System Pressure 面板 |
| dashboard/src/stores/i18n.ts | 修改 | 新增中英文 pressure 文案 |
| dashboard/src/styles.css | 修改 | 新增 pressure 面板样式 |
| README.md | 修改 | 更新 pressure API 与机制 |
| docs/deployment_openeuler.md | 修改 | 更新 PSI 环境检查和证据映射 |
| docs/testing/manual-test-guide.md | 修改 | 新增 PSI Pressure Check |
| docs/delivery/competition-checklist.md | 修改 | 更新高分证据 |

## 5. 新增 API

```text
GET /api/pressure/status
```

返回 mode、degraded、reason、CPU/memory/IO 的 some/full PSI 指标、throttle、throttle_reason、sampled_at。

## 6. 验证命令

```bash
GOCACHE="$PWD/.cache/go-build" go test ./internal/pressure
GOCACHE="$PWD/.cache/go-build" go test ./internal/api -run 'TestPressureStatusEndpoint|TestSchedulerEventIncludesPressureSnapshot'
```

提交前还应运行：

```bash
GOCACHE="$PWD/.cache/go-build" go test ./...
cd dashboard && npm run test
cd dashboard && npm run build
scripts/run_experiments.sh 5
```

## 7. 当前风险和下一步

- macOS 环境只能展示 degraded pressure 状态；openEuler/Linux 上可读取真实 PSI。
- 后续可把 pressure throttle 从事件证据进一步扩展为真实并发控制或工具沙箱 admission control。
