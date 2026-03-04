#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PKG_FILE="${ROOT_DIR}/tools/scripts/test/compile-packages.txt"

mkdir -p "${ROOT_DIR}/.cache/go-build"
export GOCACHE="${ROOT_DIR}/.cache/go-build"

echo "Running compile checks (no test execution)..."

while IFS= read -r pkg; do
  [[ -z "${pkg}" || "${pkg}" =~ ^# ]] && continue
  go test -run '^$' "${pkg}"
done < "${PKG_FILE}"

# Compile controller tests without executing TestMain/httptest server startup.
go test -c ./apps/site/web/controllers
rm -f "${ROOT_DIR}/routes.test"

echo "Compile checks passed."
