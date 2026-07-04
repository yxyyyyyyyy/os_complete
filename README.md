# AORT-R Agent Runtime

AORT-R is a prototype OS-level runtime for multi-agent execution on
openEuler/Linux.

It models each Agent as an AVP, starts real worker processes, routes Agent
actions through a Unix Domain Socket syscall gateway, tracks shared context
pages through CVM, supports page-reference IPC, records scheduler decisions,
saves lightweight checkpoints, and exposes runtime evidence through REST/SSE
and a bilingual Vue dashboard.

Core claim:

> AORT-R treats Agent as an OS-level execution unit: process/cgroup-backed AVP,
> CVM page table, syscall gateway for tools/LLM/IPC/control, token-CFS
> scheduling, fault supervision, checkpoint evidence, and replayable runtime
> trace.

## Quick Start

```bash
mkdir -p .cache/go-build
GOCACHE="$PWD/.cache/go-build" go test ./...
GOCACHE="$PWD/.cache/go-build" go run ./cmd/aortd --config configs/dev.yaml
```

In another terminal:

```bash
curl -s http://127.0.0.1:8080/api/health
curl -s -X POST http://127.0.0.1:8080/api/demo/run
sleep 3
curl -s http://127.0.0.1:8080/api/agents
curl -s http://127.0.0.1:8080/api/syscalls
curl -s http://127.0.0.1:8080/api/ipc/metrics
curl -s http://127.0.0.1:8080/api/kernel/status
curl -s http://127.0.0.1:8080/api/kernel/events
curl -s http://127.0.0.1:8080/api/pressure/status
curl -s http://127.0.0.1:8080/api/checkpoints
curl -s http://127.0.0.1:8080/api/recovery/status
curl -s -X POST http://127.0.0.1:8080/api/demo/fault/rmrf
curl -s http://127.0.0.1:8080/api/scheduler/decisions
curl -s http://127.0.0.1:8080/api/context/stats
```

Start the dashboard:

```bash
cd dashboard
npm install
npm run dev
```

## Optional DeepSeek Env

Real API credentials must stay outside Git.

To prepare an openEuler host for a future DeepSeek provider, copy the example
file and fill the key locally:

```bash
cp .env.example .env.local
# edit .env.local and set DEEPSEEK_API_KEY
```

`.env.local` is ignored by Git.

The current checked-in Runtime still uses the mock LLM provider. Adding these
environment variables does not make AORT-R call DeepSeek until a DeepSeek
provider is implemented.

## Evidence Modes

- `real`: Evidence comes from the running Runtime or OS surface directly.
  Examples include real worker PIDs, UDS registration, syscall records, CVM
  stats, scheduler decisions, Linux cgroup v2 counters, and PSI files.
- `degraded`: The Runtime is still running real code, but the local OS lacks a
  required capability or permission. Examples include macOS/non-root cgroup
  capsule fallback, `degraded-proxy` kernel exec evidence instead of eBPF,
  unavailable PSI files, and degraded-copy workspace rollback instead of
  overlayfs.
- `simulation/mock`: The path is intentionally synthetic or mocked for
  repeatable local demos, such as the mock LLM provider or experiment modes that
  model unavailable OS capabilities. These paths must be labeled as
  simulation/mock and should not be presented as real openEuler evidence.

## Implemented Mechanisms

- Real worker processes launched by `aortd`, with UDS registration and heartbeat.
- Per-Agent cgroup capsule manager with real Linux cgroup v2 support and degraded mode on macOS/non-root environments.
- CVM page store with sha256 page ids, per-agent page tables, context materialization, and saved token/byte metrics.
- Agent syscall gateway for `context.materialize`, `context.write_delta`,
  `llm.call`, `tool.exec`, `ipc.publish`, `ipc.poll`, `agent.spawn`, and
  `agent.report`, with audit records and SSE timeline events.
- Kernel observer lane for `kernel.exec` evidence. Current checked-in
  implementation uses explicit `degraded-proxy` mode through syscall-gateway
  exec observations unless a future openEuler eBPF attachment is enabled.
- Page-reference IPC Blackboard with avoided-copy byte metrics and per-subscriber polling.
- FIFO, token-CFS, and token-CFS-prefix-affinity scheduler policies with DecisionLog API.
- PSI pressure monitor with `/api/pressure/status`, `pressure.sampled`, and scheduler pressure-throttle evidence in degraded or Linux PSI mode.
- Supervisor fault record path with a runnable `tool.exec` timeout injection.
- Workspace isolation fault demo with degraded-copy rollback evidence for `rm -rf` style failures.
- Lightweight checkpoint store for AVP state, CVM page table references, scheduler vruntime, and trace offset evidence.
- Startup checkpoint recovery report at `/api/recovery/status`, with `checkpoint.recovered` and `runtime.recovered` timeline evidence.
- LLM Router interface with mock provider, fallback routing, and llama.cpp timing/cache usage parser.
- E1/E2/E3 experiment runner producing JSON and CSV under `experiments/results/`.
- Vue dashboard pages for Overview with pressure/recovery evidence, AVP/Capsule, Context, Timeline with kernel lane evidence, and Experiments.

## Experiments

```bash
mkdir -p .cache/go-build
GOCACHE="$PWD/.cache/go-build" go run ./cmd/aort-experiment --name all --runs 5 --out experiments/results
curl -s http://127.0.0.1:8080/api/experiments/results
```

Outputs:

- `experiments/results/e1-scheduler.json`
- `experiments/results/e2-fault.json`
- `experiments/results/e3-context.json`
- Matching CSV files for each experiment.

## openEuler Smoke Test

Run these commands on openEuler 24.03 LTS with Linux root permission and cgroup v2 mounted:

```bash
bash scripts/check_openeuler_env.sh
bash scripts/smoke_openeuler.sh
```

Smoke outputs are written to `experiments/results/openeuler_smoke/`.

## openEuler Notes

Use `configs/openeuler-dev.yaml` when running in an openEuler VM with root
permission and cgroup v2 mounted.

On non-Linux machines the capsule layer intentionally returns
`capsule_mode=degraded`, while the runtime, syscall gateway, scheduler, CVM,
IPC, checkpoint, fault injection, and dashboard remain usable.

See [docs/deployment_openeuler.md](docs/deployment_openeuler.md) for deployment checks, systemd service setup, and scripts.

## Known Limits

- Real overlayfs mount/commit and true eBPF attachment are planned enhancement
  targets. Degraded-copy workspace rollback, lightweight checkpoint startup
  recovery, PSI/degraded pressure monitoring, and honest degraded-proxy kernel
  exec evidence are implemented and test-covered.
- Current checked-in LLM path uses the mock provider; DeepSeek relay and llama.cpp local providers should be configured outside Git with credentials/model paths.
- Experiments are marked as real, degraded-real, or degraded-simulation according to the available local OS/runtime evidence.
