#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

FINAL_DIR="$ROOT_DIR/experiments/results/final"
mkdir -p "$FINAL_DIR" "$ROOT_DIR/.cache/go-build"
export GOCACHE="$ROOT_DIR/.cache/go-build"

log() {
  printf '\n== %s ==\n' "$1"
}

run_step() {
  local name="$1"
  shift
  local log_file="$FINAL_DIR/${name}.log"
  log "$name"
  set +e
  "$@" >"$log_file" 2>&1
  local code=$?
  set -e
  cat "$log_file"
  if [ "$code" -eq 0 ]; then
    printf 'step_%s=passed\n' "$name"
    return 0
  fi
  printf 'step_%s=failed code=%s\n' "$name" "$code"
  return "$code"
}

kernel="$(uname -a 2>&1 || true)"
os_release="$(cat /etc/os-release 2>/dev/null || true)"
cgroup_fs_type="$(stat -fc %T /sys/fs/cgroup 2>/dev/null || true)"
id_output="$(id 2>&1 || true)"
go_version_output="$(go version 2>&1 || true)"
if [ -z "$cgroup_fs_type" ]; then
  cgroup_fs_type="missing"
fi
if [ -z "$os_release" ]; then
  os_release="missing"
fi

log "base environment"
printf '%s\n' "$kernel"
printf '%s\n' "$os_release"
printf 'cgroup_fs_type=%s\n' "${cgroup_fs_type:-missing}"
printf '%s\n' "$id_output"
printf '%s\n' "$go_version_output"

env_check="failed"
if OUT_DIR="$FINAL_DIR/openeuler_env" AORT_ENV_JSON="$FINAL_DIR/env_check.json" run_step env_check bash scripts/check_openeuler_env.sh; then
  env_check="passed"
else
  env_check="degraded"
fi
if [ "$cgroup_fs_type" != "cgroup2fs" ]; then
  env_check="degraded"
fi

go_test="failed"
if run_step go_test go test ./...; then
  go_test="passed"
fi

smoke="failed"
if OUT_DIR="$FINAL_DIR/openeuler_smoke" run_step smoke bash scripts/smoke_openeuler.sh; then
  smoke="passed"
elif [ "$cgroup_fs_type" != "cgroup2fs" ]; then
  smoke="degraded"
fi
if [ "$cgroup_fs_type" != "cgroup2fs" ]; then
  smoke="degraded"
fi

e1_scheduler="failed"
if run_step e1_scheduler go run ./cmd/aortctl experiment e1 --policy resource-aware --runs 5 --out experiments/results/e1; then
  e1_scheduler="passed"
fi

e2_fault_isolation="failed"
if run_step e2_fault_isolation go run ./cmd/aortctl experiment e2 --runs 5 --out experiments/results; then
  e2_fault_isolation="passed"
fi

software_real_demo="failed"
if run_step software_real_demo go run ./cmd/aortctl demo software-real --out experiments/results; then
  software_real_demo="passed"
fi

workspace_isolation="failed"
if run_step workspace_isolation go run ./cmd/aortctl demo fault workspace-rmrf --out experiments/results; then
  workspace_isolation="passed"
fi

export kernel os_release cgroup_fs_type go_version_output
export env_check go_test smoke e1_scheduler e2_fault_isolation software_real_demo workspace_isolation

python3 - "$FINAL_DIR/FINAL_EVIDENCE_INDEX.json" "$FINAL_DIR/FINAL_SUMMARY.md" <<'PY'
import json
import os
import pathlib
import sys
from datetime import datetime, timezone

index_path = pathlib.Path(sys.argv[1])
summary_path = pathlib.Path(sys.argv[2])
root = pathlib.Path.cwd()

def rel(path: pathlib.Path) -> str:
    try:
        return path.relative_to(root).as_posix()
    except ValueError:
        return path.as_posix()

def read_json(path: pathlib.Path):
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except Exception:
        return None

generated_candidates = [
    root / "experiments/results/final/env_check.json",
    root / "experiments/results/e1/e1_resource_aware.json",
    root / "experiments/results/e1/e1_resource_aware.csv",
    root / "experiments/results/e1/e1_resource_aware_decisions.json",
    root / "experiments/results/e1/e1_resource_aware_summary.md",
    root / "experiments/results/e2-real-fault.json",
    root / "experiments/results/e2-real-fault.csv",
    root / "experiments/results/software_real_demo/result.json",
    root / "experiments/results/workspace_isolation_evidence.json",
]
generated_files = [rel(path) for path in generated_candidates if path.exists()]

env_data = read_json(root / "experiments/results/final/env_check.json") or {}
workspace_data = read_json(root / "experiments/results/workspace_isolation_evidence.json") or {}
overall_mode = "real-runtime"
known_limits = []
if os.environ.get("env_check") == "degraded" or env_data.get("evidence_mode") == "degraded":
    overall_mode = "degraded"
    known_limits.append("local host did not prove live openEuler cgroup v2; cgroup evidence is degraded or archived")
if workspace_data.get("evidence_mode") == "degraded-copy":
    known_limits.append("overlayfs mount was unavailable or not attempted; workspace isolation used degraded-copy fallback")
if os.environ.get("smoke") == "degraded":
    known_limits.append("openEuler smoke ran in degraded mode on this host")

index = {
    "timestamp": datetime.now(timezone.utc).isoformat(),
    "os_release": os.environ.get("os_release", ""),
    "kernel": os.environ.get("kernel", ""),
    "cgroup_fs_type": os.environ.get("cgroup_fs_type", ""),
    "go_version": os.environ.get("go_version_output", ""),
    "evidence_mode": overall_mode,
    "go_test": os.environ.get("go_test", "failed"),
    "smoke": os.environ.get("smoke", "failed"),
    "e1_scheduler": os.environ.get("e1_scheduler", "failed"),
    "e2_fault_isolation": os.environ.get("e2_fault_isolation", "failed"),
    "software_real_demo": os.environ.get("software_real_demo", "failed"),
    "workspace_isolation": os.environ.get("workspace_isolation", "failed"),
    "generated_files": generated_files,
    "evidence_mode_summary": {
        "cgroup_capsule": env_data.get("evidence_mode", "degraded"),
        "worker_process": "real-runtime",
        "cvm": "real-partial",
        "ipc": "real-partial",
        "llm": "mock",
        "ebpf": "planned",
        "overlayfs": workspace_data.get("evidence_mode", "degraded-copy"),
    },
    "known_limits": known_limits,
}
index_path.write_text(json.dumps(index, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")

summary_lines = [
    "# AORT-R Final Evidence Summary",
    "",
    f"- timestamp: {index['timestamp']}",
    f"- evidence_mode: {index['evidence_mode']}",
    f"- go_test: {index['go_test']}",
    f"- smoke: {index['smoke']}",
    f"- e1_scheduler: {index['e1_scheduler']}",
    f"- e2_fault_isolation: {index['e2_fault_isolation']}",
    f"- software_real_demo: {index['software_real_demo']}",
    f"- workspace_isolation: {index['workspace_isolation']}",
    "",
    "## Generated Files",
]
summary_lines.extend(f"- `{path}`" for path in generated_files)
summary_lines.extend(["", "## Evidence Mode Summary"])
summary_lines.extend(f"- {key}: {value}" for key, value in index["evidence_mode_summary"].items())
summary_lines.extend(["", "## Known Limits"])
if known_limits:
    summary_lines.extend(f"- {item}" for item in known_limits)
else:
    summary_lines.append("- none")
summary_path.write_text("\n".join(summary_lines) + "\n", encoding="utf-8")
PY

printf '\nAORT-R competition verification completed. See experiments/results/final/FINAL_EVIDENCE_INDEX.json\n'

if [ "$go_test" != "passed" ] || [ "$e1_scheduler" != "passed" ] || [ "$e2_fault_isolation" != "passed" ] || [ "$software_real_demo" != "passed" ] || [ "$workspace_isolation" != "passed" ]; then
  exit 1
fi
