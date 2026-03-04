#!/usr/bin/env bash

set -euo pipefail

export GOCACHE="${GOCACHE:-$(pwd)/.cache/go-build}"

echo "Running stateless pre-commit Go tests..."
bash tools/scripts/test-unit.sh
bash tools/scripts/check-compile.sh
bash tools/scripts/check-module-isolation.sh
go run ./tools/cli/ship/cmd/ship agent:check
go run ./tools/cli/ship/cmd/ship doctor

echo "Pre-commit test suite passed."
