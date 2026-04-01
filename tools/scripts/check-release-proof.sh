#!/usr/bin/env bash

set -euo pipefail

ARTIFACT_DIR="${ARTIFACT_DIR:-artifacts/release-proof}"
mkdir -p "$ARTIFACT_DIR"

echo "== release proof =="

echo "-- default generated app" | tee "$ARTIFACT_DIR/default.log"
go test ./tools/cli/ship/internal/commands -run 'TestFreshApp$|TestFreshAppStartupSmoke$|TestFreshAppNoInfraDefaultPath$|TestFreshAppAuthFlow$' -count=1 \
  2>&1 | tee -a "$ARTIFACT_DIR/default.log"

echo "-- api-only generated app" | tee "$ARTIFACT_DIR/api-only.log"
go test ./tools/cli/ship/internal/commands -run 'TestFreshAppAPI$|TestFreshAppAPIStartupSmoke$' -count=1 \
  2>&1 | tee -a "$ARTIFACT_DIR/api-only.log"

echo "release proof passed"
