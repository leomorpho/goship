#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${ROOT_DIR}"

bash tools/scripts/check-llm-txt-drift.sh

go test ./tools/cli/ship/internal/commands -run '^(TestCIContract_DefinesDocDriftGate_RedSpec|TestDocs_ReadmeLandingNarrativeStaysAligned_RedSpec|TestDocs_MCPWorkflowNarrativeStaysFirstClass_RedSpec|TestDocs_FrameworkFirstRuntimeSeamsStayCanonical_RedSpec)$' -count=1
