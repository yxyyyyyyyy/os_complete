# Evidence Mode

AORT-R competition evidence uses one shared vocabulary from `internal/evidence`.
Every JSON evidence object should include `evidence_mode`; fallback paths also
include `fallback_reason`.

| mode | meaning |
| --- | --- |
| `real` | Generic real implementation evidence when a more specific real mode does not apply. |
| `real-cgroup-v2` | Live Linux/openEuler cgroup v2 evidence such as `memory.current`, `pids.current`, `cpu.stat`, freeze, and kill. |
| `real-runtime` | Real AORT-R runtime path, worker process, syscall gateway, scheduler, or benchmark execution. |
| `real-api` | A real external API call completed successfully, such as DeepSeek when configured. |
| `real-partial` | Real partial implementation with an explicit boundary. |
| `real-overlayfs` | Workspace isolation mounted through Linux overlayfs. |
| `degraded` | Runtime continues, but the host lacks a required OS feature or metric. |
| `degraded-copy` | Workspace isolation fallback using copied lowerdir contents instead of overlayfs. |
| `mock` | Deterministic mock implementation, used by default for LLM calls. |
| `simulation` | Legacy non-runtime simulation evidence. |
| `planned` | Design exists but no real implementation evidence is present. |
| `missing` | Required evidence file or signal is not available. |

Important boundaries:

- CVM is `real-partial`: it performs page-level context reuse, hash dedup, and materialization optimization. It is not real model KV Cache sharing.
- IPC is `real-partial`: it supports page-reference IPC and optional memfd/mmap shared-memory IPC. It is not kernel zero-copy.
- LLM is `mock` by default. DeepSeek success may be `real-api`; API failure falls back to `mock` with `fallback=true` and `fallback_reason`.
- eBPF observer experimental path implemented; current submitted evidence is degraded unless openEuler/Linux smoke reports real-ebpf. Do not report `real-ebpf` without real load, attach, worker PID observation, and cleanup evidence.
- Overlay workspace isolation is `real-overlayfs` only after a successful mount. Otherwise it is `degraded-copy`.
- The current workspace mode must be read from `workspace_probe.json` and
  `workspace_isolation_evidence.json`: keep `real-overlayfs` only when the probe
  shows `mount_test_success=true` and `merged_is_mountpoint=true`; degraded
  reruns must write `degraded-copy` with `fallback_reason`.

Primary evidence files:

- `experiments/results/final/FINAL_EVIDENCE_INDEX.json`
- `experiments/results/e1/e1_resource_aware.json`
- `experiments/results/workspace_probe.json`
- `experiments/results/workspace_isolation_evidence.json`
- `experiments/results/software_real_demo/result.json`
- `experiments/results/ebpf_smoke/ebpf_smoke.json`
- `experiments/results/ipc_shm/ipc_shm_smoke.json`
- `experiments/results/cvm_memory/cvm_memory_smoke.json`
- `experiments/results/replay/replay_result.json`
