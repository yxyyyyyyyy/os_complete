# REAL_CGROUP_V2_SUMMARY

Latest status: `real-passed`.

The current openEuler 24.03 LTS validation is no longer the old degraded smoke.
The latest evidence shows:

```json
{
  "evidence_mode": "real-cgroup-v2",
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

Additional current evidence:

- `../openeuler_cgroupv2_multi/multi_agent_capsules.json`
- `../openeuler_cgroupv2_multi/multi_agent_freeze_unfreeze.json`
- `../openeuler_cgroupv2_multi/multi_agent_kill_recovery.json`
- `../openeuler_cgroupv2_multi/multi_agent_summary.json`
- `../openeuler_cgroupv2_limits/memory_limit_enforced.json`
- `../openeuler_cgroupv2_limits/pids_limit_enforced.json`
- `../openeuler_cgroupv2_limits/cpu_quota_stat.json`
- `../openeuler_cgroupv2_limits/limit_summary.json`

Historical degraded files in this directory are retained as before/after
evidence only. They do not describe the current cgroup v2 state.
