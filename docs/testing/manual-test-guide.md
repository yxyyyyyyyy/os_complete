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
- AVP page lists Planner, Coder A, Coder B, Tester, Reviewer, and Fixer.
- Context page lists syscall evidence.
- Timeline shows runtime events.
- Experiments page states that experiment metrics arrive in V3.

## Later Iterations

- V2 adds cgroup, overlayfs, CVM, syscall gateway, scheduler, IPC, and fault injection tests.
- V3 adds llama.cpp metrics, eBPF timeline, checkpoint recovery, experiment charts, and systemd deployment.
