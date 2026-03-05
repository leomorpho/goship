#!/usr/bin/env bash

set -euo pipefail

export GOCACHE="${GOCACHE:-$(pwd)/.cache/go-build}"

echo "Running stateless pre-commit Go tests..."
bash tools/scripts/test-unit.sh
bash tools/scripts/check-compile.sh
bash tools/scripts/check-module-tests.sh
bash tools/scripts/check-module-isolation.sh
bash tools/scripts/check-paidsubscriptions-isolation.sh
bash tools/scripts/check-jobs-sql-boundary.sh
bash tools/scripts/check-notifications-pubsub-boundary.sh
bash tools/scripts/check-controller-auth-boundary.sh
bash tools/scripts/check-controller-no-ent-imports.sh
bash tools/scripts/check-bobgen-drift.sh
go run ./tools/cli/ship/cmd/ship agent:check
go run ./tools/cli/ship/cmd/ship doctor

echo "Pre-commit test suite passed."
