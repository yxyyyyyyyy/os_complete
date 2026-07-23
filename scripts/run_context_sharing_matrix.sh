#!/usr/bin/env bash
# Open World context-sharing matrix: sizes × agents × ratios × modes.
# Portable/degraded hosts still emit measured/derived/unsupported labels.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="${1:-$ROOT/experiments/results/review_remediation/context_sharing_matrix}"
RUNS="${RUNS:-1}"
TIMEOUT="${TIMEOUT:-5s}"

mkdir -p "$OUT"
cd "$ROOT"
export GOTOOLCHAIN="${GOTOOLCHAIN:-go1.22.12}"
export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"

go run ./cmd/aortctl scenario context-sharing \
  --matrix \
  --runs "$RUNS" \
  --timeout "$TIMEOUT" \
  --out "$OUT" | tee "$OUT/cli.log"

cat > "$OUT/README.md" <<EOF
# context-sharing matrix

- host: $(hostname)
- date: $(date -u +%Y-%m-%dT%H:%M:%SZ)
- runs_per_cell: $RUNS
- sizes: 4K / 64K / 256K / 1M
- agents: 2 / 4 / 8 / 16
- ratios: 0 / 0.25 / 0.5 / 0.75 / 1.0
- modes: full-copy / shared-ipc / aort-r
- note: CVM page reuse is not model KV-cache; IPC counters are not claimed as kernel zero-copy.
EOF

echo "wrote $OUT"
