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

To enable the real DeepSeek provider, copy the example file and fill the key
locally:

```bash
cp .env.example .env.local
# edit .env.local and set DEEPSEEK_API_KEY
```

`.env.local` is ignored by Git.

Set `AORT_LLM_PROVIDER=deepseek` to request DeepSeek and
`AORT_LLM_FALLBACK_PROVIDER=mock` to keep deterministic fallback. API keys are
read only from environment variables and are never written to source,
documentation, or experiment results. Failed DeepSeek calls are explicitly
marked as `fallback=true`, `fallback_reason=no_api_key` or `api_error`, and
`evidence_mode=mock`.

## Evidence Modes

- `real`: Evidence comes from the running Runtime or OS surface directly.
  Examples include real worker PIDs, UDS registration, syscall records, CVM
  stats, scheduler decisions, Linux cgroup v2 counters, and PSI files.
- `real-cgroup-v2`: Evidence comes from openEuler 24.03 LTS with unified
  cgroup v2 mounted as `cgroup2fs`, including `memory.current`,
  `pids.current`, `cpu.stat`, freeze/unfreeze/kill, and cgroup limit
  enforcement.
- `real-runtime`: Evidence comes from live AORT-R runtime mechanisms such as
  scheduler decisions, syscall records, CVM materialization, IPC page refs,
  checkpoints, and tool execution.
- `real-api`: Evidence comes from a successful external model-provider API
  call, currently the DeepSeek OpenAI-compatible provider.
- `degraded`: The Runtime is still running real code, but the local OS lacks a
  required capability or permission. Examples include macOS/non-root cgroup
  capsule fallback, `degraded-proxy` kernel exec evidence instead of eBPF,
  unavailable PSI files, and degraded-copy workspace rollback instead of
  overlayfs.
- `mock`: The path is intentionally mocked for repeatable local demos, such as
  the default LLM provider. Mock paths are useful for deterministic tests but
  must not be presented as real model-provider evidence.
- `simulation`: The path is intentionally synthetic for unavailable OS
  capabilities or controlled experiment models. Simulation outputs must be
  labeled as simulation/degraded-simulation.
- `planned`: The design is documented but not implemented in this build, such
  as true eBPF attachment or overlayfs mount/commit isolation.
- `simulation/mock`: Legacy reports may use this combined label for
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
- LLM DeepSeek provider with environment-only API key loading and mock fallback.
- E1/E2/E3 experiment runner producing JSON and CSV under `experiments/results/`,
  including real-runtime E1 scheduler and E2 fault benchmarks.
- Vue dashboard pages for Overview with pressure/recovery evidence, AVP/Capsule, Context, Timeline with kernel lane evidence, and Experiments.

## Experiments

```bash
mkdir -p .cache/go-build
GOCACHE="$PWD/.cache/go-build" go run ./cmd/aort-experiment --name all --runs 5 --out experiments/results
GOCACHE="$PWD/.cache/go-build" go run ./cmd/aort-experiment --name e1-real-scheduler --runs 5 --out experiments/results
GOCACHE="$PWD/.cache/go-build" go run ./cmd/aort-experiment --name e2-real-fault --runs 5 --out experiments/results
curl -s http://127.0.0.1:8080/api/experiments/results
```

Outputs:

- `experiments/results/e1-scheduler.json`
- `experiments/results/e2-fault.json`
- `experiments/results/e3-context.json`
- `experiments/results/e1-real-scheduler.json`
- `experiments/results/e2-real-fault.json`
- Matching CSV files for each experiment.

`e1-scheduler` and `e2-fault` are retained as legacy
simulation/synthetic baselines. Current scheduler/fault evidence should use
`e1-real-scheduler` and `e2-real-fault`, which call the real-runtime benchmark
functions and write `evidence_mode=real-runtime`.

## openEuler Smoke Test

Run these commands on openEuler 24.03 LTS with Linux root permission and cgroup v2 mounted:

```bash
bash scripts/check_openeuler_env.sh
bash scripts/smoke_openeuler.sh
bash scripts/smoke_cgroupv2_multi_agent.sh
bash scripts/smoke_cgroupv2_limits.sh
```

Smoke outputs are written to:

- `experiments/results/openeuler_smoke/`
- `experiments/results/openeuler_cgroupv2_multi/`
- `experiments/results/openeuler_cgroupv2_limits/`

## Current openEuler Evidence Status

Latest status:

- openEuler 24.03 LTS real cgroup v2 smoke has passed.
- `stat -fc %T /sys/fs/cgroup = cgroup2fs`.
- `capsule_mode=real`.
- `real_cgroup_v2=true`.
- `memory.current`, `pids.current`, `cpu.stat`, freeze, unfreeze, and kill
  are recorded from real cgroup v2 files and APIs.
- memory, pids, and CPU limits all have real enforcement evidence.

Primary current evidence:

- `experiments/results/openeuler_smoke/`
- `experiments/results/openeuler_cgroupv2_multi/`
- `experiments/results/openeuler_cgroupv2_limits/`
- `experiments/results/openeuler_smoke/capsule_real.json`
- `experiments/results/openeuler_smoke/agent_summary.json`
- `experiments/results/openeuler_smoke/aort-r-openeuler-7d939c2-cgroupv2-real-evidence.tgz`
- `docs/phase_reports/PHASE_17_REAL_CGROUP_V2_REPORT.md`
- `docs/phase_reports/PHASE_16_OPEN_EULER_REAL_CGROUP_REPORT.md`
- `docs/phase_reports/PHASE_16_CGROUP_V2_REAL_REPORT.md`

Historical degraded evidence under `experiments/results/openeuler_smoke/` and
older phase reports is retained only as a before/after record. It does not
represent the current openEuler cgroup v2 status.

To re-verify from scratch on openEuler:

```bash
stat -fc %T /sys/fs/cgroup
# expected: cgroup2fs

bash scripts/check_openeuler_env.sh
bash scripts/smoke_openeuler.sh
bash scripts/smoke_cgroupv2_multi_agent.sh
bash scripts/smoke_cgroupv2_limits.sh
```

Do not present mock, degraded, simulation, or planned modules as real evidence.

## openEuler Notes

Use `configs/openeuler-dev.yaml` when running in an openEuler VM with root
permission and cgroup v2 mounted.

On non-Linux machines the capsule layer intentionally uses a degraded capsule
fallback, while the runtime, syscall gateway, scheduler, CVM, IPC, checkpoint,
fault injection, and dashboard remain usable. This fallback note is not the
current openEuler cgroup v2 evidence status.

See [docs/deployment_openeuler.md](docs/deployment_openeuler.md) for deployment checks, systemd service setup, and scripts.

## Known Limits

- Real overlayfs mount/commit and true eBPF attachment are planned enhancement
  targets. Degraded-copy workspace rollback, lightweight checkpoint startup
  recovery, PSI/degraded pressure monitoring, and honest degraded-proxy kernel
  exec evidence are implemented and test-covered.
- DeepSeek provider is implemented with environment-only credentials and mock
  fallback; local llama.cpp remains a planned provider path.
- Experiments are marked as real-cgroup-v2, real-runtime, real-api, mock,
  degraded, or simulation according to the available evidence.
