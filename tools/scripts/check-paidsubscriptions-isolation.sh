#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
TARGET_DIR="${ROOT_DIR}/modules/paidsubscriptions"
ALLOWLIST_FILE="${ROOT_DIR}/tools/scripts/test/module-isolation-allowlist.txt"

if [[ ! -d "${TARGET_DIR}" ]]; then
  echo "paidsubscriptions module directory not found: ${TARGET_DIR}"
  exit 1
fi

allowlist_entries=""
if [[ -f "${ALLOWLIST_FILE}" ]]; then
  allowlist_entries="$(grep -vE '^\s*(#|$)' "${ALLOWLIST_FILE}" || true)"
fi

if rg -n '^\s*"github.com/leomorpho/goship/v2/' "${TARGET_DIR}" --glob '*.go' >/tmp/paidsubscriptions-isolation-violations.txt 2>/dev/null; then
  fail=0
  while IFS= read -r line; do
    file_path="$(echo "$line" | cut -d: -f1)"
    rel_file="${file_path#$ROOT_DIR/}"
    if grep -qxF "$rel_file" <<<"$allowlist_entries"; then
      continue
    fi
    if [[ "$fail" -eq 0 ]]; then
      echo "ERROR: paidsubscriptions module is not fully isolated."
      fail=1
    fi
    echo "$line"
  done < /tmp/paidsubscriptions-isolation-violations.txt

  if [[ "$fail" -ne 0 ]]; then
    exit 1
  fi
fi

echo "paidsubscriptions isolation check passed."
