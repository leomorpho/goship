#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PKG_FILE="${ROOT_DIR}/tools/scripts/test/integration-packages.txt"

mkdir -p "${ROOT_DIR}/.cache/go-build"
export GOCACHE="${ROOT_DIR}/.cache/go-build"

echo "Running integration test package set (may require Docker/infra)..."

while IFS= read -r pkg; do
  [[ -z "${pkg}" || "${pkg}" =~ ^# ]] && continue
  go test -tags=integration "${pkg}"
done < "${PKG_FILE}"

echo "Integration test package set passed."
