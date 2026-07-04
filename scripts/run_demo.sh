#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

mkdir -p .cache/go-build
export GOCACHE="$PWD/.cache/go-build"

go test ./...
go run ./cmd/aortd --config configs/dev.yaml
