#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
CONFIG_PATH="${CONFIG_PATH:-configs/openeuler-dev.yaml}"
OUT_DIR="${OUT_DIR:-experiments/results/software_real_demo/openeuler}"
REQUIREMENT="${SOFTWARE_REAL_REQUIREMENT:-实现一个小 Go module，实现字符串工具函数，并运行 go test ./...}"
if [[ "$OUT_DIR" = /* ]]; then
  OUT_ABS="$OUT_DIR"
else
  OUT_ABS="$ROOT_DIR/$OUT_DIR"
fi
RUN_DIR="${RUN_DIR:-$ROOT_DIR/.cache/software-real-openeuler}"
WORKER_BIN="${WORKER_BIN:-$RUN_DIR/aort-worker}"
RUNTIME_CONFIG="${RUNTIME_CONFIG:-$OUT_ABS/openeuler-runtime.yaml}"
DATA_DIR="${DATA_DIR:-$RUN_DIR/data}"
SOCKET_PATH="${SOCKET_PATH:-$RUN_DIR/aortd.sock}"
mkdir -p "$OUT_DIR" "$RUN_DIR" "$DATA_DIR" .cache/go-build
export GOCACHE="$PWD/.cache/go-build"

if [ "$(stat -fc %T /sys/fs/cgroup 2>/dev/null || true)" != "cgroup2fs" ]; then
  echo "live cgroup v2 is required: /sys/fs/cgroup is not cgroup2fs" >&2
  exit 1
fi

if [ "$(id -u)" -ne 0 ]; then
  echo "root is required for live software-real cgroup smoke" >&2
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

validate_result() {
  python3 - "$OUT_DIR/run.json" "$OUT_DIR/software_real_openeuler_summary.json" "$OUT_DIR/software_real_capsules.json" <<'PY'
import json
import pathlib
import sys

result_path, summary_path, capsules_path = sys.argv[1:]
result = json.loads(pathlib.Path(result_path).read_text(encoding="utf-8"))
agents = result.get("agents") or []

if result.get("evidence_mode") != "real-runtime":
    raise SystemExit(f"software-real evidence_mode is not real-runtime: {result.get('evidence_mode')!r}")
if result.get("final_status") != "success":
    raise SystemExit(f"software-real final_status is not success: {result.get('final_status')!r}")
if result.get("first_test_status") != "failed" or result.get("second_test_status") != "passed":
    raise SystemExit("software-real test failure recovery did not run failed -> passed")
if len(agents) < 6:
    raise SystemExit(f"expected at least 6 software-real agents, got {len(agents)}")

capsules = []
for agent in agents:
    agent_id = agent.get("id")
    pid = int(agent.get("pid") or 0)
    cgroup_path = str(agent.get("cgroup_path") or "")
    capsule_mode = agent.get("capsule_mode")
    capsule_evidence_mode = agent.get("capsule_evidence_mode")
    if pid <= 0:
        raise SystemExit(f"{agent_id} has no real worker pid")
    if capsule_mode != "real":
        raise SystemExit(f"{agent_id} capsule_mode is not real: {capsule_mode!r}")
    if capsule_evidence_mode != "real-cgroup-v2":
        raise SystemExit(f"{agent_id} capsule_evidence_mode is not real-cgroup-v2: {capsule_evidence_mode!r}")
    if not cgroup_path.startswith("/sys/fs/cgroup/aort.slice/"):
        raise SystemExit(f"{agent_id} cgroup_path is not under /sys/fs/cgroup/aort.slice: {cgroup_path!r}")
    cgroup = pathlib.Path(cgroup_path)
    if not cgroup.is_dir():
        raise SystemExit(f"{agent_id} cgroup directory missing: {cgroup_path}")

    def read(name):
        path = cgroup / name
        return path.read_text(encoding="utf-8").strip()

    memory_current = int(read("memory.current") or 0)
    pids_current = int(read("pids.current") or 0)
    cpu_stat = read("cpu.stat")
    if pids_current <= 0:
        raise SystemExit(f"{agent_id} pids.current is not positive")
    if not cpu_stat:
        raise SystemExit(f"{agent_id} cpu.stat is empty")

    capsules.append({
        "agent_id": agent_id,
        "role": agent.get("role"),
        "pid": pid,
        "capsule_mode": capsule_mode,
        "capsule_evidence_mode": capsule_evidence_mode,
        "cgroup_path": cgroup_path,
        "memory.current": memory_current,
        "pids.current": pids_current,
        "cpu.stat": cpu_stat,
    })

summary = {
    "evidence_mode": "real-runtime",
    "capsule_evidence_mode": "real-cgroup-v2",
    "cgroup_fs": "cgroup2fs",
    "demo_id": result.get("demo_id"),
    "agent_count": len(agents),
    "worker_cgroup_backed_agents": len(capsules),
    "final_status": result.get("final_status"),
    "first_test_status": result.get("first_test_status"),
    "second_test_status": result.get("second_test_status"),
    "fault_recovered": bool(result.get("fault_recovered")),
    "checkpoint_used": bool(result.get("checkpoint_used")),
    "success": len(capsules) >= 6 and result.get("final_status") == "success",
}
pathlib.Path(summary_path).write_text(json.dumps(summary, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
pathlib.Path(capsules_path).write_text(json.dumps({
    "evidence_mode": "real-cgroup-v2",
    "cgroup_fs": "cgroup2fs",
    "capsules": capsules,
}, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
if not summary["success"]:
    raise SystemExit(json.dumps(summary, indent=2, ensure_ascii=False))
PY
}

echo "== go test ./... =="
go test ./... >"$OUT_DIR/go_test.txt" 2>&1

echo "== build aort-worker =="
go build -o "$WORKER_BIN" ./cmd/aort-worker >"$OUT_DIR/build_worker.txt" 2>&1
chmod +x "$WORKER_BIN"

cat >"$RUNTIME_CONFIG" <<YAML
http_addr: 127.0.0.1:8080
mode: openeuler
data_dir: $DATA_DIR
socket_path: $SOCKET_PATH
worker_command: $WORKER_BIN
heartbeat_timeout_ms: 6000
cgroup_root: /sys/fs/cgroup/aort.slice
YAML

echo "== start aortd =="
if command -v setsid >/dev/null 2>&1; then
  setsid go run ./cmd/aortd --config "$RUNTIME_CONFIG" >"$AORTD_LOG" 2>&1 &
  AORTD_PID="$!"
  AORTD_PGID="$AORTD_PID"
else
  go run ./cmd/aortd --config "$RUNTIME_CONFIG" >"$AORTD_LOG" 2>&1 &
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

python3 - "$OUT_DIR/request.json" "$REQUIREMENT" <<'PY'
import json
import pathlib
import sys

path, requirement = sys.argv[1:]
pathlib.Path(path).write_text(json.dumps({"requirement": requirement}, ensure_ascii=False) + "\n", encoding="utf-8")
PY

code="$(curl -sS -o "$OUT_DIR/run.json" -w "%{http_code}" \
  -H "Content-Type: application/json" \
  -X POST "$BASE_URL/api/demo/software-real/run" \
  --data-binary "@$OUT_DIR/request.json" || true)"
printf '%s\n' "$code" >"$OUT_DIR/run.status"
if [ "$code" -lt 200 ] || [ "$code" -ge 300 ]; then
  echo "software-real run failed with HTTP $code" >&2
  cat "$OUT_DIR/run.json" >&2 || true
  exit 1
fi

request status GET /api/demo/software-real/status
request result GET /api/demo/software-real/result
validate_result

echo "software-real openEuler smoke artifacts written to $OUT_DIR"
