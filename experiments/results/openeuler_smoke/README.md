# openEuler Smoke Evidence

This directory records the degraded-real smoke evidence captured on an
openEuler 24.03 LTS server.

The run is real for Go tests and Runtime APIs, but it is not real cgroup v2
evidence. The server reported `/sys/fs/cgroup` as `tmpfs`, not `cgroup2fs`, so
the capsule layer correctly fell back to `capsule_mode=degraded`.

## Files

| File | Status | Notes |
| --- | --- | --- |
| `manual_smoke_summary.json` | real summary | HTTP status summary from manual degraded API smoke. |
| `env_check.txt` | real output | Shows openEuler/root plus cgroup v2 failure. |
| `go_test.txt` | real output | `go test ./...` passed with Go 1.22.12 on openEuler. |
| `agents.json` | real output | Contains real worker PIDs and degraded capsule paths. |
| `agent_summary.json` | real summary | Extracted from `manual_smoke_summary.json` and API output. |
| `syscalls.json` | real output | Syscall gateway records from the degraded smoke run. |
| `context_stats.json` | real output | CVM stats from the degraded smoke run. |
| `scheduler_decisions.json` | real output | Scheduler decision log from the degraded smoke run. |
| `fault_tool_timeout.json` | real output | Tool timeout fault recovered evidence. |
| `kill.json` | real output | Agent kill API returned 200. |
| `health.json` | placeholder | `unavailable_from_manual_run`; HTTP 200 is recorded in `manual_smoke_summary.json`. |
| `demo_run.json` | placeholder | `unavailable_from_manual_run`; HTTP 202 is recorded in `manual_smoke_summary.json`. |

## Real cgroup v2 requirement

The next real smoke must run on a host where:

```bash
stat -fc %T /sys/fs/cgroup
```

prints:

```text
cgroup2fs
```
