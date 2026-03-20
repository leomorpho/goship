#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

"${SCRIPT_DIR}/up.sh" "${1:-}"

echo "Starting JS-free dev mode (Go processes only)."
echo "Use 'make dev-full' if you want JS/CSS watchers enabled."
overmind start -f Procfile.dev
