# openEuler Smoke Evidence

This directory now contains both historical degraded-real smoke evidence and
the latest real cgroup v2 evidence captured on openEuler 24.03 LTS.

Latest status:

```json
{
  "evidence_mode": "real-cgroup-v2",
  "cgroup_fs": "cgroup2fs",
  "capsule_mode": "real",
  "real_cgroup_v2": true
}
```

Older degraded files are retained as before/after evidence only. They do not
represent the current cgroup v2 state.

## Files

| File | Status | Notes |
| --- | --- | --- |
| `REAL_CGROUP_V2_SUMMARY.md` | current summary | Latest real cgroup v2 result. |
| `capsule_real.json` | current real output | `capsule_mode=real`, `real_cgroup_v2=true`. |
| `agent_summary.json` | current real summary | Worker PID, cgroup path, memory/pids counters, freeze/unfreeze/kill. |
| `env_check.json` | current real output | Shows `cgroup2fs`, root, writable cgroup mount. |
| `go_test_cgroupv2_7d939c2.txt` | current real output | `go test ./...` passed on openEuler. |
| `aort-r-openeuler-7d939c2-cgroupv2-real-evidence.tgz` | current evidence package | Compact archive of the real smoke evidence. |
| `manual_smoke_summary.json` | historical summary | Older degraded API smoke record retained for comparison. |
| `env_check_latest.txt` | historical output | Shows older cgroup v2 failure. |

## Real cgroup v2 requirement

Real smoke must run on a host where:

```bash
stat -fc %T /sys/fs/cgroup
```

prints:

```text
cgroup2fs
```
