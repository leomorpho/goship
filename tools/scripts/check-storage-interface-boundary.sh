#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

violations="$(rg -n '^\s*"github\.com/leomorpho/goship/db/gen' framework/repos/storage -g '*.go' || true)"

if [[ -n "${violations}" ]]; then
  echo "Error: storage interface boundary leaked DB/query package imports:"
  echo "${violations}"
  exit 1
fi

echo "storage interface boundary check passed."
