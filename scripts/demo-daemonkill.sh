#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${BASE_URL:-http://127.0.0.1:8080}"
SERVICE_NAME="${SERVICE_NAME:-aortd}"
PATTERN="${AORTD_PATTERN:-aortd.*--config}"

pid="$(pgrep -f "$PATTERN" | head -n 1 || true)"
if [[ -z "$pid" ]]; then
  echo "aortd process not found with pattern: $PATTERN" >&2
  exit 1
fi

echo "killing aortd pid=${pid}"
sudo kill -9 "$pid"
sleep 3

if command -v systemctl >/dev/null 2>&1; then
  systemctl status "$SERVICE_NAME" --no-pager || true
fi

echo
echo "recovery status:"
curl -s "$BASE_URL/api/recovery/status"
echo
echo
echo "tasks:"
curl -s "$BASE_URL/api/tasks"
echo
