# P9 最终验收清单

## 验收快照

- 日期：2026-07-15
- 分支：`codex/aort-r-upgrade`
- 验证基线提交：`4d43fc323c52f24b4d8876dcf437c3aae56c5ffc`
- 当前宿主：Darwin/arm64，Go 1.25.3
- 结论：代码、便携场景、证据结构和文档验收通过；本机不具备 real-only openEuler 条件，当前环境也没有 `DEEPSEEK_API_KEY`，两项均按未运行/环境不可用记录。

## 命令验收

| 检查 | 命令 | 结果 | 说明 |
|---|---|---|---|
| Go 格式 | `gofmt -l $(rg --files -g '*.go')` | PASS | exit 0，无输出 |
| Go 全量测试 | `GOCACHE=/private/tmp/aort-gocache-main-final go test -count=1 ./...` | PASS | 所有 Go package 通过 |
| Go 静态检查 | `GOCACHE=/private/tmp/aort-gocache-vet-final go vet ./...` | PASS | exit 0，无输出 |
| 原有一键验收 | `bash scripts/competition_verify.sh` | PASS_WITH_DEGRADED | 在临时克隆执行；Go、E1/E2、Demo、workspace 和 final artifacts 通过，Darwin 上 env/smoke 按设计 degraded |
| real-only 门禁 | `bash scripts/competition_verify_real.sh` | ENV_UNAVAILABLE | exit 1，停在首个 `real_env`；非 openEuler、无 root、无可写 cgroup2、无 OverlayFS |
| 资源隔离场景 | `aortctl scenario resource-isolation --mode all --warmup 3 --runs 20` | PASS_WITH_DEGRADED | 3 模式，60 个 measured runs；便携宿主不宣称真实 cgroup/OverlayFS |
| 上下文场景 | `aortctl scenario context-sharing --mode all --shared-ratio all --warmup 3 --runs 20` | PASS_WITH_DEGRADED | 3 模式 x 4 比例，240 个 measured runs |
| mock Demo | `aortctl scenario real-agent-demo --provider mock` | PASS | 6 Agent、1 次 `llm.call`、5 次 `tool.exec`，故障后继续 |
| DeepSeek Demo | `aortctl scenario real-agent-demo --provider deepseek` | NOT_RUN | 当前 `DEEPSEEK_API_KEY=unset`；历史 real-api 证据保留且脱敏 |
| 旧总证据兼容 | `aortctl evidence final --out /private/tmp/aort-final-evidence-check-20260715` | PASS | 不覆盖历史索引；`missing_files=[]`，当前快照 `git_dirty=true` |
| 整改总证据 | `aortctl evidence review-final --out experiments/results/review_final` | PASS | 三场景 present/valid/passed，`all_required_passed=true` |

## 场景与数据检查

- [x] 资源场景包含 `baseline`、`isolation-only`、`aort-r`。
- [x] 资源场景包含 Planner、Coder-A、Coder-B、Tester、Reviewer、Fault-Agent。
- [x] 故障轮换覆盖 memory、pids、CPU 和受控 workspace rm-rf。
- [x] `aort-r` 的正常 Agent 完成率均值为 1，lowerdir hash unchanged 均值为 1，跨 Agent 污染均值为 0。
- [x] 上下文场景包含 `full-copy`、`shared-ipc`、`aort-r` 和 0/25/50/75% 四档共享比例。
- [x] full-copy 四档均传输 24576 bytes；aort-r 四档为 24576/18496/12352/6208 bytes。
- [x] aort-r 的 derived saved bytes 为 0/6080/12224/18368；50% 比例每轮 Prefix Affinity 命中 5 次。
- [x] 结果包含 raw JSON、`summary.json`、稳定列 `comparison.csv` 和 `report.md`。
- [x] 统计包含 count、mean、stddev、min、max、P50、P95、success rate，并区分 measured/derived/unsupported。
- [x] 失败运行不会被聚合器静默删除；`review-final` 对缺失或失败场景返回非零。

## 完整性与安全检查

- [x] 所有整改 JSON 均可由 `jq` 解析。
- [x] 两个 comparison CSV 的每行列数与表头一致。
- [x] summary、index、report 和 defense 文档引用的仓库内路径存在。
- [x] 未发现 `passed=true` 与非空 `failed_steps`/`failed_scenarios` 的矛盾。
- [x] 改动和整改证据中未发现 API Key、密码、私密 IPv4 字面量；环境变量名和脱敏状态字段属于预期文本。
- [x] DeepSeek Key 仅从环境读取；当前生成证据记录 `api_key_present=false`、`api_key_redacted=true`。
- [x] 危险清理路径有临时根、绝对/相对路径和逃逸校验；单元测试覆盖拒绝根目录与父目录。
- [x] 验收结束后未发现 AORT-R worker/CLI/experiment 进程、AORT-R mount 或 `aort-review-*` 临时目录残留。
- [x] eBPF 保持 `degraded`，未包装为完整可用能力。
- [x] CVM、AVP、Gateway、memfd/mmap 和 cgroup/OverlayFS 的对外表述已按能力边界收敛。

## 工作树与提交范围

最终整改提交只包含 `docs/review_remediation/FINAL_*.md`、`CHANGELOG.md`、`experiments/results/review_remediation/` 和 `experiments/results/review_final/`。以下用户已有修改/未跟踪内容明确保留且不暂存：

- `dashboard/src/api/client.ts`
- `dashboard/src/pages/Overview.vue`
- `dashboard/src/stores/runtime.ts`
- `dashboard/src/styles.css`
- `deliverables/`
- `docs/superpowers/plans/2026-07-06-runtime-real-os-evidence-plan.md`
- `experiments/results/audit_all/`
- `改进.md`

因此最终提交后的 `git status` 仍会是 dirty；这是保留用户工作而非验收遗漏。
