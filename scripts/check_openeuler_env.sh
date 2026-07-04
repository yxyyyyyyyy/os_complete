#!/usr/bin/env bash
set -uo pipefail

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

section "OS release"
if [ -f /etc/os-release ]; then
  cat /etc/os-release
  pass "/etc/os-release readable"
else
  fail "/etc/os-release not found"
fi

section "Kernel"
if uname -a; then
  pass "uname available"
else
  fail "uname failed"
fi

section "User"
if id; then
  if [ "$(id -u)" -eq 0 ]; then
    pass "running as root"
  else
    fail "not running as root; cgroup write tests may degrade"
  fi
else
  fail "id failed"
fi

section "cgroup v2"
cgroup_fs="$(stat -fc %T /sys/fs/cgroup 2>/dev/null || true)"
printf 'stat -fc %%T /sys/fs/cgroup: %s\n' "${cgroup_fs:-<unavailable>}"
if [ "$cgroup_fs" = "cgroup2fs" ]; then
  pass "cgroup v2 filesystem is cgroup2fs"
else
  fail "expected cgroup2fs"
fi

if [ -w /sys/fs/cgroup ]; then
  pass "/sys/fs/cgroup is writable"
else
  fail "/sys/fs/cgroup is not writable"
fi

if [ -d /sys/fs/cgroup/aort.slice ]; then
  pass "/sys/fs/cgroup/aort.slice exists"
else
  printf '/sys/fs/cgroup/aort.slice missing; attempting mkdir -p...\n'
  if mkdir -p /sys/fs/cgroup/aort.slice 2>/dev/null; then
    pass "created /sys/fs/cgroup/aort.slice"
  else
    fail "failed to create /sys/fs/cgroup/aort.slice"
  fi
fi

section "overlayfs"
if [ -f /proc/filesystems ]; then
  if grep overlay /proc/filesystems; then
    pass "overlayfs listed in /proc/filesystems"
  else
    warn "overlayfs not listed in /proc/filesystems"
  fi
else
  warn "/proc/filesystems not found"
fi

section "PSI"
if [ -d /proc/pressure ]; then
  ls /proc/pressure
  pass "/proc/pressure available"
else
  warn "/proc/pressure not available"
fi

section "Go"
if command -v go >/dev/null 2>&1; then
  go version
  pass "go available"
else
  fail "go not found"
fi

section "Node/npm"
if command -v node >/dev/null 2>&1; then
  node -v
  pass "node available"
else
  warn "node not found; frontend smoke is optional"
fi

if command -v npm >/dev/null 2>&1; then
  npm -v
  pass "npm available"
else
  warn "npm not found; frontend smoke is optional"
fi

section "Summary"
printf 'failures=%d warnings=%d\n' "$failures" "$warnings"

if [ "$failures" -ne 0 ]; then
  exit 1
fi
