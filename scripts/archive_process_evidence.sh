#!/usr/bin/env bash
# Archive process evidence into an exclusive stamped directory.
# Usage: scripts/archive_process_evidence.sh <phase-label> [src-dir ...]
set -euo pipefail

PHASE="${1:-unnamed}"
shift || true
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
STAMP="$(date -u +%Y%m%d-%H%M%S)"
DEST="$ROOT/experiments/results/_process_evidence/${PHASE}-${STAMP}"

if [[ -e "$DEST" ]]; then
  echo "refusing to overwrite existing evidence dir: $DEST" >&2
  exit 1
fi

mkdir -p "$DEST"

if [[ "$#" -eq 0 ]]; then
  set -- \
    "$ROOT/experiments/results/openeuler_smoke" \
    "$ROOT/experiments/results/openeuler_cgroupv2_multi" \
    "$ROOT/experiments/results/openeuler_cgroupv2_limits" \
    "$ROOT/experiments/results/software_real_demo" \
    "$ROOT/experiments/results/deepseek_smoke" \
    "$ROOT/experiments/results/codebase_dag"
fi

copied=0
for src in "$@"; do
  if [[ -e "$src" ]]; then
    base="$(basename "$src")"
    # exclusive copy target
    if [[ -e "$DEST/$base" ]]; then
      echo "refusing to overwrite $DEST/$base" >&2
      exit 1
    fi
    cp -a "$src" "$DEST/$base"
    copied=$((copied + 1))
  fi
done

python3 - "$DEST" "$PHASE" "$copied" <<'PY'
import hashlib, json, pathlib, sys, time
dest = pathlib.Path(sys.argv[1])
phase = sys.argv[2]
copied = int(sys.argv[3])
files = []
for p in sorted(dest.rglob("*")):
    if not p.is_file() or p.name == "EVIDENCE_MANIFEST.json":
        continue
    files.append({
        "path": str(p.relative_to(dest)),
        "bytes": p.stat().st_size,
        "sha256": hashlib.sha256(p.read_bytes()).hexdigest(),
    })
manifest = {
    "schema": "aort.process_evidence.v1",
    "phase": phase,
    "captured_at_utc": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()),
    "copied_roots": copied,
    "file_count": len(files),
    "files": files,
}
out = dest / "EVIDENCE_MANIFEST.json"
if out.exists():
    raise SystemExit(f"refusing to overwrite {out}")
out.write_text(json.dumps(manifest, indent=2) + "\n")
print(dest)
print(f"files={len(files)} roots={copied}")
PY
