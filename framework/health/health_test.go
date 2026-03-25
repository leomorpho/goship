package health

import (
	"context"
	"strings"
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

func TestNewRegistryRegistersInitialCheckers(t *testing.T) {
	registry := NewRegistry(
		testChecker{name: "db", result: CheckResult{Status: StatusOK}},
		nil,
		testChecker{name: "cache", result: CheckResult{Status: StatusOK}},
	)

	results, allOK := registry.Run(context.Background())
	if !allOK {
		t.Fatal("expected allOK to be true")
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestRegistryValidateStartupContract(t *testing.T) {
	registry := NewRegistry(
		testChecker{name: "db", result: CheckResult{Status: StatusOK}},
		testChecker{name: "cache", result: CheckResult{Status: StatusOK}},
		testChecker{name: "jobs", result: CheckResult{Status: StatusOK}},
	)

	if err := registry.ValidateStartupContract(); err != nil {
		t.Fatalf("ValidateStartupContract() error = %v", err)
	}
}

func TestRegistryValidateStartupContractMissingChecks(t *testing.T) {
	registry := NewRegistry(
		testChecker{name: "db", result: CheckResult{Status: StatusOK}},
	)

	err := registry.ValidateStartupContract()
	if err == nil {
		t.Fatal("expected validation error when checks are missing")
	}
	if !strings.Contains(err.Error(), "cache") || !strings.Contains(err.Error(), "jobs") {
		t.Fatalf("validation error = %q, want missing cache/jobs", err.Error())
	}
}

func TestRegistryValidateStartupContractNilRegistry(t *testing.T) {
	var registry *Registry
	if err := registry.ValidateStartupContract(); err == nil {
		t.Fatal("expected error for nil registry")
	}
}

func TestRegistryStartupSummary(t *testing.T) {
	registry := NewRegistry(
		testChecker{name: "db", result: CheckResult{Status: StatusOK}},
		testChecker{name: "jobs", result: CheckResult{Status: StatusOK}},
	)

	summary := registry.StartupSummary()
	if summary.Ready {
		t.Fatal("expected startup summary to be not ready when checks are missing")
	}
	if strings.Join(summary.Required, ",") != "db,cache,jobs" {
		t.Fatalf("required = %v, want db,cache,jobs", summary.Required)
	}
	if strings.Join(summary.Registered, ",") != "db,jobs" {
		t.Fatalf("registered = %v, want db,jobs", summary.Registered)
	}
	if strings.Join(summary.Missing, ",") != "cache" {
		t.Fatalf("missing = %v, want cache", summary.Missing)
	}
}

func TestRegistryValidateStartupContractIncludesStructuredSummary(t *testing.T) {
	registry := NewRegistry(testChecker{name: "db", result: CheckResult{Status: StatusOK}})

	err := registry.ValidateStartupContract()
	if err == nil {
		t.Fatal("expected error when startup checks are missing")
	}
	if !strings.Contains(err.Error(), "health startup contract") {
		t.Fatalf("error = %q, want structured startup contract prefix", err.Error())
	}
	if !strings.Contains(err.Error(), "missing=[cache jobs]") {
		t.Fatalf("error = %q, want missing check list", err.Error())
	}
}
