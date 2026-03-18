package commands

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCIContract_DefinesDedicatedIsolationAndPortabilitySuites_RedSpec(t *testing.T) {
	root := repoRootFromCommandsTest(t)

	workflow := mustReadText(t, filepath.Join(root, ".github", "workflows", "test.yml"))
	makefile := mustReadText(t, filepath.Join(root, "Makefile"))

	if !strings.Contains(workflow, "\n  module_isolation:\n") {
		t.Fatal("test workflow should define a dedicated module_isolation job")
	}
	if !strings.Contains(workflow, "run: make test-module-isolation") {
		t.Fatal("module isolation CI job should invoke make test-module-isolation")
	}
	if !strings.Contains(workflow, "\n  sql_portability:\n") {
		t.Fatal("test workflow should define a dedicated sql_portability job")
	}
	if !strings.Contains(workflow, "run: make test-sql-portability") {
		t.Fatal("sql portability CI job should invoke make test-sql-portability")
	}
	if !strings.Contains(makefile, ".PHONY: test-sql-portability") {
		t.Fatal("Makefile should expose a canonical test-sql-portability entrypoint for CI")
	}
}
