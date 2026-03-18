#!/usr/bin/env bash
set -euo pipefail

go test ./tools/cli/ship/internal/commands -run '^TestDocsRouteContract_RedSpec$' -count=1
