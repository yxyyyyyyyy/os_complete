# AORT-R 评审整改最终报告

## 结论

本轮整改已完成代码、场景、统计、证据、概要设计和答辩源稿的闭环。对外定位从“六个并列功能”收敛为两个问题：多 Agent 资源竞争/故障扩散/工作区污染，以及共享上下文的重复写入/传输/materialization。

当前验收不是无条件的“全部完成”。便携场景和 mock Demo 已重复运行并通过；当前 Darwin 宿主不能重跑 real-only openEuler，且 `DEEPSEEK_API_KEY` 未设置，所以本轮真实 openEuler 和 DeepSeek 调用均未执行。仓库保留 2026-07-07 的 openEuler 24.03 LTS、real cgroup v2、real OverlayFS 和 DeepSeek real-api 历史证据，并与本轮 portable evidence 分层引用。

## 老师意见闭环

| 老师意见 | 修改内容与位置 | 命令/证据 | 状态与边界 |
|---|---|---|---|
| 资源干扰、隔离、安全和测试方法没有讲透 | `internal/review/resource_isolation.go`；`docs/design/04_resource_isolation_design.md`、`06_security_and_boundaries.md`、`THREAT_MODEL.md` | `scenario resource-isolation`；`resource_isolation/summary.json`、raw、CSV | PASS；本轮为 portable degraded，不宣称真实 cgroup 性能收益 |
| 技术路线过快，报告缺少问题和概要设计 | `docs/design/01..10`、架构/生命周期/故障/上下文数据流图；旧报告增加能力边界 | `docs/design/README.md` 及代码/证据链接检查 | PASS |
| 通信效率缺少量化 | `internal/review/context_sharing.go`、`metrics.go`；`docs/experiments/CONTEXT_SHARING_BENCHMARK.md` | `scenario context-sharing`；240 个 measured observations、CSV、raw | PASS；数据是 Runtime 计数，不是模型 KV Cache/端到端吞吐结论 |
| 真实服务、Demo 和实际效果不足 | `internal/review/agent_demo.go`；`docs/design/09_real_agent_demo.md`、`docs/defense/DEMO_SCRIPT.md` | 本轮 mock Demo；历史 `experiments/results/final/FINAL_EVIDENCE_INDEX.json` | PARTIAL；本轮 Key unset、real-only 宿主不可用，未冒充新实测 |
| 应聚焦 1-2 个赛题问题并做场景化评价 | README、设计文档、PPT、讲稿、专家回复统一为两条主线；其他模块降为支撑能力 | `REVIEW_TO_TASK_MATRIX.md`、`REVIEW_EVIDENCE_INDEX.json` | PASS |

## 代码与命令

新增正式入口：

```bash
aortctl scenario resource-isolation
aortctl scenario context-sharing
aortctl scenario real-agent-demo --provider mock|deepseek
aortctl evidence review-final
```

实现集中在 `internal/review/metrics.go`、`resource_isolation.go`、`context_sharing.go`、`agent_demo.go` 和 `review_final.go`，通过 `internal/experiment/review_scenarios.go` 接入现有 CLI。实现复用 CVM、IPC、Scheduler、Gateway、Timeline 和 workspace 路径，没有建立与现有 Runtime 平行的替代框架。

## 实测结果

### 资源隔离/故障控制

- 配置：warmup 3，20 measured runs/mode，固定 seed 20260713，三模式共 60 个 measured runs。
- `baseline`、`isolation-only`、`aort-r` 的 task success rate 均为 1；失败 raw 不会被丢弃。
- `aort-r` 正常 Agent completion rate 均值为 1，fault containment scope 均值为 1 个 Agent。
- `aort-r` lowerdir hash unchanged 均值为 1，cross-agent contamination 均值为 0。
- 本机 `evidence_mode=degraded`；这些值验证 portable workload、工作区哈希、污染检测和清理逻辑，不证明本机执行了真实 cgroup 限额或 OverlayFS mount，也不据此宣称性能提升。

证据：`experiments/results/review_remediation/resource_isolation/summary.json`、`comparison.csv`、`report.md` 和 60 个 raw JSON。

### 上下文复用/通信优化

- 配置：6 Agent，4096 logical bytes/Agent，warmup 3，20 measured runs/variant；三模式 x 四比例共 240 个 measured runs。
- full-copy 在 0/25/50/75% 时均传输 24576 bytes。
- aort-r 在 0/25/50/75% 时传输 24576/18496/12352/6208 bytes；derived saved bytes 为 0/6080/12224/18368。
- 50% 时 aort-r 每轮记录 5 次 Prefix Affinity 命中；所有变体 success rate 为 1。
- `peak_rss_bytes` 明确标为 unsupported；CVM/page-reference 结果不等于真实推理引擎内部 KV Cache，也不称 kernel zero-copy。

证据：`experiments/results/review_remediation/context_sharing/summary.json`、`comparison.csv`、`report.md` 和 240 个 raw JSON。

### 真实服务可用性 Demo

- 本轮 mock：6 Agent、1 次 `llm.call`、5 次 `tool.exec`。
- Fault-Agent 的受控 `false` 命令返回错误；Tester 和 Reviewer 随后继续执行，最终 `status=passed`。
- 当前 `DEEPSEEK_API_KEY=unset`，因此没有发起新的真实 API 请求。
- 历史 final 索引记录 `deepseek_real_api=true`，来源是 openEuler 历史运行；它只证明已有可用性，不作为本轮网络/性能数据。

证据：`experiments/results/review_remediation/real_agent_demo/summary.json`、`timeline.json`、`final_result.json`、`report.md`。

## 最终验证

执行结果：

- `gofmt -l $(rg --files -g '*.go')`：exit 0，无输出。
- `GOCACHE=/private/tmp/aort-gocache-main-final go test -count=1 ./...`：exit 0。
- `GOCACHE=/private/tmp/aort-gocache-vet-final go vet ./...`：exit 0，无输出。
- 在提交 `4d43fc3` 的临时克隆运行 `scripts/competition_verify.sh`：exit 0；Go、E1/E2、software/workspace/final artifacts 通过，Darwin 的 openEuler env/smoke 按设计 degraded。
- 同一临时克隆运行 `scripts/competition_verify_real.sh`：exit 1，并在第一个 `real_env` 步骤停止；原因是无 root、非 openEuler、cgroup fs 不是 cgroup2fs、cgroup 不可写、未列出 OverlayFS。后续危险 real-only 步骤未运行。
- `evidence final --out /private/tmp/aort-final-evidence-check-20260715`：exit 0，`missing_files=[]`；历史索引未覆盖。
- `evidence review-final --out experiments/results/review_final`：exit 0，三场景 present/valid/passed，`all_required_passed=true`。
- 306 个 JSON、CSV 列、报告引用、状态矛盾、密钥/IP 字面量和能力边界扫描通过；未发现 AORT-R 进程、mount 或场景临时目录残留。完整命令与范围见 `FINAL_CHECKLIST.md`。

## 能力边界和未完成项

1. 需要在同一台 openEuler 24.03 LTS 主机重跑三个新场景，才能把新场景的 cgroup/OverlayFS/ResourceSampler evidence 从 degraded 提升为 real-only。
2. 需要提供环境变量 Key 后运行 `real-agent-demo --provider deepseek`，才能形成与当前提交绑定的新 real-api 证据；Key 不得进入仓库或日志。
3. CVM 尚未接入真实推理引擎 KV Cache 命中计数，memfd/mmap 和 page reference 也没有端到端 copy trace。
4. 当前没有完整 namespace、seccomp、MAC 或 VM 级隔离；项目不是完整容器安全沙箱或完整 Agent OS。
5. eBPF 仍是可选增强，当前提交证据为 degraded；单机结果不能外推为分布式或多租户安全保证。

## Git 与交付范围

- 分支：`codex/aort-r-upgrade`
- P9 验证基线：`4d43fc323c52f24b4d8876dcf437c3aae56c5ffc`
- P9 最终提交：包含本报告的提交，以交付时 `git rev-parse HEAD` 为准。
- 生成证据时 `git_dirty=true`，符合实际工作树。

以下用户已有 Dashboard 修改明确保留且不纳入整改提交：

- `dashboard/src/api/client.ts`
- `dashboard/src/pages/Overview.vue`
- `dashboard/src/stores/runtime.ts`
- `dashboard/src/styles.css`

同时保留且不暂存：`deliverables/`、`docs/superpowers/plans/2026-07-06-runtime-real-os-evidence-plan.md`、`experiments/results/audit_all/`、`改进.md`。因此最终提交后工作树仍为 dirty；这不是隐藏失败，而是限定提交范围并保护用户现有工作。
