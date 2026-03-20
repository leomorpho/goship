package foundation

import "testing"

func TestSelectAuthStore_DefaultsToBobWhenDBAvailable(t *testing.T) {
	t.Setenv("PAGODA_AUTH_STORE", "")
	store := selectAuthStore(c.Config, c.Database)
	if _, ok := store.(*bobAuthStore); !ok {
		t.Fatalf("expected bobAuthStore default, got %T", store)
	}
}

func TestSelectAuthStore_UsesBobWhenRequested(t *testing.T) {
	t.Setenv("PAGODA_AUTH_STORE", "bob")
	store := selectAuthStore(c.Config, c.Database)
	if _, ok := store.(*bobAuthStore); !ok {
		t.Fatalf("expected bobAuthStore, got %T", store)
	}
}

func TestSelectAuthStore_UnknownFallsBackToBobWhenDBAvailable(t *testing.T) {
	t.Setenv("PAGODA_AUTH_STORE", "unknown")
	store := selectAuthStore(c.Config, c.Database)
	if _, ok := store.(*bobAuthStore); !ok {
		t.Fatalf("expected bobAuthStore fallback, got %T", store)
	}
}

func TestSelectAuthStore_BobFailsFastWhenDBMissing(t *testing.T) {
	t.Setenv("PAGODA_AUTH_STORE", "bob")
	store := selectAuthStore(c.Config, nil)
	if _, ok := store.(*unavailableAuthStore); !ok {
		t.Fatalf("expected unavailableAuthStore without db, got %T", store)
	}
}

func TestSelectAuthStore_UnknownFailsFastWhenDBMissing(t *testing.T) {
	t.Setenv("PAGODA_AUTH_STORE", "unknown")
	store := selectAuthStore(c.Config, nil)
	if _, ok := store.(*unavailableAuthStore); !ok {
		t.Fatalf("expected unavailableAuthStore without db, got %T", store)
	}
}
