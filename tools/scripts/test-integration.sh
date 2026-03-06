#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

mkdir -p "${ROOT_DIR}/.cache/go-build"
export GOCACHE="${ROOT_DIR}/.cache/go-build"

echo "Running integration test package set (may require Docker/infra)..."

packages=()
while IFS= read -r pkg; do
  packages+=("$pkg")
done < <(
  cd "${ROOT_DIR}" && \
  rg -l '^//go:build integration' \
    --glob '**/*_test.go' \
    --glob '!**/node_modules/**' \
    --glob '!**/.git/**' \
    --glob '!**/.cache/**' \
  | xargs -I{} dirname "{}" \
  | sort -u \
  | sed 's#^#./#'
)

if [[ "${#packages[@]}" -eq 0 ]]; then
  echo "No integration-tagged test packages found."
  exit 0
fi

for pkg in "${packages[@]}"; do
  start_ts="$(date '+%Y-%m-%d %H:%M:%S')"
  echo "[${start_ts}] START ${pkg}"
  if go test -v -tags=integration "${pkg}"; then
    end_ts="$(date '+%Y-%m-%d %H:%M:%S')"
    echo "[${end_ts}] PASS  ${pkg}"
  else
    end_ts="$(date '+%Y-%m-%d %H:%M:%S')"
    echo "[${end_ts}] FAIL  ${pkg}"
    exit 1
  fi
done

echo "Integration test package set passed."
