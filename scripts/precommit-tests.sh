#!/usr/bin/env bash

set -euo pipefail

echo "Running stateless pre-commit Go tests..."

go test ./config ./pkg/context ./pkg/funcmap ./pkg/htmx ./pkg/repos/msg
go test ./pkg/services -run 'Test(NewContainer|ContainerShutdownNilSafe)$'

echo "Pre-commit test suite passed."

