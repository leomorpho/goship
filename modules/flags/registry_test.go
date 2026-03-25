package flags

import "testing"

func TestFlagRegistry_RegisterLookupAll(t *testing.T) {
	resetRegistryForTest()

	keyA := Register(FlagDefinition{
		Key:         "beta.checkout",
		Description: "Enable beta checkout flow",
		Default:     false,
	})
	keyB := Register(FlagDefinition{
		Key:         "ui.new_nav",
		Description: "Enable new navigation",
		Default:     true,
	})

	if keyA != FlagKey("beta.checkout") {
		t.Fatalf("keyA = %q", keyA)
	}
	if keyB != FlagKey("ui.new_nav") {
		t.Fatalf("keyB = %q", keyB)
	}

	got, ok := Lookup(keyA)
	if !ok {
		t.Fatalf("expected Lookup(%q) to succeed", keyA)
	}
	if got.Description != "Enable beta checkout flow" || got.Default {
		t.Fatalf("lookup(%q) = %+v", keyA, got)
	}

	all := All()
	if len(all) != 2 {
		t.Fatalf("len(All()) = %d, want 2", len(all))
	}
	if all[0].Key != keyA || all[1].Key != keyB {
		t.Fatalf("All() order/values = %+v", all)
	}
}

func TestFlagRegistry_RegisterPanicsOnDuplicateKey(t *testing.T) {
	resetRegistryForTest()

	Register(FlagDefinition{Key: "dup.flag", Description: "first", Default: false})

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected duplicate register panic")
		}
	}()
	Register(FlagDefinition{Key: "dup.flag", Description: "duplicate", Default: true})
}

