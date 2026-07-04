#!/usr/bin/env bash
set -euo pipefail

echo "== AORT-R environment check =="

if command -v go >/dev/null 2>&1; then
  go version
else
  echo "WARN: go not found"
fi

if command -v node >/dev/null 2>&1; then
  node --version
else
  echo "WARN: node not found"
fi

if command -v npm >/dev/null 2>&1; then
  npm --version
else
  echo "WARN: npm not found"
fi

if [ -f /sys/fs/cgroup/cgroup.controllers ]; then
  echo "cgroup v2: available"
  cat /sys/fs/cgroup/cgroup.controllers
else
  echo "WARN: cgroup v2 controllers not found; capsule mode will be degraded"
fi

if [ -f /proc/filesystems ] && grep -q overlay /proc/filesystems; then
  echo "overlayfs: available"
else
  echo "WARN: overlayfs not detected; workspace rollback should use degraded-copy mode"
fi

if [ "$(id -u)" -eq 0 ]; then
  echo "root: yes"
else
  echo "WARN: not root; cgroup writes may be degraded"
fi
