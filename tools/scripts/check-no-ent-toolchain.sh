#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

fail=0

for p in "db/ent" "db/schema"; do
  if [[ -e "$p" ]]; then
    echo "ERROR: legacy Ent artifact path exists: $p"
    fail=1
  fi
done

if rg -n '^entgo\.io/ent\s' go.mod go.work 2>/dev/null; then
  echo "ERROR: Ent toolchain dependency found in go.mod/go.work"
  fail=1
fi

if rg -n '^\s*"entgo\.io/ent' \
  app framework modules cmd tools/cli/ship/internal \
  -g '*.go' \
  -g '!**/*_test.go' \
  -g '!**/tools/scripts/**'; then
  echo "ERROR: runtime/source import of entgo.io/ent detected"
  fail=1
fi

if [[ "$fail" -ne 0 ]]; then
  exit 1
fi

echo "no-ent toolchain check passed."
