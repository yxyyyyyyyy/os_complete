# Process Evidence Index

Immutable stamped directories (never overwrite):

- `huawei-deepseek-smoke-pull-20260723-064342`
- `huawei-phase4-20260723-062207`
- `huawei-phase4-remote-pull-20260723-063934`
- `huawei-post30k-pull-20260723-070629`
- `huawei-post30k-pull-20260723-070933`
- `loc-gate-30000-20260723-065828`
- `local-code-progress-checkpoint-20260723-065446`

Archive helper: `bash scripts/archive_process_evidence.sh <phase> [srcs...]`

## huawei-live-full-live-20260723-074147
- remote: `/root/aort-process-evidence/huawei-live-full-live-20260723-074147`
- local pull: `experiments/results/_process_evidence/huawei-live-full-pull-20260723-074737`
- local mirror: `experiments/results/_process_evidence/huawei-live-full-live-20260723-074147`
- status: `all_required_passed=true`, calls=7, model=`deepseek-v4-flash`, evidence_mode=`real-api`
- workload: physical_go_lines=36706, nonblank_go_lines=34743, tracked_go_files=162
- notes: also preserved partial failures `huawei-live-partial-timeout-20260723-072746` and `huawei-live-partial-emptycontent-20260723-073412` on remote `/root/aort-process-evidence/`
