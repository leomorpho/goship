#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

pattern='go run \./cmd/.+/main\.go([[:space:]]|$)'
procfiles=(Procfile Procfile.dev Procfile.worker)
failed=0

echo "Checking Procfile go run targets..."

for file in "${procfiles[@]}"; do
  [[ -f "${file}" ]] || continue
  if rg -n "${pattern}" "${file}" >/dev/null; then
    echo "Invalid go run target in ${file}:"
    rg -n "${pattern}" "${file}"
    echo "Use package paths (for example: go run ./cmd/worker) instead of main.go file paths."
    failed=1
  fi
done

if [[ "${failed}" -ne 0 ]]; then
  exit 1
fi

echo "Procfile go run targets are valid."
