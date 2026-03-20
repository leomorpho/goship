#!/usr/bin/env bash
set -euo pipefail

bash tools/scripts/generate-llm-txt.sh

git add LLM.txt
