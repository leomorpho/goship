package jobs

import "testing"

func TestContractIncludesCanonicalConfigSurface(t *testing.T) {
	contract := Contract()
	if len(contract.Config) == 0 {
		t.Fatal("expected config ownership entries")
	}
	if len(contract.Jobs) == 0 {
		t.Fatal("expected jobs ownership entries")
	}
}

func TestConfigValidate_BackliteRequiresSQLDB(t *testing.T) {
	err := (Config{Backend: BackendBacklite}).Validate()
	if err == nil {
		t.Fatal("expected validation error for missing SQL DB")
	}
}
