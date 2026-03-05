#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"

violations="$(rg -n "EntClient|github.com/leomorpho/goship/db/ent" \
  "$ROOT_DIR/modules/jobs/config.go" \
  "$ROOT_DIR/modules/jobs/module.go" \
  "$ROOT_DIR/modules/jobs/drivers/sql/client.go" || true)"

if [[ -n "$violations" ]]; then
  echo "ERROR: jobs SQL boundary violated (Ent coupling reintroduced):"
  echo "$violations"
  exit 1
fi

echo "jobs SQL boundary check passed."
