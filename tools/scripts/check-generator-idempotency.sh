#!/usr/bin/env bash
set -euo pipefail

go test ./tools/cli/ship/internal/generators -run 'TestGeneratorIdempotencyMatrix_RedSpec$' -count=1
