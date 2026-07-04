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
- [x] `llm.call` is served through the syscall gateway with mock provider usage metrics.
- [x] Worker requests `tool.exec` through the syscall gateway.
- [x] `ipc.publish` and `ipc.poll` transfer CVM page IDs and track avoided copy bytes.
- [x] `agent.spawn` records Reviewer-triggered Fixer creation through the syscall gateway.
- [x] Timeline sees `syscall.started` and `syscall.finished`.
- [x] Timeline sees `kernel.observer_disabled` and `kernel.exec` in honest degraded-proxy mode.
- [x] Timeline sees `ipc.published`, `ipc.polled`, `llm.called`, `agent.spawn.requested`, and `agent.spawned`.
- [x] Scheduler supports FIFO, token-CFS, and token-CFS-prefix-affinity.
- [x] Pressure monitor exposes PSI/degraded status through `GET /api/pressure/status`.
- [x] Scheduler timeline events include pressure mode/throttle/avg10 evidence.
- [x] Dashboard Overview, AVP, Context, Timeline, and Experiments pages use real APIs.
- [x] Tool timeout fault demo exists at `POST /api/demo/fault/tool-timeout`.
- [x] Workspace rmrf rollback demo exists at `POST /api/demo/fault/rmrf`.
- [x] Lightweight checkpoint snapshots are exposed through `GET /api/checkpoints`.
- [x] Startup recovery report is exposed through `GET /api/recovery/status`.
- [x] Experiment JSON/CSV files exist under `experiments/results/`.

## High-Score Evidence Present

- [x] Scheduler DecisionLog API: `GET /api/scheduler/decisions`.
- [x] Scheduler policy API: `POST /api/scheduler/policy`.
- [x] PSI pressure-aware scheduling evidence: `pressure.sampled`, `scheduler.pressure_throttle`, and `pressure_*` scheduler fields.
- [x] E1 scheduler experiment.
- [x] E2 fault isolation experiment.
- [x] E3 context sharing and IPC avoided-copy experiment.
- [x] Dashboard experiments visualization.
- [x] openEuler-oriented cgroup v2 design with graceful degraded mode.
- [x] IPC Blackboard API: `GET /api/ipc/metrics`, `GET /api/ipc/topics`.
- [x] LLM Router abstraction with mock provider and llama.cpp timing parser.
- [x] Kernel Observer API: `GET /api/kernel/status`, `GET /api/kernel/events`.
- [x] Kernel exec evidence uses explicit `degraded-proxy` mode and `syscall-gateway-proxy` probe when true eBPF is unavailable.
- [x] Checkpoint evidence API and `checkpoint.created` timeline event.
- [x] Lightweight checkpoint startup recovery with `checkpoint.recovered` and `runtime.recovered` timeline events.
- [x] Degraded-copy workspace rollback with `workspace.created`, `workspace.rmrf`, and `workspace.rollback` events.
- [x] Reference systemd unit at `deploy/systemd/aortd.service`.
- [x] daemonkill recovery demo script at `scripts/demo-daemonkill.sh`.
- [x] openEuler deployment guide and runnable helper scripts.

## Remaining Enhancement Targets

- [ ] Real overlayfs mount/commit path on openEuler root VM.
- [ ] Supervisor retry policy beyond timeout fault recording.
- [ ] True `sched_process_exec` eBPF attachment on openEuler root VM.
- [ ] Durable CVM page-content and overlay upper-layer checkpointing.
