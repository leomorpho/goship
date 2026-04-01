#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"

violations="$(rg -n "github.com/leomorpho/goship/v2/framework/core" \
  "$ROOT_DIR/modules/notifications/module.go" \
  "$ROOT_DIR/modules/notifications/notifier.go" \
  "$ROOT_DIR/modules/notifications/notifier_test.go" || true)"

if [[ -n "$violations" ]]; then
  echo "ERROR: notifications pubsub boundary violated (framework/core coupling reintroduced):"
  echo "$violations"
  exit 1
fi

echo "notifications pubsub boundary check passed."
