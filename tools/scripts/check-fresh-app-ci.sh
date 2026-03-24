#!/usr/bin/env bash

set -euo pipefail

echo "== fresh-app CI lane =="
echo "running generation + batteries + verify + smoke integration contracts"

go test ./tools/cli/ship/internal/commands -run TestFreshApp -count=1
go test ./framework/web/controllers -count=1

echo "fresh-app CI lane passed"
