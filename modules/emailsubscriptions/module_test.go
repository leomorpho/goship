package emailsubscriptions

import "testing"

func TestContractIncludesCanonicalConfigSurface(t *testing.T) {
	contract := Contract()
	if len(contract.Config) == 0 {
		t.Fatal("expected config ownership entries")
	}
	if len(contract.Routes) == 0 {
		t.Fatal("expected route ownership entries")
	}
}
