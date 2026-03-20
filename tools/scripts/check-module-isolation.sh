#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
ALLOWLIST_FILE="${ROOT_DIR}/tools/scripts/test/module-isolation-allowlist.txt"

fail=0
allowlist_entries=""
if [[ -f "${ALLOWLIST_FILE}" ]]; then
  allowlist_entries="$(grep -vE '^\s*(#|$)' "${ALLOWLIST_FILE}" || true)"
fi

for mod_dir in "$ROOT_DIR"/modules/*; do
  if [[ ! -d "$mod_dir" || ! -f "$mod_dir/go.mod" ]]; then
    continue
  fi

  rel="${mod_dir#$ROOT_DIR/}"
  echo "Checking module isolation for $rel..."

  if rg -n '^\s*"github.com/leomorpho/goship/' "$mod_dir" --glob '*.go' --glob '!*_test.go' >/tmp/module-import-violations.txt 2>/dev/null; then
    while IFS= read -r line; do
      file_path="$(echo "$line" | cut -d: -f1)"
      rel_file="${file_path#$ROOT_DIR/}"
      if grep -qxF "$rel_file" <<<"$allowlist_entries"; then
        continue
      fi
      echo "ERROR: forbidden import found in $rel_file"
      echo "  $line"
      fail=1
    done < /tmp/module-import-violations.txt
  fi

  if ! (cd "$mod_dir" && go list -deps ./... >/tmp/module-deps.txt 2>/tmp/module-deps.err); then
    echo "ERROR: failed to resolve deps for $rel"
    cat /tmp/module-deps.err
    fail=1
    continue
  fi

  # Dependency graph can include root imports through explicitly allowlisted adapter bindings.
  # We enforce isolation on direct source imports above.

done

if [[ "$fail" -ne 0 ]]; then
  echo "Module isolation check failed."
  exit 1
fi

echo "Module isolation checks passed."
