#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

mkdir -p .cache/go-build
export GOCACHE="$PWD/.cache/go-build"

go run ./cmd/aort-experiment --name all --runs "${1:-5}" --out experiments/results

echo "Wrote experiment results to experiments/results"
