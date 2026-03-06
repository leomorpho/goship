#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

violations="$(rg -n 'GetImageObjectFromFile\(file \*ent\.|GetImageObjectsFromFiles\(files \[\]\*ent\.' framework/repos/storage || true)"

if [[ -n "${violations}" ]]; then
  echo "Error: Ent types leaked into storage interface boundary:"
  echo "${violations}"
  exit 1
fi

echo "storage interface boundary check passed."
