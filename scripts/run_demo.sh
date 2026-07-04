#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if [ -f .env.local ]; then
  set -a
  . ./.env.local
  set +a
fi

mkdir -p .cache/go-build
export GOCACHE="$PWD/.cache/go-build"

go test ./...
go run ./cmd/aortd --config configs/dev.yaml
