#!/usr/bin/env bash

set -euo pipefail

echo "Running stateless pre-commit Go tests..."
bash scripts/test-unit.sh
bash scripts/check-compile.sh

echo "Pre-commit test suite passed."
