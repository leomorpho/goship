#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"

cd "${ROOT_DIR}"

echo "Checking sql-core-v1 portability contract..."

go test ./config -run 'TestRuntimeMetadata(SQLitePromotionContract|PostgresHasNoPromotionPath)$' -count=1

echo "SQL portability contract passed."
