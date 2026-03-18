#!/usr/bin/env bash
set -euo pipefail

go test ./tools/cli/ship/internal/commands -run 'Test(AlphaContract_FrozenCommandAndRouteSurface_RedSpec|RouteGroupContract_RedSpec)$' -count=1
