#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

export GOCACHE="$ROOT/.cache/go-build"
go test ./...
go run ./cmd/aortd --config configs/dev.yaml
