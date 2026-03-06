#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"

violations="$(rg -n '^\s*"github\.com/leomorpho/goship/db/ent' \
  "$ROOT_DIR/app/profile" \
  --glob '*.go' --glob '!*_test.go' || true)"

if [[ -n "$violations" ]]; then
  echo "ERROR: app/profile boundary violated (db/ent import reintroduced):"
  echo "$violations"
  exit 1
fi

echo "app/profile Ent-boundary check passed."
