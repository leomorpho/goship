package paidsubscriptions

import "testing"

func TestBuildDefaultCatalogContainsFreeAndPro(t *testing.T) {
	catalog, err := BuildDefaultCatalog()
	if err != nil {
		t.Fatalf("BuildDefaultCatalog() error = %v", err)
	}
	if _, ok := catalog.PlanByKey(DefaultFreePlanKey); !ok {
		t.Fatalf("default catalog missing %q", DefaultFreePlanKey)
	}
	if _, ok := catalog.PlanByKey(DefaultPaidPlanKey); !ok {
		t.Fatalf("default catalog missing %q", DefaultPaidPlanKey)
	}
}

func TestContractIncludesRoutes(t *testing.T) {
	contract := Contract()
	if len(contract.Routes) == 0 {
		t.Fatal("expected route ownership entries")
	}
}
