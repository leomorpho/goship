#!/usr/bin/env bash
set -euo pipefail

if [[ "${UPDATE_GENERATOR_SNAPSHOTS:-0}" == "1" ]]; then
  echo "Updating generator snapshots before running the contract gate..."
fi

go test ./tools/cli/ship/internal/generators -run 'TestGeneratorOutputSnapshotContract$' -count=1
