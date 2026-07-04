# AORT-R Agent Runtime

AORT-R is a prototype OS-level runtime for multi-agent execution. It models each Agent as an AVP, starts real worker processes, routes Agent actions through a Unix Domain Socket syscall gateway, tracks shared context pages through CVM, records scheduler decisions, and exposes runtime evidence through REST/SSE and a Vue dashboard.

## Quick Start

```bash
GOCACHE=/Users/yxy/Documents/比赛/操作系统/.cache/go-build go test ./...
GOCACHE=/Users/yxy/Documents/比赛/操作系统/.cache/go-build go run ./cmd/aortd --config configs/dev.yaml
```

In another terminal:

```bash
curl -s http://127.0.0.1:8080/api/health
curl -s -X POST http://127.0.0.1:8080/api/demo/run
sleep 3
curl -s http://127.0.0.1:8080/api/agents
curl -s http://127.0.0.1:8080/api/syscalls
curl -s http://127.0.0.1:8080/api/scheduler/decisions
curl -s http://127.0.0.1:8080/api/context/stats
```

Start the dashboard:

```bash
cd dashboard
npm install
npm run dev
```

## Implemented Mechanisms

- Real worker processes launched by `aortd`, with UDS registration and heartbeat.
- Per-Agent cgroup capsule manager with real Linux cgroup v2 support and degraded mode on macOS/non-root environments.
- CVM page store with sha256 page ids, per-agent page tables, context materialization, and saved token/byte metrics.
- Agent syscall gateway for `context.materialize`, `context.write_delta`, `tool.exec`, and `agent.report`, with audit records and SSE timeline events.
- FIFO, token-CFS, and token-CFS-prefix-affinity scheduler policies with DecisionLog API.
- Supervisor fault record path with a runnable `tool.exec` timeout injection.
- E1/E2/E3 experiment runner producing JSON and CSV under `experiments/results/`.
- Vue dashboard pages for Overview, AVP/Capsule, Context, Timeline, and Experiments.

## Experiments

```bash
GOCACHE=/Users/yxy/Documents/比赛/操作系统/.cache/go-build go run ./cmd/aort-experiment --name all --runs 5 --out experiments/results
curl -s http://127.0.0.1:8080/api/experiments/results
```

Outputs:

- `experiments/results/e1-scheduler.json`
- `experiments/results/e2-fault.json`
- `experiments/results/e3-context.json`
- Matching CSV files for each experiment.

## openEuler Notes

Use `configs/openeuler-dev.yaml` when running in an openEuler VM with root permission and cgroup v2 mounted. On non-Linux machines the capsule layer intentionally returns `capsule_mode=degraded`, while the runtime, syscall gateway, scheduler, CVM, fault injection, and dashboard remain usable.
