# PHASE 13 REPORT

## 1. 阶段目标

让项目能在 openEuler/Linux 上讲得清楚、跑得起来、交得干净。

## 2. 本阶段实际完成内容

- 新增 openEuler 部署文档。
- 新增环境检查、demo 启动、实验运行脚本。
- 更新 README、manual test guide、competition checklist。
- `.gitignore` 已覆盖 node_modules、dist、.cache、.DS_Store、日志和本地运行数据。

## 3. 修改文件清单

| 文件 | 类型 | 说明 |
|---|---|---|
| docs/deployment_openeuler.md | 新增 | openEuler 部署与证据指南 |
| scripts/check_env.sh | 新增 | 环境检查 |
| scripts/run_demo.sh | 新增 | 测试并启动 demo runtime |
| scripts/run_experiments.sh | 新增 | 运行 E1/E2/E3 实验 |
| README.md | 修改 | 更新创新点、运行方式、限制 |
| docs/testing/manual-test-guide.md | 修改 | 新增 IPC/LLM/spawn/checkpoint 验收 |
| docs/delivery/competition-checklist.md | 修改 | 更新完成度 |

## 4. 核心实现说明

部署文档将环境要求、运行命令、Dashboard、实验输出和证据映射放在同一文件中。脚本只做可复现命令编排，不写入敏感信息。

## 5. 新增或修改的 API

本阶段无新增 API。

## 6. 验证命令

```bash
scripts/check_env.sh
scripts/run_experiments.sh 5
```

## 7. 验证结果

- `scripts/run_experiments.sh 5` 已由等价命令验证，实验结果已输出到 `experiments/results`。
- `scripts/check_env.sh` 会按当前 OS 能力报告 real/degraded 条件。

## 8. 当前风险和遗留问题

- openEuler root 环境需要在 VM 上最终复测 cgroup v2 freeze/unfreeze。
- systemd service 尚未加入。

## 9. 下一阶段建议

全量运行 Go/前端验证，提交并推送到 GitHub。
