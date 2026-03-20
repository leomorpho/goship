package authsupport

import (
	"database/sql"
	"testing"

	"github.com/leomorpho/goship/config"
	_ "modernc.org/sqlite"
)

func testConfig() *config.Config {
	return &config.Config{
		Adapters: config.AdaptersConfig{
			DB: "sqlite",
		},
	}
}

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestSelectAuthStore_DefaultsToBobWhenDBAvailable(t *testing.T) {
	t.Setenv("PAGODA_AUTH_STORE", "")
	store := SelectStore(testConfig(), testDB(t))
	if _, ok := store.(*bobAuthStore); !ok {
		t.Fatalf("expected bobAuthStore default, got %T", store)
	}
}

func TestSelectAuthStore_UsesBobWhenRequested(t *testing.T) {
	t.Setenv("PAGODA_AUTH_STORE", "bob")
	store := SelectStore(testConfig(), testDB(t))
	if _, ok := store.(*bobAuthStore); !ok {
		t.Fatalf("expected bobAuthStore, got %T", store)
	}
}

func TestSelectAuthStore_UnknownFallsBackToBobWhenDBAvailable(t *testing.T) {
	t.Setenv("PAGODA_AUTH_STORE", "unknown")
	store := SelectStore(testConfig(), testDB(t))
	if _, ok := store.(*bobAuthStore); !ok {
		t.Fatalf("expected bobAuthStore fallback, got %T", store)
	}
}

func TestSelectAuthStore_BobFailsFastWhenDBMissing(t *testing.T) {
	t.Setenv("PAGODA_AUTH_STORE", "bob")
	store := SelectStore(testConfig(), nil)
	if _, ok := store.(*unavailableAuthStore); !ok {
		t.Fatalf("expected unavailableAuthStore without db, got %T", store)
	}
}

func TestSelectAuthStore_UnknownFailsFastWhenDBMissing(t *testing.T) {
	t.Setenv("PAGODA_AUTH_STORE", "unknown")
	store := SelectStore(testConfig(), nil)
	if _, ok := store.(*unavailableAuthStore); !ok {
		t.Fatalf("expected unavailableAuthStore without db, got %T", store)
	}
}
