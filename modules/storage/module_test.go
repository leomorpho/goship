package storage

import "testing"

func TestContractIncludesCanonicalConfigSurface(t *testing.T) {
	contract := Contract()
	if len(contract.Config) == 0 {
		t.Fatal("expected config ownership entries")
	}
	if len(contract.Tests) == 0 {
		t.Fatal("expected test ownership entries")
	}
}

func TestNewReturnsModule(t *testing.T) {
	if New() == nil {
		t.Fatal("expected non-nil module")
	}
}
