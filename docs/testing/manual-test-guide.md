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
- Context page also lists IPC topic/page references and avoided-copy bytes.
- Timeline shows runtime, syscall, IPC, LLM, spawn, checkpoint, and supervisor events.
- Experiments page shows E1 scheduler bars, E2 fault isolation table, and E3 context plus IPC metrics.

## Stage 1 Real Worker Demo

```bash
GOCACHE=/Users/yxy/Documents/比赛/操作系统/.cache/go-build go test ./...
GOCACHE=/Users/yxy/Documents/比赛/操作系统/.cache/go-build go run ./cmd/aortd --config configs/dev.yaml
curl -s -X POST http://127.0.0.1:8080/api/demo/run
sleep 3
curl -s http://127.0.0.1:8080/api/agents
curl -s http://127.0.0.1:8080/api/syscalls
curl -s http://127.0.0.1:8080/api/ipc/metrics
curl -s http://127.0.0.1:8080/api/ipc/topics
curl -s http://127.0.0.1:8080/api/checkpoints
curl -s http://127.0.0.1:8080/api/recovery/status
curl -s http://127.0.0.1:8080/api/scheduler/decisions
curl -s http://127.0.0.1:8080/api/context/stats
curl -N --max-time 2 http://127.0.0.1:8080/api/events
```

Expected:

- `/api/demo/run` starts Planner, Coder, and Tester worker processes.
- `/api/agents` returns non-zero `pid` values.
- `/api/syscalls` contains `context.materialize`, `llm.call`, `ipc.publish`, `ipc.poll`, `agent.spawn`, `tool.exec`, `context.write_delta`, and `agent.report`.
- `/api/ipc/metrics` reports positive `avoided_copy_bytes`.
- `/api/checkpoints` contains a `runtime-state` snapshot.
- `/api/recovery/status` returns `checkpoint-light` recovery metadata.
- `/api/scheduler/decisions` contains `token-cfs-prefix-affinity` decisions.
- `/api/context/stats` reports positive `saved_bytes` and `saved_tokens`.
- SSE contains `agent.registered`, `scheduler.selected`, `syscall.started`, `syscall.finished`, `llm.called`, `ipc.published`, `ipc.polled`, `agent.spawned`, `checkpoint.created`, and `agent.report`.

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

## IPC, LLM, Spawn, and Checkpoint Check

```bash
curl -s -X POST http://127.0.0.1:8080/api/demo/run
curl -s http://127.0.0.1:8080/api/syscalls
curl -s http://127.0.0.1:8080/api/ipc/metrics
curl -s http://127.0.0.1:8080/api/ipc/topics
curl -s http://127.0.0.1:8080/api/checkpoints
curl -s http://127.0.0.1:8080/api/recovery/status
```

Expected:

- `/api/syscalls` includes `llm.call`, `ipc.publish`, `ipc.poll`, and `agent.spawn`.
- `/api/ipc/topics` contains `review.feedback` with page IDs, not copied content.
- `/api/ipc/metrics` has positive `avoided_copy_bytes`.
- `/api/checkpoints` contains AVP and page table state for the demo task.
- `/api/recovery/status` exposes startup recovery mode, recovered task count, ready agents, completed agents, page table references, and scheduler vruntime.

## Checkpoint Startup Recovery Check

Local two-process simulation:

```bash
rm -rf .aort-dev/checkpoints
GOCACHE=/Users/yxy/Documents/比赛/操作系统/.cache/go-build go run ./cmd/aortd --config configs/dev.yaml
curl -s -X POST http://127.0.0.1:8080/api/demo/run
curl -s http://127.0.0.1:8080/api/checkpoints
# Stop aortd with Ctrl-C, then start it again with the same config/data_dir.
GOCACHE=/Users/yxy/Documents/比赛/操作系统/.cache/go-build go run ./cmd/aortd --config configs/dev.yaml
curl -s http://127.0.0.1:8080/api/recovery/status
curl -s http://127.0.0.1:8080/api/tasks
curl -N --max-time 2 http://127.0.0.1:8080/api/events
```

Expected:

- `/api/recovery/status` has `mode=checkpoint-light`, `task_count>=1`, and `degraded=true`.
- `/api/tasks` contains the task restored from checkpoint.
- SSE contains `checkpoint.recovered` and `runtime.recovered`.

openEuler systemd daemonkill demo:

```bash
sudo systemctl restart aortd
curl -s -X POST http://127.0.0.1:8080/api/demo/run
scripts/demo-daemonkill.sh
```

Expected:

- `systemctl status aortd` shows the daemon was restarted.
- The script prints `/api/recovery/status` and `/api/tasks` evidence after restart.
- Dashboard Overview shows the Checkpoint Recovery panel, and Timeline shows recovery events.

## Fault Injection Check

```bash
curl -s -X POST http://127.0.0.1:8080/api/demo/fault/tool-timeout
curl -s -X POST http://127.0.0.1:8080/api/demo/fault/rmrf
curl -s http://127.0.0.1:8080/api/faults
curl -s http://127.0.0.1:8080/api/syscalls
```

Expected:

- Fault response has `type=TOOL_TIMEOUT` and `status=RECOVERED`.
- `/api/syscalls` contains a `tool.exec` record with `status=TIMEOUT`.
- The rmrf response has `type=WORKSPACE_ROLLBACK`, `workspace_mode=degraded-copy`, `rollback_success=true`, and `base_intact=true`.
- SSE contains `supervisor.detected`, `workspace.created`, `workspace.rmrf`, and `workspace.rollback`.

## Experiment Check

```bash
GOCACHE=/Users/yxy/Documents/比赛/操作系统/.cache/go-build go run ./cmd/aort-experiment --name all --runs 5 --out experiments/results
curl -s http://127.0.0.1:8080/api/experiments/results
ls experiments/results
```

Expected:

- `experiments/results/e1-scheduler.json` compares FIFO, token-CFS, and token-CFS-prefix-affinity.
- `experiments/results/e2-fault.json` compares no-capsule and per-agent-capsule modes.
- `experiments/results/e3-context.json` reports positive `saved_tokens`, `saved_bytes`, and `ipc_avoided_copy_bytes`.

## Later Iterations

- Remaining V2/V3 extensions: real overlayfs mount/commit, richer Supervisor retry policies, eBPF timeline, durable CVM page-content checkpointing, PSI, and openKylin/OpenHarmony smoke tests.
