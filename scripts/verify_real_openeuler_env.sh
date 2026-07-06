#!/usr/bin/env bash
set -euo pipefail

OUT="${AORT_REAL_ENV_OUT:-experiments/results/real_env/real_openeuler_env.json}"
mkdir -p "$(dirname "$OUT")"

openEuler=false
cgroup2fs=false
root=false
cgroup_writable=false
memory_current_readable=false
pids_current_readable=false
cpu_stat_readable=false
cgroup_kill_supported=false
overlayfs_available=false
overlayfs_mount_success=false
failure_reasons=()

SUDO=()
if [ "$(id -u)" -eq 0 ]; then
  root=true
elif command -v sudo >/dev/null 2>&1 && sudo -n true >/dev/null 2>&1; then
  root=true
  SUDO=(sudo -n)
else
  failure_reasons+=("no root permission")
fi

if [ -r /etc/os-release ] && grep -Eqi '^(ID|NAME)="?openEuler"?' /etc/os-release; then
  openEuler=true
else
  failure_reasons+=("not openEuler")
fi

cgroup_fs="$(stat -fc %T /sys/fs/cgroup 2>/dev/null || true)"
if [ "$cgroup_fs" = "cgroup2fs" ]; then
  cgroup2fs=true
else
  failure_reasons+=("cgroup fs is not cgroup2fs")
fi

test_cgroup="/sys/fs/cgroup/aort-real-env-$$"
sleep_pid=""
cleanup() {
  set +e
  if [ -n "${sleep_pid:-}" ] && kill -0 "$sleep_pid" >/dev/null 2>&1; then
    kill "$sleep_pid" >/dev/null 2>&1
    wait "$sleep_pid" >/dev/null 2>&1
  fi
  if [ -d "$test_cgroup" ]; then
    "${SUDO[@]}" rmdir "$test_cgroup" >/dev/null 2>&1
  fi
  if [ -n "${overlay_root:-}" ] && [ -d "${overlay_root:-}" ]; then
    if grep -q " ${overlay_root}/merged " /proc/self/mountinfo 2>/dev/null; then
      "${SUDO[@]}" umount "${overlay_root}/merged" >/dev/null 2>&1
    fi
    rm -rf "$overlay_root"
  fi
}
trap cleanup EXIT

if $root && $cgroup2fs; then
  if "${SUDO[@]}" mkdir "$test_cgroup" >/dev/null 2>&1; then
    cgroup_writable=true
    sleep 30 &
    sleep_pid="$!"
    disown "$sleep_pid" 2>/dev/null || true
    if printf '%s\n' "$sleep_pid" | "${SUDO[@]}" tee "$test_cgroup/cgroup.procs" >/dev/null 2>&1; then
      :
    else
      failure_reasons+=("cannot write cgroup.procs")
    fi
    if "${SUDO[@]}" test -r "$test_cgroup/memory.current"; then
      memory_current_readable=true
    else
      failure_reasons+=("memory.current not readable")
    fi
    if "${SUDO[@]}" test -r "$test_cgroup/pids.current"; then
      pids_current_readable=true
    else
      failure_reasons+=("pids.current not readable")
    fi
    if "${SUDO[@]}" test -r "$test_cgroup/cpu.stat"; then
      cpu_stat_readable=true
    else
      failure_reasons+=("cpu.stat not readable")
    fi
    if "${SUDO[@]}" test -e "$test_cgroup/cgroup.kill" && printf '1\n' | "${SUDO[@]}" tee "$test_cgroup/cgroup.kill" >/dev/null 2>&1; then
      cgroup_kill_supported=true
      for _ in 1 2 3 4 5 6 7 8 9 10; do
        if ! kill -0 "$sleep_pid" >/dev/null 2>&1; then
          break
        fi
        sleep 0.1
      done
      sleep_pid=""
    else
      failure_reasons+=("cgroup.kill unsupported")
    fi
  else
    failure_reasons+=("cgroup path is not writable")
  fi
elif ! $cgroup2fs; then
  failure_reasons+=("cgroup path is not writable")
fi

if grep -qw overlay /proc/filesystems 2>/dev/null; then
  overlayfs_available=true
else
  failure_reasons+=("overlayfs not listed")
fi

overlay_root="$(mktemp -d /tmp/aort-real-overlay.XXXXXX)"
mkdir -p "$overlay_root/lower" "$overlay_root/upper" "$overlay_root/work" "$overlay_root/merged"
printf 'overlay lower evidence\n' > "$overlay_root/lower/evidence.txt"
if $root && $overlayfs_available; then
  if "${SUDO[@]}" mount -t overlay overlay -o "lowerdir=$overlay_root/lower,upperdir=$overlay_root/upper,workdir=$overlay_root/work" "$overlay_root/merged" >/dev/null 2>&1; then
    if grep -q " $overlay_root/merged " /proc/self/mountinfo 2>/dev/null; then
      overlayfs_mount_success=true
    else
      failure_reasons+=("overlayfs mount failed")
    fi
    "${SUDO[@]}" umount "$overlay_root/merged" >/dev/null 2>&1 || true
  else
    failure_reasons+=("overlayfs mount failed")
  fi
fi

python3 - "$OUT" \
  "$openEuler" "$cgroup2fs" "$root" "$cgroup_writable" \
  "$memory_current_readable" "$pids_current_readable" "$cpu_stat_readable" \
  "$cgroup_kill_supported" "$overlayfs_available" "$overlayfs_mount_success" \
  "${failure_reasons[@]}" <<'PY'
import json
import sys

out = sys.argv[1]
keys = [
    "openEuler",
    "cgroup2fs",
    "root",
    "cgroup_writable",
    "memory_current_readable",
    "pids_current_readable",
    "cpu_stat_readable",
    "cgroup_kill_supported",
    "overlayfs_available",
    "overlayfs_mount_success",
]
values = {key: (sys.argv[idx + 2] == "true") for idx, key in enumerate(keys)}
values["evidence_mode"] = "real-openeuler"
if len(sys.argv) > 12:
    values["failure_reasons"] = sys.argv[12:]
with open(out, "w", encoding="utf-8") as fh:
    json.dump(values, fh, indent=2, ensure_ascii=False)
    fh.write("\n")
PY

if [ "${#failure_reasons[@]}" -gt 0 ]; then
  printf 'real openEuler environment check failed:\n' >&2
  for reason in "${failure_reasons[@]}"; do
    printf -- '- %s\n' "$reason" >&2
  done
  exit 1
fi

printf 'real openEuler environment check passed: %s\n' "$OUT"
