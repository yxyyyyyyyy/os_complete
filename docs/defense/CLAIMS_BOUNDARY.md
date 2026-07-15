# Claims Boundary

| 类别 | 可使用表述 | 证据/限制 |
|---|---|---|
| 可以直接说 | AORT-R 已实现真实 worker、Gateway、CVM 页、page-reference IPC、调度、Timeline 和 replay。 | `internal/*` 与 `go test ./...` |
| 可以直接说 | 已有 openEuler 24.03 LTS real-cgroup-v2、real-overlayfs、ResourceSampler 和 DeepSeek real-api 历史证据。 | `experiments/results/final/FINAL_EVIDENCE_INDEX.json`；说明提交和时间 |
| 可以直接说 | 新 context benchmark 比较三模式、四比例并输出真实计数字段。 | `experiments/results/review_remediation/context_sharing/` |
| 可以直接说 | mock Demo 有 6 Agent、1 次 LLM、5 次工具调用和一次受控故障后继续。 | `real_agent_demo/summary.json` |
| 谨慎说 | AVP 是“OS-inspired runtime execution object”。 | 不是 Linux 新进程类型；真实 worker/cgroup 取决于运行环境 |
| 谨慎说 | cgroup/OverlayFS 可限制资源和工作区故障范围。 | 不等于完整容器/VM 安全沙箱；portable 场景为 degraded |
| 谨慎说 | CVM 和 memfd/mmap 减少 Runtime 层重复传输/materialization。 | 不等于模型 KV Cache；不说 kernel zero-copy |
| 谨慎说 | Prefix Affinity 利用公共上下文页关系。 | 当前命中是 Runtime 调度计数，不是模型缓存命中 |
| 不能说 | AORT-R 新增了 Linux 原生 Agent 进程类型或内核系统调用。 | 当前无内核补丁 |
| 不能说 | 已实现真实 LLM KV Cache 共享或端到端零拷贝。 | 当前没有 provider 内部缓存接口与完整 copy trace |
| 不能说 | 已实现完整 namespace/seccomp/MAC/VM 隔离或完整 Agent OS。 | 相关机制缺失 |
| 不能说 | eBPF 已完整启用或 100% 覆盖。 | 当前 eBPF evidence=degraded |
| 不能说 | 本轮运行了真实 DeepSeek。 | 当前环境 Key unset；只能引用历史 real-api evidence |
