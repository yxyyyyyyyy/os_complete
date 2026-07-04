# AORT-R Manual Test Guide

## V1 Mock Demo

```bash
GOCACHE=/Users/yxy/Documents/比赛/操作系统/.cache/go-build go test ./...
GOCACHE=/Users/yxy/Documents/比赛/操作系统/.cache/go-build go run ./cmd/aortd --config configs/dev.yaml
curl -s http://127.0.0.1:8080/api/health
curl -s -X POST http://127.0.0.1:8080/api/demo/run
curl -N --max-time 2 http://127.0.0.1:8080/api/events
```

Expected:

- Health returns `{"mode":"mock","status":"ok"}`.
- Demo returns a `task_id`.
- SSE output contains `task.completed`.

## V1 Dashboard

```bash
cd dashboard
npm install
npm run test
npm run build
npm run dev
```

Expected:

- Overview shows task count, event count, SSE state, and DAG nodes.
- AVP page lists real Agent IDs, state, PID, cgroup path/mode, memory, PIDs, and capsule controls.
- Context page lists CVM pages, ref counts, saved bytes, and saved tokens.
- Timeline shows runtime events.
- Experiments page shows E1 scheduler bars, E2 fault isolation table, and E3 context sharing metrics.

## Stage 1 Real Worker Demo

```bash
GOCACHE=/Users/yxy/Documents/比赛/操作系统/.cache/go-build go test ./...
GOCACHE=/Users/yxy/Documents/比赛/操作系统/.cache/go-build go run ./cmd/aortd --config configs/dev.yaml
curl -s -X POST http://127.0.0.1:8080/api/demo/run
sleep 3
curl -s http://127.0.0.1:8080/api/agents
curl -s http://127.0.0.1:8080/api/syscalls
curl -s http://127.0.0.1:8080/api/scheduler/decisions
curl -s http://127.0.0.1:8080/api/context/stats
curl -N --max-time 2 http://127.0.0.1:8080/api/events
```

Expected:

- `/api/demo/run` starts Planner, Coder, and Tester worker processes.
- `/api/agents` returns non-zero `pid` values.
- `/api/syscalls` contains `context.materialize`, `tool.exec`, `context.write_delta`, and `agent.report`.
- `/api/scheduler/decisions` contains `token-cfs-prefix-affinity` decisions.
- `/api/context/stats` reports positive `saved_bytes` and `saved_tokens`.
- SSE contains `agent.registered`, `scheduler.selected`, `syscall.started`, `syscall.finished`, and `agent.report`.

Heartbeat lost check:

```bash
curl -s http://127.0.0.1:8080/api/agents
kill -INT <one_worker_pid>
sleep 7
curl -s http://127.0.0.1:8080/api/agents
curl -N --max-time 2 http://127.0.0.1:8080/api/events
```

Expected:

- The killed worker changes to `FAILED`.
- SSE contains `agent.heartbeat_lost`.

Capsule check:

```bash
curl -s http://127.0.0.1:8080/api/agents
curl -s -X POST http://127.0.0.1:8080/api/agents/<agent_id>/freeze
curl -s -X POST http://127.0.0.1:8080/api/agents/<agent_id>/unfreeze
curl -s -X POST http://127.0.0.1:8080/api/agents/<agent_id>/kill
```

Expected on openEuler/Linux with cgroup v2:

- `capsule_mode` is `real`.
- `cgroup_path` is under `/sys/fs/cgroup/aort.slice/`.
- `memory_current` and `pids_current` are populated.
- freeze/unfreeze/kill return `{"status":"ok"}`.

Expected on macOS or non-cgroup environments:

- `capsule_mode` is `degraded`.
- freeze/unfreeze return a structured error explaining why cgroup v2 is unavailable.
- Runtime remains usable and does not panic.

## Syscall Gateway Check

```bash
curl -s http://127.0.0.1:8080/api/syscalls
curl -N --max-time 2 http://127.0.0.1:8080/api/events
```

Expected:

- `tool.exec` records include `duration_ms`, `status`, `input_size`, and `output_size`.
- Timeline contains paired `syscall.started` and `syscall.finished` events for each worker syscall.

## Fault Injection Check

```bash
curl -s -X POST http://127.0.0.1:8080/api/demo/fault/tool-timeout
curl -s http://127.0.0.1:8080/api/faults
curl -s http://127.0.0.1:8080/api/syscalls
```

Expected:

- Fault response has `type=TOOL_TIMEOUT` and `status=RECOVERED`.
- `/api/syscalls` contains a `tool.exec` record with `status=TIMEOUT`.
- SSE contains `supervisor.detected`.

## Experiment Check

```bash
GOCACHE=/Users/yxy/Documents/比赛/操作系统/.cache/go-build go run ./cmd/aort-experiment --name all --runs 5 --out experiments/results
curl -s http://127.0.0.1:8080/api/experiments/results
ls experiments/results
```

Expected:

- `experiments/results/e1-scheduler.json` compares FIFO, token-CFS, and token-CFS-prefix-affinity.
- `experiments/results/e2-fault.json` compares no-capsule and per-agent-capsule modes.
- `experiments/results/e3-context.json` reports positive `saved_tokens` and `saved_bytes`.

## Later Iterations

- Remaining V2/V3 extensions: IPC blackboard, overlayfs rollback, richer Supervisor retry/spawn flows, eBPF timeline, checkpoint recovery, PSI, and systemd deployment.
