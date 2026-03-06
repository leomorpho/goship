#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

bash "${ROOT_DIR}/tools/scripts/generate-llm-txt.sh"

if ! git diff --quiet -- LLM.txt; then
  echo "ERROR: LLM.txt is out of date. Run: bash tools/scripts/generate-llm-txt.sh"
  git --no-pager diff -- LLM.txt
  exit 1
fi

echo "LLM.txt drift check passed."
