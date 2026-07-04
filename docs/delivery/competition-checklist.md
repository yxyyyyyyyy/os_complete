# AORT-R Competition Checklist

## Minimum Runnable Evidence

- [x] `POST /api/demo/run` starts real worker processes.
- [x] `GET /api/agents` returns real worker PIDs.
- [x] Worker registration uses Unix Domain Socket.
- [x] Worker heartbeat is tracked by `aortd`.
- [x] Heartbeat timeout marks an Agent as `FAILED`.
- [x] Each Agent has a cgroup path; macOS/non-root returns degraded paths.
- [x] `memory_current`, `pids_current`, and `cpu_stat` are exposed when cgroup v2 is available.
- [x] `freeze`, `unfreeze`, and `kill` Agent APIs exist.
- [x] CVM page store and per-agent page table exist.
- [x] Shared context pages increase ref counts and saved byte/token metrics.
- [x] `context.materialize` is served through the syscall gateway.
- [x] Worker requests `tool.exec` through the syscall gateway.
- [x] Timeline sees `syscall.started` and `syscall.finished`.
- [x] Scheduler supports FIFO, token-CFS, and token-CFS-prefix-affinity.
- [x] Dashboard Overview, AVP, Context, Timeline, and Experiments pages use real APIs.
- [x] Tool timeout fault demo exists at `POST /api/demo/fault/tool-timeout`.
- [x] Experiment JSON/CSV files exist under `experiments/results/`.

## High-Score Evidence Present

- [x] Scheduler DecisionLog API: `GET /api/scheduler/decisions`.
- [x] Scheduler policy API: `POST /api/scheduler/policy`.
- [x] E1 scheduler experiment.
- [x] E2 fault isolation experiment.
- [x] E3 context sharing experiment.
- [x] Dashboard experiments visualization.
- [x] openEuler-oriented cgroup v2 design with graceful degraded mode.

## Remaining Enhancement Targets

- [ ] IPC Blackboard with page-id publish/poll.
- [ ] overlayfs workspace commit/rollback.
- [ ] Supervisor retry and dynamic Fixer spawn.
- [ ] eBPF execve observer.
- [ ] checkpoint recovery.
- [ ] systemd deployment.
- [ ] PSI display.
