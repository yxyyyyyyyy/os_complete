#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
CONFIG_PATH="${CONFIG_PATH:-configs/dev.yaml}"
OUT_DIR="${OUT_DIR:-experiments/results/openeuler_smoke}"
mkdir -p "$OUT_DIR" .cache/go-build
export GOCACHE="$PWD/.cache/go-build"
export OUT_DIR
export AORT_ENV_JSON="$OUT_DIR/env_check.json"

if [ -f .env.local ]; then
  set -a
  . ./.env.local
  set +a
fi

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required for openEuler smoke test" >&2
  exit 1
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 is required to validate JSON smoke outputs" >&2
  exit 1
fi

validate_archived_smoke_evidence() {
  python3 - "$OUT_DIR" <<'PY'
import json
import pathlib
import sys

out = pathlib.Path(sys.argv[1])
required = {
    "env_check.json": lambda data: data.get("evidence_mode") == "real-cgroup-v2"
    and data.get("cgroup", {}).get("fs_type") == "cgroup2fs",
    "agent_summary.json": lambda data: data.get("evidence_mode") == "real-cgroup-v2"
    and data.get("capsule_mode") == "real"
    and data.get("real_cgroup_v2") is True
    and int(data.get("memory_current") or 0) > 0
    and int(data.get("pids_current") or 0) > 0
    and bool(data.get("cpu_stat")),
    "capsule_real.json": lambda data: data.get("evidence_mode") == "real-cgroup-v2"
    and data.get("cgroup_fs") == "cgroup2fs"
    and data.get("capsule_mode") == "real",
    "smoke_summary.json": lambda data: data.get("agent", {}).get("real_cgroup_v2") is True
    and data.get("agent", {}).get("capsule_mode") == "real"
    and data.get("capsule_real", {}).get("evidence_mode") == "real-cgroup-v2",
}
for name, check in required.items():
    path = out / name
    if not path.exists():
        raise SystemExit(f"missing archived smoke evidence: {path}")
    data = json.loads(path.read_text(encoding="utf-8"))
    if not check(data):
        raise SystemExit(f"archived smoke evidence failed validation: {path}")

limit = pathlib.Path("experiments/results/openeuler_cgroupv2_limits/limit_summary.json")
multi = pathlib.Path("experiments/results/openeuler_cgroupv2_multi/multi_agent_summary.json")
for path in (limit, multi):
    if not path.exists():
        raise SystemExit(f"missing linked evidence: {path}")
    data = json.loads(path.read_text(encoding="utf-8"))
    if data.get("evidence_mode") != "real-cgroup-v2":
        raise SystemExit(f"linked evidence is not real-cgroup-v2: {path}")
if json.loads(limit.read_text(encoding="utf-8")).get("memory_limit_enforced") is not True:
    raise SystemExit("memory limit evidence is not enforced")
if json.loads(limit.read_text(encoding="utf-8")).get("pids_limit_enforced") is not True:
    raise SystemExit("pids limit evidence is not enforced")
if json.loads(limit.read_text(encoding="utf-8")).get("cpu_quota_observable") is not True:
    raise SystemExit("cpu quota evidence is not observable")
PY
}

if [ "$(stat -fc %T /sys/fs/cgroup 2>/dev/null || true)" != "cgroup2fs" ] && [ "${AORT_REQUIRE_LIVE_OPENEULER:-0}" != "1" ]; then
  echo "== archived openEuler real-cgroup-v2 smoke evidence =="
  if validate_archived_smoke_evidence; then
    echo "current host is not live cgroup2fs; archived real-cgroup-v2 smoke evidence is valid."
    echo "Set AORT_REQUIRE_LIVE_OPENEULER=1 to require live smoke execution."
  else
    echo "current host is not live cgroup2fs and archived smoke evidence is unavailable; writing degraded smoke summary."
    python3 - "$OUT_DIR/smoke_summary.json" <<'PY'
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
path.write_text(json.dumps({
    "evidence_mode": "degraded",
    "fallback_reason": "current host is not cgroup2fs and archived openEuler smoke evidence is unavailable",
    "smoke": "degraded"
}, indent=2) + "\n", encoding="utf-8")
PY
  fi
  exit 0
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

request_allow_failure() {
  local name="$1"
  local method="$2"
  local path="$3"
  local output="$OUT_DIR/${name}.json"
  local status_file="$OUT_DIR/${name}.status"
  local code

  code="$(curl -sS -o "$output" -w "%{http_code}" -X "$method" "$BASE_URL$path" || true)"
  printf '%s\n' "$code" >"$status_file"
}

validate_real_capsule() {
  python3 - "$OUT_DIR/agents.json" "$OUT_DIR/capsules.json" "$OUT_DIR/agent.env" "$OUT_DIR/agent_summary.json" "$OUT_DIR/capsule_real.json" <<'PY'
import json
import pathlib
import shlex
import sys

agents_path, capsules_path, env_path, summary_path, capsule_real_path = sys.argv[1:]
agents = json.loads(pathlib.Path(agents_path).read_text(encoding="utf-8"))
capsules = json.loads(pathlib.Path(capsules_path).read_text(encoding="utf-8"))

if not isinstance(agents, list) or not agents:
    raise SystemExit("expected /api/agents to return at least one agent")
if not isinstance(capsules, list) or not capsules:
    raise SystemExit("expected /api/capsules to return at least one capsule")

selected = None
for agent in agents:
    try:
        if int(agent.get("pid") or 0) > 0:
            selected = agent
            break
    except (TypeError, ValueError):
        pass
if selected is None:
    raise SystemExit("no agent has a non-empty pid")

agent_id = selected.get("agent_id") or selected.get("id")
capsule = next((item for item in capsules if item.get("agent_id") == agent_id), None)
if capsule is None:
    raise SystemExit(f"no capsule record for {agent_id}")

mode = capsule.get("capsule_mode") or selected.get("capsule_mode")
cgroup_path = capsule.get("cgroup_path") or selected.get("cgroup_path")
memory_current = int(capsule.get("memory_current") or selected.get("memory_current") or 0)
pids_current = int(capsule.get("pids_current") or selected.get("pids_current") or 0)
real_cgroup_v2 = bool(capsule.get("real_cgroup_v2"))

summary = {
    "evidence_mode": capsule.get("evidence_mode", "degraded"),
    "real_cgroup_v2": real_cgroup_v2,
    "agent_id": agent_id,
    "pid": selected.get("pid"),
    "capsule_mode": mode,
    "cgroup_path": cgroup_path,
    "memory_current": memory_current,
    "pids_current": pids_current,
    "cpu_stat": capsule.get("cpu_stat") or {},
    "events": capsule.get("events") or {},
    "frozen": bool(capsule.get("frozen")),
    "error": capsule.get("error") or selected.get("error") or "",
}
pathlib.Path(summary_path).write_text(json.dumps(summary, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")

with open(env_path, "w", encoding="utf-8") as fh:
    for key, value in {
        "AGENT_ID": agent_id,
        "CAPSULE_MODE": mode,
        "CGROUP_PATH": cgroup_path,
    }.items():
        fh.write(f"{key}={shlex.quote(str(value))}\n")

if mode != "real" or not real_cgroup_v2:
    capsule_real = {
        "evidence_mode": "degraded",
        "real_cgroup_v2": False,
        "reason": summary["error"] or "capsule_mode is not real",
        "agent": summary,
    }
    pathlib.Path(capsule_real_path).write_text(json.dumps(capsule_real, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
    raise SystemExit("capsule_mode is not real; refusing to generate fake real evidence")

if not cgroup_path or not str(cgroup_path).startswith("/sys/fs/cgroup/"):
    raise SystemExit(f"real capsule has invalid cgroup_path: {cgroup_path!r}")
if memory_current <= 0:
    raise SystemExit("real capsule memory_current must be non-zero")
if pids_current <= 0:
    raise SystemExit("real capsule pids_current must be non-zero")

capsule_real = {
    "evidence_mode": "real-cgroup-v2",
    "os": "openEuler 24.03 LTS",
    "cgroup_fs": "cgroup2fs",
    "capsule_mode": "real",
    "cgroup_path": cgroup_path,
    "memory_current": memory_current,
    "pids_current": pids_current,
    "cpu_stat": summary["cpu_stat"],
    "events": summary["events"],
}
pathlib.Path(capsule_real_path).write_text(json.dumps(capsule_real, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
PY
}

update_action_summary() {
  python3 - "$OUT_DIR" <<'PY'
import json
import pathlib
import sys

out = pathlib.Path(sys.argv[1])
summary_path = out / "agent_summary.json"
summary = json.loads(summary_path.read_text(encoding="utf-8"))
for name in ("freeze", "unfreeze", "kill"):
    status_path = out / f"{name}.status"
    if status_path.exists():
        summary[name] = status_path.read_text(encoding="utf-8").strip()
summary_path.write_text(json.dumps(summary, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")

capsule_real_path = out / "capsule_real.json"
capsule_real = json.loads(capsule_real_path.read_text(encoding="utf-8"))
for name in ("freeze", "unfreeze", "kill"):
    status_path = out / f"{name}.status"
    if status_path.exists():
        capsule_real[name] = status_path.read_text(encoding="utf-8").strip()
capsule_real_path.write_text(json.dumps(capsule_real, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
PY
}

write_summary() {
  python3 - "$OUT_DIR" <<'PY'
import json
import pathlib
import sys

out = pathlib.Path(sys.argv[1])
summary = {}
for status_file in sorted(out.glob("*.status")):
    summary[status_file.stem] = status_file.read_text(encoding="utf-8").strip()
if (out / "agent_summary.json").exists():
    summary["agent"] = json.loads((out / "agent_summary.json").read_text(encoding="utf-8"))
if (out / "capsule_real.json").exists():
    summary["capsule_real"] = json.loads((out / "capsule_real.json").read_text(encoding="utf-8"))
if (out / "env_check.json").exists():
    summary["env_check"] = json.loads((out / "env_check.json").read_text(encoding="utf-8"))
for name in ("manual_smoke_summary.json", "smoke_summary.json"):
    (out / name).write_text(json.dumps(summary, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
PY
}

echo "== openEuler environment check =="
bash scripts/check_openeuler_env.sh | tee "$OUT_DIR/env_check.txt"

echo "== go test ./... =="
go test ./... >"$OUT_DIR/go_test.txt" 2>&1
printf '0\n' >"$OUT_DIR/go_test.status"

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
  if [ -n "$AORTD_PID" ] && ! kill -0 "$AORTD_PID" 2>/dev/null; then
    echo "aortd exited before becoming healthy" >&2
    tail -100 "$AORTD_LOG" >&2 || true
    exit 1
  fi
  code="$(curl -sS -o "$OUT_DIR/health.json" -w "%{http_code}" "$BASE_URL/api/health" || true)"
  printf '%s\n' "$code" >"$OUT_DIR/health.status"
  if [ "$code" = "200" ]; then
    break
  fi
  sleep 1
done

if [ "$(cat "$OUT_DIR/health.status")" != "200" ]; then
  echo "aortd did not become healthy within 30s" >&2
  tail -100 "$AORTD_LOG" >&2 || true
  exit 1
fi

echo "== API smoke =="
request health GET /api/health
request demo_run POST /api/demo/run

for _ in $(seq 1 20); do
  request agents GET /api/agents
  request capsules GET /api/capsules
  if validate_real_capsule >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
validate_real_capsule

# shellcheck disable=SC1090
source "$OUT_DIR/agent.env"

cat "$CGROUP_PATH/memory.current" >"$OUT_DIR/cgroup_memory_current.txt"
cat "$CGROUP_PATH/pids.current" >"$OUT_DIR/cgroup_pids_current.txt"

request context_stats GET /api/context/stats
request syscalls GET /api/syscalls
request scheduler_decisions GET /api/scheduler/decisions

request freeze POST "/api/capsules/$AGENT_ID/freeze"
request unfreeze POST "/api/capsules/$AGENT_ID/unfreeze"
request kill POST "/api/capsules/$AGENT_ID/kill"
update_action_summary

request fault_tool_timeout POST /api/demo/fault/tool-timeout
write_summary

echo "openEuler smoke artifacts written to $OUT_DIR"
