# AORT-R Final Evidence Summary

## Overall conclusion
- Real-only openEuler evidence is present and all required real checks passed.
- Git commit: `aaa669dbf1b7d7e0c1ecc805928c0de925c3d238`
- Git branch: `codex/aort-r-upgrade`
- git_dirty: `false`

## generic evidence
| evidence | status |
| --- | --- |
| cvm_memory | passed |
| e1_pressure | passed |
| e1_scheduler | passed |
| e2_fault_isolation | passed |
| e2_pressure_fault | passed |
| ebpf_observer | degraded |
| go_test | passed |
| ipc_shm | passed |
| replay | passed |
| smoke | passed |
| software_real_demo | passed |
| workspace_isolation | passed |
| workspace_probe | passed |

## real-only openEuler evidence
| evidence | status |
| --- | --- |
| real_all | passed |
| real_cgroup_smoke | passed |
| real_env | passed |
| real_pressure_smoke | failed |
| tool_workspace | passed |
| workspace_probe | passed |
| workspace_rmrf | passed |

## evidence_mode_summary
- cgroup_capsule: real-cgroup-v2
- cvm: real-partial
- ebpf: degraded
- ipc: real-partial + real-shm-ipc
- llm: mock
- replay: real-runtime
- resource_sampler: degraded
- scheduler: real-runtime
- tool_workspace: real-overlayfs
- worker_process: real-runtime
- workspace_overlayfs: real-overlayfs

## known_limits
- Portable E1 benchmark may use degraded pressure fallback; real-pressure-smoke proves real-cgroup-v2 ResourceSampler on openEuler.
- Current local final evidence does not prove real-pressure-smoke; run scripts/competition_verify_real.sh on root openEuler.
- eBPF observer experimental path implemented; current submitted evidence is degraded unless openEuler/Linux smoke reports real-ebpf.

## Key file paths
- `experiments/results/final/FINAL_EVIDENCE_INDEX.json`
- `experiments/results/final/FINAL_SUMMARY.md`
- `experiments/results/real_all/REAL_EVIDENCE_INDEX.json`
- `experiments/results/real_all/REAL_VERIFY_SUMMARY.json`

## fresh clone verification
```bash
git clone git@github.com:yxyyyyyyyy/os_complete.git
cd os_complete
bash scripts/competition_verify_real.sh
```
