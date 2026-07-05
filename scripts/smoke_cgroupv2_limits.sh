#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

OUT_DIR="${OUT_DIR:-experiments/results/openeuler_cgroupv2_limits}"
CGROUP_PARENT="${CGROUP_PARENT:-/sys/fs/cgroup/aort.slice}"
mkdir -p "$OUT_DIR"
export OUT_DIR

if [ "$(stat -fc %T /sys/fs/cgroup 2>/dev/null || true)" != "cgroup2fs" ]; then
  echo "cgroup v2 is required: /sys/fs/cgroup is not cgroup2fs" >&2
  exit 1
fi

if [ "$(id -u)" -ne 0 ]; then
  echo "root is required for cgroup v2 limit smoke" >&2
  exit 1
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 is required" >&2
  exit 1
fi

mkdir -p "$CGROUP_PARENT"
for controller in cpu memory pids; do
  if grep -qw "$controller" "$CGROUP_PARENT/cgroup.controllers" 2>/dev/null; then
    echo "+$controller" >"$CGROUP_PARENT/cgroup.subtree_control" 2>/dev/null || true
  fi
done

memory_cg="$CGROUP_PARENT/aort-limit-memory-$$"
pids_cg="$CGROUP_PARENT/aort-limit-pids-$$"
cpu_cg="$CGROUP_PARENT/aort-limit-cpu-$$"

cleanup() {
  for cg in "$memory_cg" "$pids_cg" "$cpu_cg"; do
    if [ -d "$cg" ]; then
      while read -r pid; do
        [ -n "$pid" ] && kill -KILL "$pid" 2>/dev/null || true
      done <"$cg/cgroup.procs"
      rmdir "$cg" 2>/dev/null || true
    fi
  done
}
trap cleanup EXIT

mkdir -p "$memory_cg" "$pids_cg" "$cpu_cg"

echo "== memory.max enforcement =="
echo 20971520 >"$memory_cg/memory.max"
memory_before="$(cat "$memory_cg/memory.events")"
set +e
python3 - "$memory_cg" <<'PY'
import os
import sys
import time

cgroup = sys.argv[1]
with open(os.path.join(cgroup, "cgroup.procs"), "w", encoding="utf-8") as fh:
    fh.write(str(os.getpid()))
chunks = []
for _ in range(128):
    chunks.append(bytearray(1024 * 1024))
    time.sleep(0.01)
print(len(chunks))
PY
memory_exit=$?
set -e
memory_after="$(cat "$memory_cg/memory.events")"

python3 - "$OUT_DIR/memory_limit_enforced.json" "$memory_cg" "$memory_exit" "$memory_before" "$memory_after" <<'PY'
import json
import pathlib
import sys

out, cgroup, exit_code, before, after = sys.argv[1:]

def parse_events(text):
    result = {}
    for line in text.splitlines():
        parts = line.split()
        if len(parts) == 2:
            result[parts[0]] = int(parts[1])
    return result

before_events = parse_events(before)
after_events = parse_events(after)
oom_delta = after_events.get("oom", 0) - before_events.get("oom", 0)
oom_kill_delta = after_events.get("oom_kill", 0) - before_events.get("oom_kill", 0)
data = {
    "evidence_mode": "real-cgroup-v2",
    "cgroup_path": cgroup,
    "memory.max": pathlib.Path(cgroup, "memory.max").read_text(encoding="utf-8").strip(),
    "memory.current": pathlib.Path(cgroup, "memory.current").read_text(encoding="utf-8").strip(),
    "exit_code": int(exit_code),
    "memory.events_before": before_events,
    "memory.events_after": after_events,
    "oom_delta": oom_delta,
    "oom_kill_delta": oom_kill_delta,
    "enforced": int(exit_code) != 0 and (oom_delta > 0 or oom_kill_delta > 0),
}
if not data["enforced"]:
    raise SystemExit(json.dumps(data, indent=2, ensure_ascii=False))
pathlib.Path(out).write_text(json.dumps(data, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
PY

echo "== pids.max enforcement =="
echo 12 >"$pids_cg/pids.max"
python3 - "$OUT_DIR/pids_limit_enforced.json" "$pids_cg" <<'PY'
import json
import os
import pathlib
import subprocess
import sys
import time

out, cgroup = sys.argv[1:]
with open(os.path.join(cgroup, "cgroup.procs"), "w", encoding="utf-8") as fh:
    fh.write(str(os.getpid()))
started = []
errors = []
try:
    for _ in range(30):
        try:
            started.append(subprocess.Popen(["sleep", "3"]))
        except OSError as exc:
            errors.append(str(exc))
            break
    time.sleep(0.2)
finally:
    for proc in started:
        proc.kill()
    for proc in started:
        proc.wait()

events_path = pathlib.Path(cgroup, "pids.events")
events = events_path.read_text(encoding="utf-8") if events_path.exists() else ""
data = {
    "evidence_mode": "real-cgroup-v2",
    "cgroup_path": cgroup,
    "pids.max": pathlib.Path(cgroup, "pids.max").read_text(encoding="utf-8").strip(),
    "pids.current": pathlib.Path(cgroup, "pids.current").read_text(encoding="utf-8").strip(),
    "started_processes": len(started),
    "fork_errors": errors,
    "pids.events": events,
    "enforced": len(errors) > 0 or len(started) < 30,
}
if not data["enforced"]:
    raise SystemExit(json.dumps(data, indent=2, ensure_ascii=False))
pathlib.Path(out).write_text(json.dumps(data, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
PY

echo "== cpu.max / cpu.stat observability =="
echo "10000 100000" >"$cpu_cg/cpu.max"
cpu_before="$(cat "$cpu_cg/cpu.stat")"
python3 - "$cpu_cg" <<'PY'
import os
import sys
import time

cgroup = sys.argv[1]
with open(os.path.join(cgroup, "cgroup.procs"), "w", encoding="utf-8") as fh:
    fh.write(str(os.getpid()))
deadline = time.time() + 1.5
value = 0
while time.time() < deadline:
    value += 1
print(value)
PY
cpu_after="$(cat "$cpu_cg/cpu.stat")"

python3 - "$OUT_DIR/cpu_quota_stat.json" "$cpu_cg" "$cpu_before" "$cpu_after" <<'PY'
import json
import pathlib
import sys

out, cgroup, before, after = sys.argv[1:]

def parse_stat(text):
    result = {}
    for line in text.splitlines():
        parts = line.split()
        if len(parts) == 2:
            result[parts[0]] = int(parts[1])
    return result

before_stat = parse_stat(before)
after_stat = parse_stat(after)
usage_delta = after_stat.get("usage_usec", 0) - before_stat.get("usage_usec", 0)
data = {
    "evidence_mode": "real-cgroup-v2",
    "cgroup_path": cgroup,
    "cpu.max": pathlib.Path(cgroup, "cpu.max").read_text(encoding="utf-8").strip(),
    "cpu.stat_before": before_stat,
    "cpu.stat_after": after_stat,
    "usage_usec_delta": usage_delta,
    "observable": usage_delta > 0,
}
if not data["observable"]:
    raise SystemExit(json.dumps(data, indent=2, ensure_ascii=False))
pathlib.Path(out).write_text(json.dumps(data, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
PY

python3 - "$OUT_DIR" <<'PY'
import json
import pathlib
import sys

out = pathlib.Path(sys.argv[1])
memory = json.loads((out / "memory_limit_enforced.json").read_text(encoding="utf-8"))
pids = json.loads((out / "pids_limit_enforced.json").read_text(encoding="utf-8"))
cpu = json.loads((out / "cpu_quota_stat.json").read_text(encoding="utf-8"))
summary = {
    "evidence_mode": "real-cgroup-v2",
    "cgroup_fs": "cgroup2fs",
    "memory_limit_enforced": memory["enforced"],
    "pids_limit_enforced": pids["enforced"],
    "cpu_quota_observable": cpu["observable"],
    "success": memory["enforced"] and pids["enforced"] and cpu["observable"],
}
if not summary["success"]:
    raise SystemExit(json.dumps(summary, indent=2, ensure_ascii=False))
(out / "limit_summary.json").write_text(json.dumps(summary, indent=2, ensure_ascii=False) + "\n", encoding="utf-8")
PY

echo "cgroup v2 limit smoke artifacts written to $OUT_DIR"
