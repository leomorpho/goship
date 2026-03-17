package health

import (
	"context"
	"testing"
)

type testChecker struct {
	name   string
	result CheckResult
}

func (t testChecker) Name() string {
	return t.name
}

func (t testChecker) Check(context.Context) CheckResult {
	return t.result
}

func TestRegistryRun(t *testing.T) {
	registry := NewRegistry()
	registry.Register(testChecker{name: "db", result: CheckResult{Status: StatusOK}})
	registry.Register(testChecker{name: "cache", result: CheckResult{Status: StatusError, Error: "down"}})

	results, allOK := registry.Run(context.Background())
	if allOK {
		t.Fatal("expected allOK to be false")
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results["db"].Status != StatusOK {
		t.Fatalf("unexpected db status: %s", results["db"].Status)
	}
	if results["cache"].Status != StatusError {
		t.Fatalf("unexpected cache status: %s", results["cache"].Status)
	}
}

func TestRegistryRunNil(t *testing.T) {
	var registry *Registry
	results, allOK := registry.Run(context.Background())
	if !allOK {
		t.Fatal("expected nil registry to be healthy")
	}
	if len(results) != 0 {
		t.Fatalf("expected empty results, got %d", len(results))
	}
}

func TestRegistryRegisterIgnoresNil(t *testing.T) {
	registry := NewRegistry()
	registry.Register(nil)
	results, allOK := registry.Run(context.Background())
	if !allOK {
		t.Fatal("expected no registered checks to be healthy")
	}
	if len(results) != 0 {
		t.Fatalf("expected empty results, got %d", len(results))
	}
}

func TestRegistryRunDefaultsMissingStatusToError(t *testing.T) {
	registry := NewRegistry()
	registry.Register(testChecker{name: "db", result: CheckResult{Error: "missing status"}})
	results, allOK := registry.Run(context.Background())
	if allOK {
		t.Fatal("expected allOK false when status missing")
	}
	if results["db"].Status != StatusError {
		t.Fatalf("expected default error status, got %q", results["db"].Status)
	}
}
