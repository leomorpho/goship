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

default_parallel=$(getconf _NPROCESSORS_ONLN 2>/dev/null || echo 2)
if [[ "${default_parallel}" -lt 1 ]]; then
  default_parallel=2
elif [[ "${default_parallel}" -gt 4 ]]; then
  default_parallel=4
fi
parallel_jobs=${INTEGRATION_PARALLEL:-$default_parallel}
if [[ "${parallel_jobs}" -lt 1 ]]; then
  parallel_jobs=1
fi

if [[ "${#packages[@]}" -eq 0 ]]; then
  echo "No integration-tagged test packages found."
  exit 0
fi
run_pkg() {
  pkg="$1"
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
}

if [[ "${parallel_jobs}" -le 1 ]]; then
  for pkg in "${packages[@]}"; do
    run_pkg "${pkg}"
  done
else
  export -f run_pkg
  printf '%s\0' "${packages[@]}" | \
    xargs -0 -n1 -P "${parallel_jobs}" -I{} bash -c 'set -euo pipefail; pkg="$1"; run_pkg "${pkg}"' -- {}
fi

echo "Integration test package set passed."
