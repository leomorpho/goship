package notifications

import "testing"

func TestNewRequiresDB(t *testing.T) {
	if _, err := New(RuntimeDeps{}); err == nil {
		t.Fatal("expected notifications module to require DB")
	}
}

func TestModuleIDIsStable(t *testing.T) {
	if ModuleID != "notifications" {
		t.Fatalf("ModuleID = %q, want %q", ModuleID, "notifications")
	}
}
