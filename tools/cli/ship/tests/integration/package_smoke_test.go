package integration

import "testing"

// This keeps `go test ./tools/cli/ship/tests/integration` buildable in default mode.
// Full integration coverage remains under `//go:build integration` tests.
func TestIntegrationPackageCompilesWithoutTags(t *testing.T) {}
