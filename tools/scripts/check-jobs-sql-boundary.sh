#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"

violations="$(rg -n "github.com/leomorpho/goship/v2/db/gen|github.com/leomorpho/goship/v2/framework/core" \
  "$ROOT_DIR/modules/jobs/config.go" \
  "$ROOT_DIR/modules/jobs/module.go" \
  "$ROOT_DIR/modules/jobs/drivers/sql/client.go" || true)"

if [[ -n "$violations" ]]; then
  echo "ERROR: jobs SQL boundary violated (module-local SQL path coupled to app/framework internals):"
  echo "$violations"
  exit 1
fi

echo "jobs SQL boundary check passed."
