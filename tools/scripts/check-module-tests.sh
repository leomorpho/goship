#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"

mkdir -p "${ROOT_DIR}/.cache/go-build"
export GOCACHE="${ROOT_DIR}/.cache/go-build"

echo "Running module unit tests..."

found=0
for mod_dir in "${ROOT_DIR}"/modules/*; do
  if [[ ! -d "${mod_dir}" || ! -f "${mod_dir}/go.mod" ]]; then
    continue
  fi
  found=1
  rel="${mod_dir#${ROOT_DIR}/}"
  echo "-> ${rel}"
  (
    cd "${mod_dir}"
    go test ./...
  )
done

if [[ "${found}" -eq 0 ]]; then
  echo "No modules found under modules/."
fi

echo "Module unit tests passed."
