#!/usr/bin/env bash
# Real openEuler / cgroup v2 resource-isolation driver.
# Runs baseline, isolation-only, and aort-r inside a dedicated temp root.
# Does not mutate the developer's primary workspace.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${1:-$ROOT/experiments/results/review_remediation/resource_isolation_openeuler}"
RUNS="${RUNS:-3}"
TIMEOUT="${TIMEOUT:-8s}"

mkdir -p "$OUT"
export GOTOOLCHAIN="${GOTOOLCHAIN:-go1.22.12}"
export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"

if [[ "$(uname -s)" != "Linux" ]]; then
  echo "skip: requires Linux/openEuler host" >&2
  exit 0
fi

if [[ "$(stat -fc %T /sys/fs/cgroup 2>/dev/null || true)" != "cgroup2fs" ]]; then
  echo "skip: cgroup v2 not mounted as cgroup2fs" >&2
  exit 0
fi

cd "$ROOT"
export AORT_REQUIRE_OPENEULER=1
go test ./internal/review/ -run TestResourceIsolationOpenEulerSmoke -count=1 -timeout=20m \
  -v 2>&1 | tee "$OUT/go_test.txt" || true

go run ./cmd/aortctl scenario resource-isolation \
  --mode all \
  --runs "$RUNS" \
  --warmup 0 \
  --timeout "$TIMEOUT" \
  --out "$OUT/scenario" 2>&1 | tee "$OUT/scenario.log" || true

cat > "$OUT/README.md" <<EOF
# openEuler resource isolation evidence

- host: $(hostname)
- date: $(date -u +%Y-%m-%dT%H:%M:%SZ)
- cgroup_fs: $(stat -fc %T /sys/fs/cgroup)
- runs: $RUNS
- note: baseline executes only under dedicated temp roots created by the scenario
- evidence_mode: must remain degraded when cgroup writes fail; never rewrite as real
EOF

echo "wrote $OUT"
