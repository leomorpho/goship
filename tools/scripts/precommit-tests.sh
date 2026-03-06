#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd -- "${SCRIPT_DIR}/../.." && pwd)"

cd "${ROOT_DIR}"

export GOCACHE="${GOCACHE:-${ROOT_DIR}/.cache/go-build}"

echo "Running stateless pre-commit Go tests..."
bash "${ROOT_DIR}/tools/scripts/test-unit.sh"
bash "${ROOT_DIR}/tools/scripts/check-compile.sh"
bash "${ROOT_DIR}/tools/scripts/check-module-tests.sh"
bash "${ROOT_DIR}/tools/scripts/check-module-isolation.sh"
bash "${ROOT_DIR}/tools/scripts/check-paidsubscriptions-isolation.sh"
bash "${ROOT_DIR}/tools/scripts/check-jobs-sql-boundary.sh"
bash "${ROOT_DIR}/tools/scripts/check-notifications-pubsub-boundary.sh"
bash "${ROOT_DIR}/tools/scripts/check-storage-interface-boundary.sh"
bash "${ROOT_DIR}/tools/scripts/check-composition-no-legacy-module-wiring.sh"
bash "${ROOT_DIR}/tools/scripts/check-controller-auth-boundary.sh"
bash "${ROOT_DIR}/tools/scripts/check-bobgen-drift.sh"
bash "${ROOT_DIR}/tools/scripts/check-llm-txt-drift.sh"
go run "${ROOT_DIR}/tools/cli/ship/cmd/ship" agent:check
go run "${ROOT_DIR}/tools/cli/ship/cmd/ship" doctor

echo "Pre-commit test suite passed."
