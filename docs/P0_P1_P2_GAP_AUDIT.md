# P0/P1/P2 Gap Audit

审计时间：2026-07-06

## 审计命令

| command | result | notes |
| --- | --- | --- |
| `go test ./...` | passed | 使用本仓库 `.cache/go-build` 在 macOS 上通过。 |
| `bash -n scripts/competition_verify.sh` | passed | 脚本语法有效。 |
| `bash scripts/competition_verify.sh` | passed | macOS host 会标记 overall `degraded`；Linux/root/openEuler 复跑保留 real cgroup/overlayfs 证据边界。 |
| `go run ./cmd/aortctl experiment e1 --policy resource-aware --runs 5 --out experiments/results/e1` | passed | resource-aware 路径可用，输出汇总 JSON/CSV/decision log。 |
| `go run ./cmd/aortctl experiment e2 --runs 5 --out experiments/results` | passed | 生成 E2 real fault JSON/CSV。 |
| `go run ./cmd/aortctl demo software-real --out experiments/results` | passed | 生成 `experiments/results/software_real_demo/result.json`。 |
| `go run ./cmd/aortctl demo fault workspace-rmrf --out experiments/results` | passed | 生成 `experiments/results/workspace_isolation_evidence.json`。 |
| `go run ./cmd/aortctl experiment e1 --policy all --runs 2 --out /tmp/aort-e1-all-audit` | passed | `all` policy 路径可用，包含四个 scheduler policy。 |

## 发现的缺口

| area | gap | files | status |
| --- | --- | --- | --- |
| P0 one-command verification | `competition_verify.sh` 可运行，并明确 degraded/archived 边界。 | `scripts/competition_verify.sh`, final evidence index | verified |
| P1 E1 resource-aware output | `e1_resource_aware.json` 原先是裸 per-policy 数组，不满足要求的汇总对象 schema。 | `internal/experiment/real_benchmark.go`, `internal/experiment/resource_aware_test.go` | fixed |
| P1 scheduler pressure tests | 已有综合 resource pressure 测试，但缺少 memory/pids/cpu throttle 独立惩罚测试文件。 | `internal/scheduler/scheduler_resource_aware_test.go` | fixed |
| P2 workspace isolation | `workspace-rmrf` demo 在 Linux/root/mount 成功时为 `real-overlayfs`；其他环境为 `degraded-copy`。 | `internal/workspace/manager.go`, `experiments/results/workspace_isolation_evidence.json` | verified |
| docs/CLI consistency | README 和设计文档中部分 `aortctl` 命令缺少 `--out`，且旧 kernel evidence vocabulary 需要统一。 | `README.md`, `docs/SCHEDULER_DESIGN.md`, `docs/WORKSPACE_ISOLATION_DESIGN.md` | fixed |

## Evidence 文件检查

| artifact | exists | key fields |
| --- | --- | --- |
| `experiments/results/final/FINAL_EVIDENCE_INDEX.json` | yes | `go_test`, `smoke`, `e1_scheduler`, `e2_fault_isolation`, `software_real_demo`, `workspace_isolation`, `evidence_mode_summary` |
| `experiments/results/final/FINAL_SUMMARY.md` | yes | human-readable final summary |
| `experiments/results/e1/e1_resource_aware.json` | yes | `experiment`, `runs`, `policies`, `metrics`, `improvement`, `evidence_mode` |
| `experiments/results/e1/e1_resource_aware.csv` | yes | per-policy metrics |
| `experiments/results/e1/e1_resource_aware_decisions.json` | yes | scheduler decision log with resource pressure fields |
| `experiments/results/e1/e1_resource_aware_summary.md` | yes | E1 summary |
| `experiments/results/workspace_isolation_evidence.json` | yes | `evidence_mode`, `fallback_reason`, `lowerdir_unchanged`, `rollback_success`, `safety_checks` |

## 当前完成度

- P0：一键复现实验脚本可执行；非 openEuler host 输出 degraded/archived 边界；final index/summary 生成。
- P1：resource-aware policy 已在 `Policies()`、`SetPolicy()`、E1 benchmark、CLI 和 decision log 中闭环；E1 JSON schema 已补为汇总对象；新增独立压力惩罚测试。
- P2：workspace manager 和 `workspace-rmrf` CLI/API demo 闭环；Linux/root
  mount 成功时为 `real-overlayfs`，macOS/non-root fallback 为
  `degraded-copy`。

## 风险

- 本地 macOS 无法现场证明 live overlayfs mount 或 live cgroup2fs；最终
  checked-in workspace evidence 来自 Linux/root 服务器复跑，必须与实际
  `workspace_isolation_evidence.json` 保持一致。
- eBPF 仍为 `planned`，不能宣传为已实现。
- LLM 默认 provider 仍为 `mock`；只有带环境变量的 DeepSeek smoke 才能标 `real-api`。
