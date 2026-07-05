#!/usr/bin/env bash
set -euo pipefail

OUT_DIR="experiments/results/deepseek_smoke"
mkdir -p "$OUT_DIR"
SUMMARY="$OUT_DIR/summary.json"

PROVIDER="${AORT_LLM_PROVIDER:-deepseek}"
FALLBACK_PROVIDER="${AORT_LLM_FALLBACK_PROVIDER:-mock}"
BASE_URL="${DEEPSEEK_BASE_URL:-https://api.deepseek.com}"
MODEL="${DEEPSEEK_MODEL:-deepseek-v4-flash}"

if [[ -z "${DEEPSEEK_API_KEY:-}" ]]; then
  python3 -c 'import json, os
summary = {
  "provider": "mock",
  "requested_provider": os.environ.get("AORT_LLM_PROVIDER", "deepseek"),
  "fallback": True,
  "fallback_reason": "no_api_key",
  "evidence_mode": "mock",
  "status": "skipped"
}
os.makedirs("experiments/results/deepseek_smoke", exist_ok=True)
open("experiments/results/deepseek_smoke/summary.json", "w", encoding="utf-8").write(json.dumps(summary, indent=2) + "\n")
print("DEEPSEEK_API_KEY is not set; skipped real DeepSeek smoke without failing.")'
  exit 0
fi

REQUEST_JSON="$OUT_DIR/request.redacted.json"
RESPONSE_JSON="$OUT_DIR/response.json"
cat > "$REQUEST_JSON" <<JSON
{
  "model": "$MODEL",
  "messages": [
    {
      "role": "user",
      "content": "Reply with the single word ok."
    }
  ],
  "stream": false
}
JSON

START_MS="$(python3 -c 'import time; print(int(time.time() * 1000))')"
HTTP_CODE="$(curl -sS -o "$RESPONSE_JSON" -w "%{http_code}" \
  -H "Authorization: Bearer ${DEEPSEEK_API_KEY}" \
  -H "Content-Type: application/json" \
  -X POST "${BASE_URL%/}/chat/completions" \
  --data-binary "@$REQUEST_JSON" || printf '000')"
END_MS="$(python3 -c 'import time; print(int(time.time() * 1000))')"

python3 - "$HTTP_CODE" "$START_MS" "$END_MS" "$MODEL" "$PROVIDER" "$FALLBACK_PROVIDER" "$RESPONSE_JSON" "$SUMMARY" <<'PY'
import json
import sys

http_code, start_ms, end_ms, model, provider, fallback_provider, response_path, summary_path = sys.argv[1:]
duration_ms = int(end_ms) - int(start_ms)
try:
    response = json.load(open(response_path, "r", encoding="utf-8"))
except Exception as exc:
    response = {"parse_error": str(exc)}

ok = str(http_code).startswith("2") and bool(response.get("choices"))
usage = response.get("usage") or {}
if ok:
    summary = {
        "provider": "deepseek",
        "model": model,
        "fallback": False,
        "evidence_mode": "real-api",
        "duration_ms": duration_ms,
        "tokens": int(usage.get("total_tokens") or usage.get("prompt_tokens", 0) + usage.get("completion_tokens", 0)),
        "status": "passed"
    }
else:
    summary = {
        "provider": fallback_provider,
        "requested_provider": provider,
        "fallback": True,
        "fallback_reason": "api_error",
        "evidence_mode": "mock",
        "duration_ms": duration_ms,
        "http_code": http_code,
        "status": "fallback"
    }
open(summary_path, "w", encoding="utf-8").write(json.dumps(summary, indent=2) + "\n")
print(json.dumps(summary, indent=2))
PY
