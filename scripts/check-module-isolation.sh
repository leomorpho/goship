#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"

fail=0

for mod_dir in "$ROOT_DIR"/pkg/modules/*; do
  if [[ ! -d "$mod_dir" || ! -f "$mod_dir/go.mod" ]]; then
    continue
  fi

  rel="${mod_dir#$ROOT_DIR/}"
  echo "Checking module isolation for $rel..."

  if rg -n '^\s*"github.com/leomorpho/goship/' "$mod_dir" --glob '*.go' >/tmp/module-import-violations.txt 2>/dev/null; then
    echo "ERROR: forbidden imports found in $rel"
    cat /tmp/module-import-violations.txt
    fail=1
  fi

  if ! (cd "$mod_dir" && go list -deps ./... >/tmp/module-deps.txt 2>/tmp/module-deps.err); then
    echo "ERROR: failed to resolve deps for $rel"
    cat /tmp/module-deps.err
    fail=1
    continue
  fi

  if rg -n '^github.com/leomorpho/goship/' /tmp/module-deps.txt >/tmp/module-dep-violations.txt 2>/dev/null; then
    echo "ERROR: forbidden dependency graph entries found in $rel"
    cat /tmp/module-dep-violations.txt
    fail=1
  fi

done

if [[ "$fail" -ne 0 ]]; then
  echo "Module isolation check failed."
  exit 1
fi

echo "Module isolation checks passed."
