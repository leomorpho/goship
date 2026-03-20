#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

violations="$(rg -n --glob '*.go' --glob '!**/*_test.go' '(appsubscriptions\.NewEntStore|NewEntStore\(|ORM:\s*c\.ORM)' cmd app/web/wiring.go || true)"

if [[ -n "${violations}" ]]; then
  echo "Error: composition root contains legacy ORM-bound module wiring in runtime paths:"
  echo "${violations}"
  exit 1
fi

echo "composition-root legacy ORM wiring check passed."
