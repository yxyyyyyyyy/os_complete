# AORT-R Competition Checklist

## Minimum Runnable Evidence

| 状态 | 检查项 | 证据位置 |
| --- | --- | --- |
| [x] | `POST /api/demo/run` starts real worker processes. | `experiments/results/openeuler_smoke/demo_run.json` |
| [x] | `GET /api/agents` returns real worker PIDs. | `experiments/results/openeuler_smoke/agents.json` |
| [x] | Worker registration uses Unix Domain Socket. | `experiments/results/openeuler_smoke/agents.json` and `aortd.log` |
| [x] | Worker heartbeat is tracked by `aortd`. | `experiments/results/openeuler_smoke/agents.json` |
| [x] | Heartbeat timeout marks an Agent as `FAILED`. | `docs/testing/manual-test-guide.md` heartbeat lost check |
| [x] | Each Agent has a cgroup path; macOS/non-root returns degraded paths. | `experiments/results/openeuler_smoke/agent_summary.json` |
| [x] | `memory_current`, `pids_current`, and `cpu_stat` are exposed when cgroup v2 is available. | `experiments/results/openeuler_smoke/agent_summary.json` |
| [x] | `freeze`, `unfreeze`, and `kill` Agent APIs exist. | `experiments/results/openeuler_smoke/freeze.json`, `unfreeze.json`, `kill.json` |
| [x] | CVM page store and per-agent page table exist. | `experiments/results/openeuler_smoke/context_stats.json` |
| [x] | Shared context pages increase ref counts and saved byte/token metrics. | `experiments/results/openeuler_smoke/context_stats.json` |
| [x] | `context.materialize` is served through the syscall gateway. | `experiments/results/openeuler_smoke/syscalls.json` |
| [x] | `llm.call` is served through the syscall gateway with mock provider usage metrics. | `experiments/results/openeuler_smoke/syscalls.json` |
| [x] | Worker requests `tool.exec` through the syscall gateway. | `experiments/results/openeuler_smoke/syscalls.json` |
| [x] | `ipc.publish` and `ipc.poll` transfer CVM page IDs and track avoided copy bytes. | `experiments/results/openeuler_smoke/syscalls.json`, `experiments/results/e3-context.json` |
| [x] | `agent.spawn` records Reviewer-triggered Fixer creation through the syscall gateway. | `experiments/results/openeuler_smoke/syscalls.json` |
| [x] | Timeline sees `syscall.started` and `syscall.finished`. | `experiments/results/openeuler_smoke/aortd.log` |
| [x] | Timeline sees `kernel.observer_disabled` and `kernel.exec` in honest degraded-proxy mode. | `experiments/results/openeuler_smoke/aortd.log` |
| [x] | Timeline sees `ipc.published`, `ipc.polled`, `llm.called`, `agent.spawn.requested`, and `agent.spawned`. | `experiments/results/openeuler_smoke/aortd.log` |
| [x] | Scheduler supports FIFO, token-CFS, and token-CFS-prefix-affinity. | `experiments/results/e1-scheduler.json` |
| [x] | Pressure monitor exposes PSI/degraded status through `GET /api/pressure/status`. | `experiments/results/openeuler_smoke/env_check.txt` and `/api/pressure/status` manual check |
| [x] | Scheduler timeline events include pressure mode/throttle/avg10 evidence. | `experiments/results/openeuler_smoke/scheduler_decisions.json` |
| [x] | Dashboard Overview, AVP, Context, Timeline, and Experiments pages use real APIs. | `dashboard/src/` and `docs/testing/manual-test-guide.md` |
| [x] | Tool timeout fault demo exists at `POST /api/demo/fault/tool-timeout`. | `experiments/results/openeuler_smoke/fault_tool_timeout.json` |
| [x] | Workspace rmrf rollback demo exists at `POST /api/demo/fault/rmrf`. | `docs/testing/manual-test-guide.md` fault injection check |
| [x] | Lightweight checkpoint snapshots are exposed through `GET /api/checkpoints`. | `docs/testing/manual-test-guide.md` checkpoint check |
| [x] | Startup recovery report is exposed through `GET /api/recovery/status`. | `docs/testing/manual-test-guide.md` recovery check |
| [x] | Experiment JSON/CSV files exist under `experiments/results/`. | `experiments/results/` |

## High-Score Evidence Present

| 状态 | 检查项 | 证据位置 |
| --- | --- | --- |
| [x] | Scheduler DecisionLog API: `GET /api/scheduler/decisions`. | `experiments/results/openeuler_smoke/scheduler_decisions.json` |
| [x] | Scheduler policy API: `POST /api/scheduler/policy`. | `docs/testing/manual-test-guide.md` |
| [x] | PSI pressure-aware scheduling evidence: `pressure.sampled`, `scheduler.pressure_throttle`, and `pressure_*` scheduler fields. | `experiments/results/openeuler_smoke/scheduler_decisions.json` |
| [x] | E1 scheduler experiment. | `experiments/results/e1-scheduler.json` |
| [x] | E2 fault isolation experiment. | `experiments/results/e2-fault.json` |
| [x] | E3 context sharing and IPC avoided-copy experiment. | `experiments/results/e3-context.json` |
| [x] | Dashboard experiments visualization. | `dashboard/src/` |
| [x] | openEuler-oriented cgroup v2 design with graceful degraded mode. | `scripts/check_openeuler_env.sh`, `scripts/smoke_openeuler.sh` |
| [x] | IPC Blackboard API: `GET /api/ipc/metrics`, `GET /api/ipc/topics`. | `docs/testing/manual-test-guide.md` |
| [x] | LLM Router abstraction with mock provider and llama.cpp timing parser. | `internal/llm/router.go` |
| [x] | Kernel Observer API: `GET /api/kernel/status`, `GET /api/kernel/events`. | `docs/testing/manual-test-guide.md` |
| [x] | Kernel exec evidence uses explicit `degraded-proxy` mode and `syscall-gateway-proxy` probe when true eBPF is unavailable. | `docs/phase_reports/PHASE_15_OPEN_EULER_SMOKE_REPORT.md` |
| [x] | Checkpoint evidence API and `checkpoint.created` timeline event. | `docs/testing/manual-test-guide.md` |
| [x] | Lightweight checkpoint startup recovery with `checkpoint.recovered` and `runtime.recovered` timeline events. | `docs/testing/manual-test-guide.md` |
| [x] | Degraded-copy workspace rollback with `workspace.created`, `workspace.rmrf`, and `workspace.rollback` events. | `docs/testing/manual-test-guide.md` |
| [x] | Reference systemd unit at `deploy/systemd/aortd.service`. | `deploy/systemd/aortd.service` |
| [x] | daemonkill recovery demo script at `scripts/demo-daemonkill.sh`. | `scripts/demo-daemonkill.sh` |
| [x] | openEuler deployment guide and runnable helper scripts. | `docs/deployment_openeuler.md`, `scripts/check_openeuler_env.sh`, `scripts/smoke_openeuler.sh` |

## Remaining Enhancement Targets

| 状态 | 检查项 | 证据位置 |
| --- | --- | --- |
| [ ] | Real overlayfs mount/commit path on openEuler root VM. | Not implemented; next-phase target |
| [ ] | Supervisor retry policy beyond timeout fault recording. | Not implemented; next-phase target |
| [ ] | True `sched_process_exec` eBPF attachment on openEuler root VM. | Not implemented; next-phase target |
| [ ] | Durable CVM page-content and overlay upper-layer checkpointing. | Not implemented; next-phase target |
