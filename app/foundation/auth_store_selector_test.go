package foundation

import "testing"

func TestSelectAuthStore_DefaultsToBobWhenDBAvailable(t *testing.T) {
	t.Setenv("PAGODA_AUTH_STORE", "")
	store := selectAuthStore(c.Config, c.ORM, c.Database)
	if _, ok := store.(*bobAuthStore); !ok {
		t.Fatalf("expected bobAuthStore default, got %T", store)
	}
}

func TestSelectAuthStore_UsesBobWhenRequested(t *testing.T) {
	t.Setenv("PAGODA_AUTH_STORE", "bob")
	store := selectAuthStore(c.Config, c.ORM, c.Database)
	if _, ok := store.(*bobAuthStore); !ok {
		t.Fatalf("expected bobAuthStore, got %T", store)
	}
}

func TestSelectAuthStore_UnknownFallsBackToEnt(t *testing.T) {
	t.Setenv("PAGODA_AUTH_STORE", "unknown")
	store := selectAuthStore(c.Config, c.ORM, c.Database)
	if _, ok := store.(*entAuthStore); !ok {
		t.Fatalf("expected entAuthStore fallback, got %T", store)
	}
}

func TestSelectAuthStore_BobFallsBackToEntWhenDBMissing(t *testing.T) {
	t.Setenv("PAGODA_AUTH_STORE", "bob")
	store := selectAuthStore(c.Config, c.ORM, nil)
	if _, ok := store.(*entAuthStore); !ok {
		t.Fatalf("expected entAuthStore fallback without db, got %T", store)
	}
}
