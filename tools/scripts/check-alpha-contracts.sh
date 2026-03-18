#!/usr/bin/env bash
set -euo pipefail

if [[ "${UPDATE_ALPHA_CONTRACTS:-0}" == "1" ]]; then
  echo "Updating alpha contract snapshots before running the gate..."
fi

go test ./tools/cli/ship/internal/commands -run 'Test(AlphaContract_FrozenCommandAndRouteSurface_RedSpec|CIContract_DefinesAlphaFreezeGate_RedSpec)$' -count=1
