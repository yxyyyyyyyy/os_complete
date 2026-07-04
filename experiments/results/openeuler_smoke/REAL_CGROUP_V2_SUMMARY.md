# REAL_CGROUP_V2_SUMMARY

Latest status: `real-passed`.

The current openEuler 24.03 LTS validation is no longer the old degraded smoke.
The latest evidence shows:

```json
{
  "evidence_mode": "real",
  "cgroup_fs": "cgroup2fs",
  "capsule_mode": "real",
  "real_cgroup_v2": true,
  "memory_current": 8192,
  "pids_current": 5,
  "freeze": "200",
  "unfreeze": "200",
  "kill": "200"
}
```

Primary files:

- `env_check.json`
- `capsule_real.json`
- `agent_summary.json`
- `go_test_cgroupv2_7d939c2.txt`
- `smoke_cgroupv2_7d939c2.log`
- `aort-r-openeuler-7d939c2-cgroupv2-real-evidence.tgz`

Historical degraded files in this directory are retained as before/after
evidence only. They do not describe the current cgroup v2 state.
