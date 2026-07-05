#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
CONFIG_PATH="${CONFIG_PATH:-configs/dev.yaml}"
OUT_DIR="${OUT_DIR:-experiments/results/openeuler_cgroupv2_multi}"
mkdir -p "$OUT_DIR" .cache/go-build
export GOCACHE="$PWD/.cache/go-build"
export OUT_DIR

if [ "$(stat -fc %T /sys/fs/cgroup 2>/dev/null || true)" != "cgroup2fs" ]; then
  echo "cgroup v2 is required: /sys/fs/cgroup is not cgroup2fs" >&2
  exit 1
fi

if [ "$(id -u)" -ne 0 ]; then
  echo "root is required for cgroup v2 capsule smoke" >&2
  exit 1
fi

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required" >&2
  exit 1
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 is required" >&2
  exit 1
fi

AORTD_PID=""
AORTD_PGID=""
AORTD_LOG="$OUT_DIR/runtime_start.log"

cleanup() {
  if [ -n "$AORTD_PGID" ]; then
    kill -TERM -- "-$AORTD_PGID" 2>/dev/null || true
    sleep 1
    kill -KILL -- "-$AORTD_PGID" 2>/dev/null || true
  elif [ -n "$AORTD_PID" ]; then
    kill -TERM "$AORTD_PID" 2>/dev/null || true
    pkill -P "$AORTD_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT

request() {
  local name="$1"
  local method="$2"
  local path="$3"
  local output="$OUT_DIR/${name}.json"
  local status_file="$OUT_DIR/${name}.status"
  local code

  code="$(curl -sS -o "$output" -w "%{http_code}" -X "$method" "$BASE_URL$path" || true)"
  printf '%s\n' "$code" >"$status_file"
  if [ "$code" -lt 200 ] || [ "$code" -ge 300 ]; then
    echo "request $method $path failed with HTTP $code" >&2
    cat "$output" >&2 || true
    return 1
  fi
}

write_capsule_snapshot() {
  python3 - "$OUT_DIR/capsules.json" "$OUT_DIR/multi_agent_capsules.json" "$OUT_DIR/selected_agents.env" <<'PY'
import json
import os
import pathlib
import shlex
import sys

capsules_path, output_path, env_path = sys.argv[1:]
capsules = json.loads(pathlib.Path(capsules_path).read_text(encoding="utf-8"))
real = [
    item for item in capsules
    if item.get("capsule_mode") == "real"
    and item.get("real_cgroup_v2") is True
    and int(item.get("pid") or 0) > 0
    and str(item.get("cgroup_path") or "").startswith("/sys/fs/cgroup/")
]
if len(real) < 3:
    raise SystemExit(f"expected at least 3 real cgroup v2 capsules, got {len(real)}")

records = []
for item in real:
    cgroup = pathlib.Path(item["cgroup_path"])
    def read(name):
        path = cgroup / name
        return path.read_text(encoding="utf-8").strip()
    records.append({
        "agent_id": item["agent_id"],
        "worker_pid": item["pid"],
        "cgroup_path": item["cgroup_path"],
        "capsule_mode": item["capsule_mode"],
        "real_cgroup_v2": item["real_cgroup_v2"],
        "memory.max": read("memory.max"),
        "memory.current": read("memory.current"),
        "pids.max": read("pids.max"),
        "pids.current": read("pids.current"),
        "cpu.max": read("cpu.max"),
        "cpu.stat": read("cpu.stat"),
        "cgroup.events": read("cgroup.events"),
    })

pathlib.Path(output_path).write_text(json.dumps({
    "evidence_mode": "real-cgroup-v2",
    "cgroup_fs": "cgroup2fs",
    "agent_count": len(records),
    "capsules": records,
}, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")

with open(env_path, "w", encoding="utf-8") as fh:
    fh.write(f"FREEZE_AGENT={shlex.quote(real[0]['agent_id'])}\n")
    fh.write(f"KILL_AGENT={shlex.quote(real[1]['agent_id'])}\n")
    fh.write("UNAFFECTED_AGENTS=" + shlex.quote(" ".join(item["agent_id"] for item in real[2:])) + "\n")
PY
}

write_action_summaries() {
  python3 - "$OUT_DIR" <<'PY'
import json
import pathlib
import sys

out = pathlib.Path(sys.argv[1])
capsules_before = json.loads((out / "multi_agent_capsules.json").read_text(encoding="utf-8"))
agents_after = json.loads((out / "agents_after_kill.json").read_text(encoding="utf-8"))
capsules_after = json.loads((out / "capsules_after_kill.json").read_text(encoding="utf-8"))
env = {}
for line in (out / "selected_agents.env").read_text(encoding="utf-8").splitlines():
    if "=" in line:
        key, value = line.split("=", 1)
        env[key] = value.strip("'")

freeze_agent = env["FREEZE_AGENT"]
kill_agent = env["KILL_AGENT"]
unaffected = env.get("UNAFFECTED_AGENTS", "").split()

def status(name):
    return (out / f"{name}.status").read_text(encoding="utf-8").strip()

freeze_summary = {
    "evidence_mode": "real-cgroup-v2",
    "agent_id": freeze_agent,
    "freeze_status": status("freeze"),
    "unfreeze_status": status("unfreeze"),
    "success": status("freeze") == "200" and status("unfreeze") == "200",
}
(out / "multi_agent_freeze_unfreeze.json").write_text(
    json.dumps(freeze_summary, indent=2, ensure_ascii=False) + "\n",
    encoding="utf-8",
)

agent_state = {item.get("agent_id") or item.get("id"): item for item in agents_after}
capsule_state = {item.get("agent_id"): item for item in capsules_after}
unaffected_records = []
for agent_id in unaffected:
    capsule = capsule_state.get(agent_id, {})
    unaffected_records.append({
        "agent_id": agent_id,
        "state": agent_state.get(agent_id, {}).get("state", ""),
        "capsule_mode": capsule.get("capsule_mode", ""),
        "pids_current": int(capsule.get("pids_current") or 0),
        "memory_current": int(capsule.get("memory_current") or 0),
    })

kill_summary = {
    "evidence_mode": "real-cgroup-v2",
    "killed_agent_id": kill_agent,
    "kill_status": status("kill"),
    "killed_state": agent_state.get(kill_agent, {}).get("state", ""),
    "unaffected_agents": unaffected_records,
    "other_agents_unaffected": all(item["capsule_mode"] == "real" and item["pids_current"] > 0 for item in unaffected_records),
}
(out / "multi_agent_kill_recovery.json").write_text(
    json.dumps(kill_summary, indent=2, ensure_ascii=False) + "\n",
    encoding="utf-8",
)

summary = {
    "evidence_mode": "real-cgroup-v2",
    "cgroup_fs": "cgroup2fs",
    "capsule_count": capsules_before["agent_count"],
    "freeze_unfreeze": freeze_summary,
    "kill_recovery": kill_summary,
    "success": freeze_summary["success"] and kill_summary["kill_status"] == "200" and kill_summary["other_agents_unaffected"],
}
if not summary["success"]:
    raise SystemExit(json.dumps(summary, indent=2, ensure_ascii=False))
(out / "multi_agent_summary.json").write_text(
    json.dumps(summary, indent=2, ensure_ascii=False) + "\n",
    encoding="utf-8",
)
PY
}

echo "== go test ./... =="
go test ./... >"$OUT_DIR/go_test.txt" 2>&1

echo "== start aortd =="
if command -v setsid >/dev/null 2>&1; then
  setsid go run ./cmd/aortd --config "$CONFIG_PATH" >"$AORTD_LOG" 2>&1 &
  AORTD_PID="$!"
  AORTD_PGID="$AORTD_PID"
else
  go run ./cmd/aortd --config "$CONFIG_PATH" >"$AORTD_LOG" 2>&1 &
  AORTD_PID="$!"
fi

for _ in $(seq 1 30); do
  code="$(curl -sS -o "$OUT_DIR/health.json" -w "%{http_code}" "$BASE_URL/api/health" || true)"
  printf '%s\n' "$code" >"$OUT_DIR/health.status"
  if [ "$code" = "200" ]; then
    break
  fi
  sleep 1
done

if [ "$(cat "$OUT_DIR/health.status")" != "200" ]; then
  echo "aortd did not become healthy" >&2
  tail -100 "$AORTD_LOG" >&2 || true
  exit 1
fi

request demo_run POST /api/demo/run

for _ in $(seq 1 20); do
  request agents GET /api/agents
  request capsules GET /api/capsules
  if write_capsule_snapshot >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
write_capsule_snapshot

# shellcheck disable=SC1090
source "$OUT_DIR/selected_agents.env"
request freeze POST "/api/capsules/$FREEZE_AGENT/freeze"
request unfreeze POST "/api/capsules/$FREEZE_AGENT/unfreeze"
request kill POST "/api/capsules/$KILL_AGENT/kill"
request agents_after_kill GET /api/agents
request capsules_after_kill GET /api/capsules
write_action_summaries

echo "multi-agent cgroup v2 smoke artifacts written to $OUT_DIR"
