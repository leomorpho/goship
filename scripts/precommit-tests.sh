#!/usr/bin/env bash

set -euo pipefail

echo "Running stateless pre-commit Go tests..."

mkdir -p .cache/go-build
export GOCACHE="$(pwd)/.cache/go-build"

go test ./config ./pkg/context ./pkg/funcmap ./pkg/htmx ./pkg/repos/msg
go test ./pkg/services -run 'Test(NewContainer|ContainerShutdownNilSafe)$'
go test ./pkg/runtimeplan

echo "Pre-commit test suite passed."
