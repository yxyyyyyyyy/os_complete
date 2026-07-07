#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

RESULT_DIR="experiments/results/real_all"
LOG_DIR="experiments/results/real_all/logs"
SUMMARY_JSON="$RESULT_DIR/REAL_VERIFY_SUMMARY.json"
SUMMARY_MD="$RESULT_DIR/REAL_VERIFY_SUMMARY.md"
STEPS_TSV="$RESULT_DIR/REAL_VERIFY_STEPS.tsv"
mkdir -p "$LOG_DIR"
: >"$STEPS_TSV"

FAILED_STEPS=()

quote_command() {
  local quoted=""
  local part
  local escaped
  for part in "$@"; do
    printf -v escaped '%q' "$part"
    quoted="$quoted $escaped"
  done
  printf '%s' "${quoted# }"
}

write_dependency_failure_summary() {
  local reason="$1"
  cat >"$SUMMARY_JSON" <<JSON
{
  "script": "competition_verify_real",
  "evidence_mode": "real-openeuler",
  "steps": [
    {
      "name": "dependency_check",
      "status": "failed",
      "command": "command -v go && command -v python3",
      "log_file": "",
      "error": "$reason"
    }
  ],
  "all_passed": false,
  "failed_steps": ["dependency_check"],
  "real_required": true
}
JSON
  cat >"$SUMMARY_MD" <<MD
# AORT-R Real openEuler Verification

- all_passed: false
- failed_steps: dependency_check
- reason: $reason
MD
}

if ! command -v go >/dev/null 2>&1; then
  write_dependency_failure_summary "go not found"
  printf 'go not found\n' >&2
  exit 1
fi

if ! command -v python3 >/dev/null 2>&1; then
  write_dependency_failure_summary "python3 not found"
  printf 'python3 not found\n' >&2
  exit 1
fi

write_summary() {
  local all_passed="true"
  if [ "${#FAILED_STEPS[@]}" -gt 0 ]; then
    all_passed="false"
  fi
  python3 - "$SUMMARY_JSON" "$SUMMARY_MD" "$STEPS_TSV" "$all_passed" <<'PY'
import json
import pathlib
import sys
from datetime import datetime, timezone

json_path = pathlib.Path(sys.argv[1])
md_path = pathlib.Path(sys.argv[2])
steps_path = pathlib.Path(sys.argv[3])
all_passed = sys.argv[4] == "true"

steps = []
failed = []
if steps_path.exists():
    for line in steps_path.read_text(encoding="utf-8").splitlines():
        if not line:
            continue
        name, status, code, log_file, command = line.split("\t", 4)
        step = {
            "name": name,
            "status": status,
            "command": command,
            "log_file": log_file,
        }
        if code != "0":
            step["exit_code"] = int(code)
        steps.append(step)
        if status != "passed":
            failed.append(name)

summary = {
    "script": "competition_verify_real",
    "timestamp": datetime.now(timezone.utc).isoformat(),
    "evidence_mode": "real-openeuler",
    "steps": steps,
    "all_passed": all_passed and not failed,
    "failed_steps": failed,
    "real_required": True,
}
json_path.write_text(json.dumps(summary, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")

lines = [
    "# AORT-R Real openEuler Verification",
    "",
    f"- evidence_mode: {summary['evidence_mode']}",
    f"- real_required: {str(summary['real_required']).lower()}",
    f"- all_passed: {str(summary['all_passed']).lower()}",
    "",
    "| step | status | log |",
    "| --- | --- | --- |",
]
for step in steps:
    lines.append(f"| {step['name']} | {step['status']} | `{step['log_file']}` |")
lines.extend(["", "## failed_steps"])
if failed:
    lines.extend(f"- {name}" for name in failed)
else:
    lines.append("- none")
md_path.write_text("\n".join(lines) + "\n", encoding="utf-8")
PY
}

run_required_step() {
  local name="$1"
  shift
  local log_file="$LOG_DIR/${name}.log"
  local command
  command="$(quote_command "$@")"
  printf '\n== %s ==\n' "$name"
  printf 'command=%s\n' "$command"
  printf 'log_file=%s\n' "$log_file"
  set +e
  "$@" >"$log_file" 2>&1
  local code=$?
  set -e
  if [ "$code" -eq 0 ]; then
    printf '%s\tpassed\t0\t%s\t%s\n' "$name" "$log_file" "$command" >>"$STEPS_TSV"
    return 0
  fi
  printf '%s\tfailed\t%s\t%s\t%s\n' "$name" "$code" "$log_file" "$command" >>"$STEPS_TSV"
  FAILED_STEPS+=("$name")
  write_summary
  cat "$log_file" >&2
  exit 1
}

run_required_step real_env bash scripts/verify_real_openeuler_env.sh
run_required_step go_test go test ./...
run_required_step real_cgroup_smoke go run ./cmd/aortctl experiment real-cgroup-smoke --out experiments/results/real_cgroup_smoke
run_required_step real_pressure_smoke go run ./cmd/aortctl experiment real-pressure-smoke --runs 3 --out experiments/results/real_pressure_smoke --require-real
run_required_step workspace_probe go run ./cmd/aortctl workspace probe --out experiments/results/workspace_probe.json --require-real
run_required_step workspace_rmrf go run ./cmd/aortctl demo fault workspace-rmrf --out experiments/results --require-real-overlayfs
run_required_step tool_workspace go run ./cmd/aortctl demo tool-workspace --out experiments/results --require-real-overlayfs
run_required_step real_all go run ./cmd/aortctl experiment real-all --runs 3 --out experiments/results/real_all
run_required_step evidence_final go run ./cmd/aortctl evidence final --out experiments/results/final

write_summary
printf '\nAORT-R real openEuler verification completed.\n'
printf 'See experiments/results/real_all/REAL_VERIFY_SUMMARY.json\n'
