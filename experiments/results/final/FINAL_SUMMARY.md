# AORT-R Final Evidence Summary

## Overall conclusion
- Real-only openEuler evidence is present and all required real checks passed.
- Git commit: `1c8574d9577b06bf6882633579fc26fd987619ca`
- Git branch: `main`
- git_dirty: `true`

## generic evidence
| evidence | status |
| --- | --- |
| e1_pressure | passed |
| e1_scheduler | passed |
| e2_fault_isolation | passed |
| e2_pressure_fault | passed |
| go_test | passed |
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
| real_pressure_smoke | passed |
| tool_workspace | passed |
| workspace_probe | passed |
| workspace_rmrf | passed |

## evidence_mode_summary
- cgroup_capsule: real-cgroup-v2
- cvm: real-partial
- ebpf: planned
- ipc: real-partial
- llm: mock
- resource_sampler: real-cgroup-v2
- scheduler: real-runtime
- tool_workspace: real-overlayfs
- worker_process: real-runtime
- workspace_overlayfs: real-overlayfs

## known_limits
- Portable E1 benchmark may use degraded pressure fallback; real-pressure-smoke proves real-cgroup-v2 ResourceSampler on openEuler.

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
