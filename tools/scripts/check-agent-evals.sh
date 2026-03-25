#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

go run ./tools/cli/ship/cmd/agent-eval \
  --pack tools/cli/ship/internal/agenteval/testdata/cold_start_task_pack.json \
  --attempts tools/cli/ship/internal/agenteval/testdata/cold_start_attempts_baseline.json \
  --threshold 0.66 \
  --out artifacts/agent-eval-report.json
