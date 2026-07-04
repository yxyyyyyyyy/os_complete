# PHASE_16_CGROUP_V2_REAL_REPORT

mode=real-passed

本报告是 `PHASE_16_OPEN_EULER_REAL_CGROUP_REPORT.md` 的答辩版摘要，用于
消除旧 degraded 文档造成的歧义。

## 当前结论

最新 openEuler 24.03 LTS / Linux root / unified cgroup v2 环境已经跑通
AORT-R real cgroup v2 capsule：

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

旧 degraded evidence 只作为历史记录和对照组，不代表当前状态。

## 证据位置

| 证据 | 文件 |
| --- | --- |
| openEuler 环境检查 | `experiments/results/openeuler_smoke/env_check.json` |
| real capsule 摘要 | `experiments/results/openeuler_smoke/capsule_real.json` |
| Agent/cgroup 摘要 | `experiments/results/openeuler_smoke/agent_summary.json` |
| go test 输出 | `experiments/results/openeuler_smoke/go_test_cgroupv2_7d939c2.txt` |
| smoke 输出 | `experiments/results/openeuler_smoke/smoke_cgroupv2_7d939c2.log` |
| 证据包 | `experiments/results/openeuler_smoke/aort-r-openeuler-7d939c2-cgroupv2-real-evidence.tgz` |

## 当前 real 模块

- Worker Process：真实 worker PID。
- Cgroup Capsule：真实 cgroup v2，`capsule_mode=real`。
- Syscall Gateway：真实 Runtime syscall record。
- Scheduler：真实 token-CFS-prefix-affinity decision log。
- CVM：真实 Runtime page store / saved bytes / saved tokens。
- Page-reference IPC：真实 Runtime page-id publish/poll 与 avoided-copy metrics。
- Fault Supervisor：真实 tool-timeout recovery record。

## 当前非 real 模块

| 模块 | 状态 | 说明 |
| --- | --- | --- |
| Workspace Isolation | `degraded` | 当前为 `degraded-copy`，不宣称 overlayfs real。 |
| Kernel Observer | `degraded` | 当前为 `degraded-proxy`，不宣称 eBPF real。 |
| PSI Monitor | `unavailable/degraded` | 当前 openEuler 实例无 `/proc/pressure`。 |
| eBPF Observer | `planned` | 未实现。 |
| LLM Provider | `mock` | 默认 mock provider；DeepSeek/llama.cpp 未启用。 |

## 重新验证命令

```bash
stat -fc %T /sys/fs/cgroup
bash scripts/check_openeuler_env.sh
bash scripts/smoke_openeuler.sh
bash scripts/smoke_cgroupv2_multi_agent.sh
bash scripts/smoke_cgroupv2_limits.sh
```
