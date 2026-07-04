# PHASE_19_REAL_MULTI_AGENT_DEMO

mode=real-runtime

本阶段记录 `Software Engineering Multi-Agent Runtime Demo` 与 E5 端到端 benchmark。
Demo API 已接入现有 Runtime 机制：scheduler、CVM、syscall gateway、tool.exec、
page-ref IPC、supervisor fault record 和 checkpoint。

## Demo API

```bash
curl -s -X POST http://127.0.0.1:8080/api/demo/software-real/run \
  -H "Content-Type: application/json" \
  -d '{"requirement":"实现一个带测试的字符串工具函数"}'

curl -s http://127.0.0.1:8080/api/demo/software-real/status
curl -s http://127.0.0.1:8080/api/demo/software-real/result
```

## 必经 Runtime 机制

| 机制 | 当前证据 |
| --- | --- |
| agent.spawn | `/api/syscalls`, `agent.spawn` |
| context.materialize | `/api/syscalls`, `/api/context/stats` |
| context.write_delta | `/api/syscalls`, `/api/context/pages` |
| tool.exec | `/api/syscalls`, E5 `tool_exec=3` |
| ipc.publish / ipc.poll | `/api/syscalls`, `/api/ipc/metrics` |
| agent.report | `/api/syscalls` |
| scheduler decision | `/api/scheduler/decisions` |
| checkpoint save | `/api/checkpoints` 包含 `software-real` |
| fault handling | `/api/faults`, `TOOL_TIMEOUT` recovered |

## E5 运行命令

```bash
mkdir -p .cache/go-build
GOCACHE="$PWD/.cache/go-build" go run ./cmd/aort-experiment --name e5-end-to-end --runs 5 --out experiments/results
```

总实验命令：

```bash
GOCACHE="$PWD/.cache/go-build" go run ./cmd/aort-experiment --name all --runs 5 --out experiments/results
```

## E5 证据文件

```text
experiments/results/e5-end-to-end.json
experiments/results/e5-end-to-end.csv
```

## E5 当前结果

```json
{
  "experiment": "E5_end_to_end",
  "demo": "software-real",
  "evidence_mode": "real-runtime",
  "wall_time_ms": 12,
  "agents": 6,
  "syscalls": 11,
  "tool_exec": 3,
  "ipc_messages": 1,
  "context_saved_tokens": 175,
  "fault_recovered": true,
  "final_success": true
}
```

## 当前边界

- `real-runtime` 表示真实经过 AORT-R Runtime 内部机制。
- 当前 E5 JSON/CSV 已在远程 openEuler 24.03 LTS 上通过
  `go run ./cmd/aort-experiment --name all --runs 5 --out experiments/results`
  重新生成。
- 本报告不代表 real cgroup v2 已通过；openEuler cgroup v2 状态以
  `PHASE_16_OPEN_EULER_REAL_CGROUP_REPORT.md` 为准。
- 当前 E5 使用本地 shell tool.exec 和 Runtime supervisor fault record；后续可接入
  openEuler real worker/cgroup 证据生成更强版本。

## 验证

```bash
GOCACHE="$PWD/.cache/go-build" go test ./internal/api ./internal/experiment
```
