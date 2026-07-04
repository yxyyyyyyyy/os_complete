#!/usr/bin/env bash
set -uo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${OUT_DIR:-$ROOT_DIR/experiments/results/openeuler_smoke}"
JSON_OUT="${AORT_ENV_JSON:-$OUT_DIR/env_check.json}"

mkdir -p "$OUT_DIR"

failures=0
warnings=0

section() {
  printf '\n== %s ==\n' "$1"
}

pass() {
  printf '[PASS] %s\n' "$1"
}

warn() {
  warnings=$((warnings + 1))
  printf '[WARN] %s\n' "$1"
}

fail() {
  failures=$((failures + 1))
  printf '[FAIL] %s\n' "$1"
}

bool_json() {
  if [ "$1" = "true" ]; then
    printf 'true'
  else
    printf 'false'
  fi
}

section "OS release"
if [ -f /etc/os-release ]; then
  os_release="$(cat /etc/os-release)"
  printf '%s\n' "$os_release"
  pass "/etc/os-release readable"
else
  os_release=""
  fail "/etc/os-release not found"
fi

section "Kernel"
kernel="$(uname -a 2>&1 || true)"
if [ -n "$kernel" ]; then
  printf '%s\n' "$kernel"
  pass "uname available"
else
  fail "uname failed"
fi

section "User"
id_output="$(id 2>&1 || true)"
if [ -n "$id_output" ]; then
  printf '%s\n' "$id_output"
  if [ "$(id -u 2>/dev/null || printf 1)" -eq 0 ]; then
    is_root="true"
    pass "running as root"
  else
    is_root="false"
    fail "not running as root; cgroup write tests may degrade"
  fi
else
  is_root="false"
  fail "id failed"
fi

section "cgroup v2"
cgroup_fs="$(stat -fc %T /sys/fs/cgroup 2>/dev/null || true)"
printf 'stat -fc %%T /sys/fs/cgroup: %s\n' "${cgroup_fs:-<unavailable>}"
if [ "$cgroup_fs" = "cgroup2fs" ]; then
  cgroup_v2="true"
  pass "cgroup v2 filesystem is cgroup2fs"
else
  cgroup_v2="false"
  fail "expected cgroup2fs"
fi

if [ -w /sys/fs/cgroup ]; then
  cgroup_writable="true"
  pass "/sys/fs/cgroup is writable"
else
  cgroup_writable="false"
  fail "/sys/fs/cgroup is not writable"
fi

if [ -d /sys/fs/cgroup/aort.slice ]; then
  aort_slice="exists"
  pass "/sys/fs/cgroup/aort.slice exists"
else
  printf '/sys/fs/cgroup/aort.slice missing; attempting mkdir -p...\n'
  if mkdir -p /sys/fs/cgroup/aort.slice 2>/dev/null; then
    aort_slice="created"
    pass "created /sys/fs/cgroup/aort.slice"
  else
    aort_slice="failed"
    fail "failed to create /sys/fs/cgroup/aort.slice"
  fi
fi

section "overlayfs"
if [ -f /proc/filesystems ]; then
  if grep overlay /proc/filesystems; then
    overlay_supported="true"
    pass "overlayfs listed in /proc/filesystems"
  else
    overlay_supported="false"
    warn "overlayfs not listed in /proc/filesystems"
  fi
else
  overlay_supported="false"
  warn "/proc/filesystems not found"
fi

section "PSI"
if [ -d /proc/pressure ]; then
  psi_available="true"
  psi_files="$(ls /proc/pressure 2>/dev/null || true)"
  printf '%s\n' "$psi_files"
  pass "/proc/pressure available"
else
  psi_available="false"
  psi_files=""
  warn "/proc/pressure not available"
fi

section "Go"
if command -v go >/dev/null 2>&1; then
  go_version="$(go version 2>&1 || true)"
  printf '%s\n' "$go_version"
  pass "go available"
else
  go_version=""
  fail "go not found"
fi

section "Node/npm"
if command -v node >/dev/null 2>&1; then
  node_version="$(node -v 2>&1 || true)"
  printf '%s\n' "$node_version"
  pass "node available"
else
  node_version=""
  warn "node not found; frontend smoke is optional"
fi

if command -v npm >/dev/null 2>&1; then
  npm_version="$(npm -v 2>&1 || true)"
  printf '%s\n' "$npm_version"
  pass "npm available"
else
  npm_version=""
  warn "npm not found; frontend smoke is optional"
fi

section "Summary"
printf 'failures=%d warnings=%d\n' "$failures" "$warnings"

if command -v python3 >/dev/null 2>&1; then
  export os_release kernel id_output cgroup_fs aort_slice psi_files go_version node_version npm_version
  export is_root cgroup_v2 cgroup_writable overlay_supported psi_available failures warnings
  python3 - "$JSON_OUT" <<'PY'
import json
import os
import sys

def flag(name):
    return os.environ.get(name) == "true"

data = {
    "evidence_mode": "real" if flag("cgroup_v2") and flag("cgroup_writable") else "degraded",
    "os_release": os.environ.get("os_release", ""),
    "kernel": os.environ.get("kernel", ""),
    "id": os.environ.get("id_output", ""),
    "is_root": flag("is_root"),
    "cgroup": {
        "fs_type": os.environ.get("cgroup_fs", ""),
        "is_cgroup2fs": flag("cgroup_v2"),
        "writable": flag("cgroup_writable"),
        "aort_slice": os.environ.get("aort_slice", ""),
    },
    "overlayfs": {
        "supported": flag("overlay_supported"),
    },
    "psi": {
        "available": flag("psi_available"),
        "files": [line for line in os.environ.get("psi_files", "").splitlines() if line],
    },
    "go_version": os.environ.get("go_version", ""),
    "node_version": os.environ.get("node_version", ""),
    "npm_version": os.environ.get("npm_version", ""),
    "failures": int(os.environ.get("failures", "0")),
    "warnings": int(os.environ.get("warnings", "0")),
}

with open(sys.argv[1], "w", encoding="utf-8") as fh:
    json.dump(data, fh, indent=2, ensure_ascii=False)
    fh.write("\n")
PY
  printf 'env_check_json=%s\n' "$JSON_OUT"
else
  warn "python3 not found; env_check.json was not written"
fi

if [ "$failures" -ne 0 ]; then
  exit 1
fi
