#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
OUT_DIR="experiments/results/openeuler_smoke"
mkdir -p "$OUT_DIR" .cache/go-build
export GOCACHE="$PWD/.cache/go-build"

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

AORTD_PID=""
AORTD_PGID=""

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

validate_agents() {
  python3 - "$OUT_DIR/agents.json" "$OUT_DIR/agent.env" "$OUT_DIR/agent_summary.json" <<'PY'
import json
import shlex
import sys

agents_path, env_path, summary_path = sys.argv[1:]
with open(agents_path, "r", encoding="utf-8") as fh:
    agents = json.load(fh)

if not isinstance(agents, list) or not agents:
    raise SystemExit("expected /api/agents to return at least one agent")

selected = None
for agent in agents:
    pid = agent.get("pid")
    try:
        pid_value = int(pid)
    except (TypeError, ValueError):
        pid_value = 0
    if pid_value > 0:
        selected = agent
        break

if selected is None:
    raise SystemExit("no agent has a non-empty pid")

agent_id = selected.get("agent_id") or selected.get("id")
if not agent_id:
    raise SystemExit("selected agent has no agent_id")

cgroup_path = selected.get("cgroup_path")
if not cgroup_path:
    raise SystemExit("selected agent has empty cgroup_path")

mode = selected.get("capsule_mode")
if mode not in ("real", "degraded"):
    raise SystemExit(f"capsule_mode must be real or degraded, got {mode!r}")

if mode == "real":
    for key in ("memory_current", "pids_current"):
        if key not in selected or selected[key] in (None, ""):
            raise SystemExit(f"real capsule agent missing {key}")
        int(selected[key])

summary = {
    "agent_id": agent_id,
    "pid": selected.get("pid"),
    "capsule_mode": mode,
    "cgroup_path": cgroup_path,
    "memory_current": selected.get("memory_current"),
    "pids_current": selected.get("pids_current"),
}
with open(summary_path, "w", encoding="utf-8") as fh:
    json.dump(summary, fh, indent=2, ensure_ascii=False)
    fh.write("\n")
with open(env_path, "w", encoding="utf-8") as fh:
    for key, value in {
        "AGENT_ID": agent_id,
        "CAPSULE_MODE": mode,
        "CGROUP_PATH": cgroup_path,
    }.items():
        fh.write(f"{key}={shlex.quote(str(value))}\n")
PY
}

echo "== openEuler environment check =="
bash scripts/check_openeuler_env.sh | tee "$OUT_DIR/env_check.txt"

echo "== go test ./... =="
go test ./... >"$OUT_DIR/go_test.txt" 2>&1

echo "== start aortd =="
if command -v setsid >/dev/null 2>&1; then
  setsid go run ./cmd/aortd --config configs/dev.yaml >"$OUT_DIR/aortd.log" 2>&1 &
  AORTD_PID="$!"
  AORTD_PGID="$AORTD_PID"
else
  go run ./cmd/aortd --config configs/dev.yaml >"$OUT_DIR/aortd.log" 2>&1 &
  AORTD_PID="$!"
fi

for _ in $(seq 1 30); do
  if [ -n "$AORTD_PID" ] && ! kill -0 "$AORTD_PID" 2>/dev/null; then
    echo "aortd exited before becoming healthy" >&2
    tail -100 "$OUT_DIR/aortd.log" >&2 || true
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
  tail -100 "$OUT_DIR/aortd.log" >&2 || true
  exit 1
fi

echo "== API smoke =="
request health GET /api/health
request demo_run POST /api/demo/run

for _ in $(seq 1 20); do
  request agents GET /api/agents
  if validate_agents >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
validate_agents

# shellcheck disable=SC1090
source "$OUT_DIR/agent.env"

if [ "$CAPSULE_MODE" = "real" ]; then
  cat "$CGROUP_PATH/memory.current" >"$OUT_DIR/cgroup_memory_current.txt"
  cat "$CGROUP_PATH/pids.current" >"$OUT_DIR/cgroup_pids_current.txt"
else
  printf 'capsule_mode=%s; cgroup file checks skipped\n' "$CAPSULE_MODE" >"$OUT_DIR/cgroup_memory_current.txt"
  printf 'capsule_mode=%s; cgroup file checks skipped\n' "$CAPSULE_MODE" >"$OUT_DIR/cgroup_pids_current.txt"
fi

request context_stats GET /api/context/stats
request syscalls GET /api/syscalls
request scheduler_decisions GET /api/scheduler/decisions

request_allow_failure freeze POST "/api/agents/$AGENT_ID/freeze"
request_allow_failure unfreeze POST "/api/agents/$AGENT_ID/unfreeze"
request_allow_failure kill POST "/api/agents/$AGENT_ID/kill"

request fault_tool_timeout POST /api/demo/fault/tool-timeout

python3 - "$OUT_DIR" <<'PY'
import json
import pathlib
import sys

out = pathlib.Path(sys.argv[1])
summary = {}
for status_file in sorted(out.glob("*.status")):
    summary[status_file.stem] = status_file.read_text(encoding="utf-8").strip()
summary["agent"] = json.loads((out / "agent_summary.json").read_text(encoding="utf-8"))
(out / "smoke_summary.json").write_text(json.dumps(summary, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
PY

echo "openEuler smoke artifacts written to $OUT_DIR"
