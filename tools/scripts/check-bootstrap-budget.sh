#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "${WORK_DIR}"' EXIT

BUDGET_SECONDS="${BOOTSTRAP_BUDGET_SECONDS:-120}"
APP_NAME="bootstrap-budget-demo"
APP_DIR="${WORK_DIR}/${APP_NAME}"
SHIP_BIN="${WORK_DIR}/ship"

echo "== bootstrap budget =="
echo "budget_seconds=${BUDGET_SECONDS}"
echo "work_dir=${WORK_DIR}"

(
	cd "${ROOT_DIR}"
	go build -o "${SHIP_BIN}" ./tools/cli/ship/cmd/ship
)

start_ts="$(date +%s)"

(
	cd "${WORK_DIR}"
	"${SHIP_BIN}" new "${APP_NAME}" --module "example.com/${APP_NAME}" --no-i18n >/dev/null
)

(
	cd "${APP_DIR}"
	go run ./cmd/web >/dev/null
)

elapsed="$(( $(date +%s) - start_ts ))"
echo "elapsed_seconds=${elapsed}"

if (( elapsed > BUDGET_SECONDS )); then
	echo "bootstrap budget exceeded: ship new + go run ./cmd/web took ${elapsed}s (budget ${BUDGET_SECONDS}s)" >&2
	exit 1
fi

echo "bootstrap budget passed: ship new + go run ./cmd/web took ${elapsed}s"
