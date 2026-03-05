#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
TARGET_DIR="${ROOT_DIR}/modules/paidsubscriptions"

if [[ ! -d "${TARGET_DIR}" ]]; then
  echo "paidsubscriptions module directory not found: ${TARGET_DIR}"
  exit 1
fi

if rg -n '^\s*"github.com/leomorpho/goship/' "${TARGET_DIR}" --glob '*.go' >/tmp/paidsubscriptions-isolation-violations.txt 2>/dev/null; then
  echo "ERROR: paidsubscriptions module is not fully isolated."
  cat /tmp/paidsubscriptions-isolation-violations.txt
  exit 1
fi

echo "paidsubscriptions isolation check passed."
