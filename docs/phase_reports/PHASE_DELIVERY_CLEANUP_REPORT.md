# PHASE_DELIVERY_CLEANUP_REPORT

## 修复结果

| 检查项 | 结果 |
| --- | --- |
| `go.mod` 是否修复 | 已修复，为 `module aort-r`、空行、`go 1.22` 的合法格式。 |
| `go test ./...` 是否通过 | 已通过，使用 `GOCACHE="$PWD/.cache/go-build" go test ./...`。 |
| `bash -n scripts/*.sh` 是否通过 | 已通过。 |
| `docs/testing/manual-test-guide.md` 是否清除本机路径 | 已清除，未发现指定本机路径残留。 |
| README 是否重新排版 | 已重新排版，标题、段落、列表和代码块均保持正常 Markdown 结构。 |
| PHASE_15 状态 | 已由后续 PHASE_16 同步为 `mode=degraded-real`。 |

## 命令记录

```bash
gofmt ./...
```

原命令结果：

```text
stat ./...: no such file or directory
```

说明：`gofmt` 不接受 `./...` 包通配参数。已执行等价全量格式化：

```bash
rg --files -g '*.go' | xargs gofmt -w
```

其余验证命令：

```bash
mkdir -p .cache/go-build
GOCACHE="$PWD/.cache/go-build" go test ./...
bash -n scripts/*.sh
```

## 当前 openEuler 证据状态

当前已有一次 openEuler 24.03 LTS / Linux root degraded-real smoke 输出，但仍缺
unified cgroup v2 的 `capsule_mode=real` 满血证据。以下文件记录 degraded-real
证据，不能作为 cgroup v2 real 成功证据：

- `experiments/results/openeuler_smoke/env_check.txt`
- `experiments/results/openeuler_smoke/agents.json`
- `experiments/results/openeuler_smoke/agent_summary.json`
- `experiments/results/openeuler_smoke/syscalls.json`
- `experiments/results/openeuler_smoke/context_stats.json`
- `experiments/results/openeuler_smoke/scheduler_decisions.json`
- `experiments/results/openeuler_smoke/fault_tool_timeout.json`

openEuler 实机运行入口：

```bash
bash scripts/check_openeuler_env.sh
bash scripts/smoke_openeuler.sh
```

下一次 real cgroup v2 smoke 之前必须确认：

```bash
stat -fc %T /sys/fs/cgroup
```

输出为：

```text
cgroup2fs
```
