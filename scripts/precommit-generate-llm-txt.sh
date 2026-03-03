#!/usr/bin/env bash
set -euo pipefail

bash scripts/generate-llm-txt.sh

git add LLM.txt
