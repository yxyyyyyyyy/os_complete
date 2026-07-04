# AORT-R openEuler Deployment Guide

## Target Environment

- OS: openEuler 24.03 LTS x86_64 preferred.
- Kernel: cgroup v2 capable kernel, 6.x recommended.
- Go: 1.22 or newer.
- Node.js: 18 or newer for the Vue dashboard.
- Privilege: root is recommended for real cgroup freeze/unfreeze/kill evidence.

The runtime also runs on macOS and non-root Linux in degraded mode. In degraded mode, worker processes, UDS syscalls, CVM, IPC, scheduler, checkpoint, experiments, and dashboard remain available, while cgroup writes return structured degraded errors.

## Environment Check

```bash
scripts/check_env.sh
```

Expected checks:

- Go is available.
- Node/npm is available.
- cgroup v2 is mounted at `/sys/fs/cgroup`.
- `cgroup.controllers` is readable.
- overlayfs is listed in `/proc/filesystems` when available.

Missing root, overlayfs, or cgroup v2 does not stop the local demo; it changes OS-backed modules to degraded mode.

## Start Runtime

```bash
GOCACHE="$PWD/.cache/go-build" go test ./...
GOCACHE="$PWD/.cache/go-build" go run ./cmd/aortd --config configs/dev.yaml
```

In another terminal:

```bash
curl -s -X POST http://127.0.0.1:8080/api/demo/run
curl -s http://127.0.0.1:8080/api/agents
curl -s http://127.0.0.1:8080/api/syscalls
curl -s http://127.0.0.1:8080/api/ipc/metrics
curl -s http://127.0.0.1:8080/api/checkpoints
```

## Start Dashboard

```bash
cd dashboard
npm install
npm run dev
```

Open the dashboard URL printed by Vite, usually `http://127.0.0.1:5173/`.

## Run Experiments

```bash
scripts/run_experiments.sh
```

Outputs:

- `experiments/results/e1-scheduler.json/csv`
- `experiments/results/e2-fault.json/csv`
- `experiments/results/e3-context.json/csv`

## Evidence Map

| Requirement | Evidence |
|---|---|
| Worker process runtime | `/api/agents`, PID fields, UDS registration events |
| cgroup capsule | `capsule_mode`, `cgroup_path`, freeze/unfreeze/kill APIs |
| Context optimization | `/api/context/pages`, `/api/context/stats`, E3 results |
| Efficient communication | `ipc.publish`, `ipc.poll`, `/api/ipc/metrics` |
| Unified syscall abstraction | `/api/syscalls`, `tool.exec`, `llm.call`, `agent.spawn` |
| Dynamic task generation | `agent.spawn.requested`, `agent.spawned` timeline events |
| Fault isolation | `POST /api/demo/fault/tool-timeout`, `/api/faults`, E2 results |
| Workspace rollback | `POST /api/demo/fault/rmrf`, `workspace.rollback`, `base_intact` |
| Checkpoint evidence | `/api/checkpoints`, `checkpoint.created` timeline events |

## Known Limits

- Workspace rollback is implemented in degraded-copy mode and proves that an Agent workspace can be destroyed and restored from a base snapshot without touching the base. Real overlayfs mount/commit is the next openEuler-root enhancement.
- eBPF observer is planned as an enhancement; the current timeline is application/syscall/runtime level.
- DeepSeek and llama.cpp providers are represented by the `llm.Router` interface and mock provider in this repository; real provider credentials and local model paths should be configured outside Git.
