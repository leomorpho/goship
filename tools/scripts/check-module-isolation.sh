#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="${ROOT_DIR:-$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)}"
ALLOWLIST_FILE="${ALLOWLIST_FILE:-${ROOT_DIR}/tools/scripts/test/module-isolation-allowlist.txt}"

fail=0
allowlist_entries=""
violations_file="$(mktemp)"
trap 'rm -f "$violations_file"' EXIT
if [[ -f "${ALLOWLIST_FILE}" ]]; then
  allowlist_entries="$(grep -vE '^\s*(#|$)' "${ALLOWLIST_FILE}" || true)"
fi

for mod_dir in "$ROOT_DIR"/modules/*; do
  if [[ ! -d "$mod_dir" || ! -f "$mod_dir/go.mod" ]]; then
    continue
  fi

  rel="${mod_dir#$ROOT_DIR/}"
  echo "Checking module isolation for $rel..."

  if rg -n '^\s*"github.com/leomorpho/goship/v2/' "$mod_dir" --glob '*.go' --glob '!*_test.go' >/tmp/module-import-violations.txt 2>/dev/null; then
    while IFS= read -r line; do
      file_path="$(echo "$line" | cut -d: -f1)"
      rel_file="${file_path#$ROOT_DIR/}"
      printf '%s\n' "$rel_file" >>"$violations_file"
      if grep -qxF "$rel_file" <<<"$allowlist_entries"; then
        continue
      fi
      echo "ERROR: forbidden import found in module=${rel} file=${rel_file}"
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

if [[ -n "$allowlist_entries" ]]; then
  while IFS= read -r rel_file; do
    [[ -z "$rel_file" ]] && continue
    if ! grep -qxF "$rel_file" "$violations_file"; then
      echo "ERROR: stale allowlist entry: ${rel_file}"
      fail=1
    fi
  done <<<"$allowlist_entries"
fi

if [[ "$fail" -ne 0 ]]; then
  echo "Module isolation check failed."
  exit 1
fi

echo "Module isolation checks passed."
