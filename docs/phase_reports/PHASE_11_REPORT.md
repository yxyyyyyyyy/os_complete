# PHASE 11 REPORT

## 1. 阶段目标

强化 E1/E2/E3 实验输出，让实验结果能支撑调度、故障隔离、上下文复用与 IPC 性能收益。

## 2. 本阶段实际完成内容

- E3 增加 `ipc_avoided_copy_bytes`。
- E3 实验通过 IPC Blackboard 发布 tester delta page，并由 Fixer poll + mount。
- 重新生成 `experiments/results/e3-context.json` 与 CSV。
- Dashboard Experiments 页展示 IPC avoided copy 指标。

## 3. 修改文件清单

| 文件 | 类型 | 说明 |
|---|---|---|
| internal/experiment/experiment.go | 修改 | E3 加入 IPC 指标 |
| internal/experiment/experiment_test.go | 修改 | 验证 IPC avoided bytes 为正 |
| experiments/results/e3-context.json | 修改 | 更新实验结果 |
| experiments/results/e3-context.csv | 修改 | 更新 CSV 字段 |
| dashboard/src/pages/Experiments.vue | 修改 | 展示 E3 IPC 指标 |

## 4. 核心实现说明

`RunE3ContextSharing` 继续创建共享 system/project/task pages，并为每个 Agent 写 delta page。tester delta 通过 Blackboard 发布为 page reference，Fixer poll 后 mount page。实验同时输出 full-copy token、unique page token、saved token/bytes、IPC avoided copy bytes。

## 5. 新增或修改的 API

本阶段无新增 API，复用 `/api/experiments/results`。

## 6. 验证命令

```bash
GOCACHE=/Users/yxy/Documents/比赛/操作系统/.cache/go-build go test ./internal/experiment
GOCACHE=/Users/yxy/Documents/比赛/操作系统/.cache/go-build go run ./cmd/aort-experiment --name all --runs 5 --out experiments/results
```

## 7. 验证结果

- `internal/experiment` 测试通过。
- 实验结果已重新生成。

## 8. 当前风险和遗留问题

- E1 当前仍为 degraded-simulation 策略对照，可进一步接入真实 demo scheduler decision replay。

## 9. 下一阶段建议

继续强化 Dashboard 证据页和 openEuler 部署材料。
