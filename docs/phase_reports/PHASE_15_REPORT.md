# PHASE 15 REPORT

## 1. 阶段目标

补齐系统级可观测性中的 Kernel Observer 证据，让 Timeline 不只展示应用层和 syscall audit，也有明确的 kernel lane。当前阶段不伪装真 eBPF，采用可运行、可验证的 `degraded-proxy` 模式。

## 2. 本阶段实际完成内容

- 新增 `internal/kernel` Observer，提供状态、事件记录和 SSE 事件发布。
- 新增 `GET /api/kernel/status` 与 `GET /api/kernel/events`。
- `tool.exec` 通过 syscall gateway 上报 exec observation，生成 `kernel.exec` 事件。
- Observer 启动时检测 BTF/bpffs 条件，并发布 `kernel.observer_disabled`。
- Dashboard Timeline 新增 Kernel Mode、Probe、Kernel Events、BTF 指标。
- `scripts/check_env.sh` 新增 BTF 与 bpffs 检查。
- 更新 README、openEuler 部署指南、手工测试指南和竞赛清单。

## 3. 实现边界

当前模式为：

```text
mode=degraded-proxy
probe=syscall-gateway-proxy
```

含义是：AORT-R 已经把 exec 证据纳入 kernel lane，但事件来源是 syscall gateway 对 tool process 的观察，不是假装来自真实 eBPF。后续在 openEuler root VM 上接入 `sched:sched_process_exec` 后，可将 probe 切换为真实 eBPF。

## 4. 修改文件清单

| 文件 | 类型 | 说明 |
|---|---|---|
| internal/kernel/observer.go | 新增 | Kernel Observer 与降级代理事件 |
| internal/kernel/observer_test.go | 新增 | Observer 状态与事件测试 |
| internal/syscall/gateway.go | 修改 | tool.exec 上报 exec observation |
| internal/syscall/gateway_test.go | 修改 | 验证 tool.exec 产生 exec observation |
| internal/api/server.go | 修改 | 接入 Kernel Observer 和 API |
| internal/api/kernel_api_test.go | 新增 | 验证 kernel status/events API |
| dashboard/src/api/client.ts | 修改 | 新增 KernelStatus/KernelEvent 类型与 API |
| dashboard/src/stores/runtime.ts | 修改 | 加载 kernel 状态/事件并订阅 SSE |
| dashboard/src/pages/Timeline.vue | 修改 | 新增 kernel 指标面板 |
| dashboard/src/stores/i18n.ts | 修改 | 新增中英文文案 |
| scripts/check_env.sh | 修改 | 新增 BTF/bpffs 检查 |
| README.md | 修改 | 更新 kernel observer 入口 |
| docs/deployment_openeuler.md | 修改 | 更新环境检查和证据映射 |
| docs/testing/manual-test-guide.md | 修改 | 新增 Kernel Observer 验收 |
| docs/delivery/competition-checklist.md | 修改 | 更新高分证据和剩余项 |

## 5. 新增 API

```text
GET /api/kernel/status
GET /api/kernel/events
```

`/api/kernel/status` 返回 enabled、mode、probe、reason、btf_available、bpffs_ready、event_count。

`/api/kernel/events` 返回 command、args、pid、workspace、status、mode、probe、timestamp 等 exec 事件字段。

## 6. 验证命令

```bash
GOCACHE=/Users/yxy/Documents/比赛/操作系统/.cache/go-build go test ./internal/kernel
GOCACHE=/Users/yxy/Documents/比赛/操作系统/.cache/go-build go test ./internal/syscall -run TestGatewayToolExecReportsKernelExecObservation
GOCACHE=/Users/yxy/Documents/比赛/操作系统/.cache/go-build go test ./internal/api -run TestKernelAPIsExposeObserverStatusAndExecEvents
```

提交前还应运行全量验证：

```bash
GOCACHE=/Users/yxy/Documents/比赛/操作系统/.cache/go-build go test ./...
cd dashboard && npm run test
cd dashboard && npm run build
scripts/run_experiments.sh 5
```

## 7. 当前风险和下一步

- 真实 eBPF attachment 尚未实现，当前为明确标记的 degraded-proxy。
- 下一步可在 openEuler 24.03 root VM 上实现 `sched_process_exec` eBPF 程序，并保留当前 proxy 作为 fallback。
