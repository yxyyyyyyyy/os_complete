# Reproduce Huawei judge-full codebase-dag (real DeepSeek)

## Preconditions

- Host: openEuler 24.03, root, cgroup2fs writable
- Repo at `/root/aort-r-huawei-run` (clean git tree)
- Env file `/root/aort-deepseek.env` (0600) with `DEEPSEEK_API_KEY` (never commit)

## Commands

```bash
set -a
source /root/aort-deepseek.env
set +a
cd /root/aort-r-huawei-run
export GOTOOLCHAIN=local GOPROXY=https://goproxy.cn,direct GOSUMDB=sum.golang.google.cn
go build -o /tmp/aortctl ./cmd/aortctl
go build -o /tmp/aort-code-worker ./cmd/aort-code-worker

# resource-coder owns ≥20k DeepSeek-authored lines under internal/codebasedag/resourceagent
python3 - <<'PY'
from pathlib import Path
root=Path('internal/codebasedag/resourceagent')
phys=sum(p.read_text(errors='ignore').count('\n') for p in root.rglob('*.go') if '_broken' not in p.parts)
print('resourceagent_physical', phys)
assert phys >= 20000
PY

RUN_ID="judge-full-$(date -u +%Y%m%d-%H%M%S)"
OUT="/root/aort-process-evidence/huawei-${RUN_ID}"
/tmp/aortctl scenario codebase-dag \
  --provider deepseek \
  --model deepseek-v4-flash \
  --workload /root/aort-r-huawei-run \
  --ticket review-remediation \
  --run-id "$RUN_ID" \
  --out "$OUT" \
  --max-calls 10 \
  --worker-command /tmp/aort-code-worker \
  --judge-mode strict \
  --seed-judge \
  --force-fix-once

/tmp/aortctl evidence codebase-dag --run "$OUT/$RUN_ID"
```

## Expected evidence

- `resourceagent_physical_lines` ≥ 20000 (real DeepSeek corpus, not hooks)
- Seed injects `AORT_SEED_BROKEN` into worktree; resource-coder/fixer restore via unified diffs
- Per-agent `processes/*.json` with real PID + cgroup on Linux
- `fault_report`, `communication_comparison`, `cvm_metrics`
- Fixer loop appears when `--force-fix-once` / strict mode
- `ValidateRun` passes
