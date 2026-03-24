package bootstrap

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestNewContainer_StartupErrorsNameMissingSecretsConfigAndServices(t *testing.T) {
	t.Run("missing resend secret names env vars", func(t *testing.T) {
		t.Setenv("PAGODA_APP_ENVIRONMENT", "test")
		t.Setenv("PAGODA_DB_PATH", filepath.Join(t.TempDir(), "app.db"))
		t.Setenv("PAGODA_ADAPTERS_CACHE", "otter")
		t.Setenv("PAGODA_MAIL_DRIVER", "resend")
		t.Setenv("PAGODA_MAIL_RESEND_API_KEY", "")
		t.Setenv("PAGODA_MAIL_RESENDAPIKEY", "")

		message := mustPanicMessage(t, func() {
			_ = NewContainer(nil)
		})
		if !strings.Contains(message, "startup mail secret missing") {
			t.Fatalf("panic=%q should name missing mail secret", message)
		}
		if !strings.Contains(message, "PAGODA_MAIL_RESEND_API_KEY") {
			t.Fatalf("panic=%q should reference resend env var", message)
		}
	})

	t.Run("cache startup failure names service and config keys", func(t *testing.T) {
		t.Setenv("PAGODA_APP_ENVIRONMENT", "test")
		t.Setenv("PAGODA_DB_PATH", filepath.Join(t.TempDir(), "app.db"))
		t.Setenv("PAGODA_MAIL_DRIVER", "log")
		t.Setenv("PAGODA_ADAPTERS_CACHE", "redis")
		t.Setenv("PAGODA_CACHE_HOSTNAME", "127.0.0.1")
		t.Setenv("PAGODA_CACHE_PORT", "1")

		message := mustPanicMessage(t, func() {
			_ = NewContainer(nil)
		})
		if !strings.Contains(message, "startup cache service failure") {
			t.Fatalf("panic=%q should name cache service failure", message)
		}
		if !strings.Contains(message, "PAGODA_CACHE_HOSTNAME/PAGODA_CACHE_PORT") {
			t.Fatalf("panic=%q should reference cache config keys", message)
		}
	})

	t.Run("database startup failure names service and config keys", func(t *testing.T) {
		t.Setenv("PAGODA_APP_ENVIRONMENT", "development")
		t.Setenv("PAGODA_ADAPTERS_CACHE", "otter")
		t.Setenv("PAGODA_MAIL_DRIVER", "log")
		t.Setenv("PAGODA_DATABASE_DRIVER", "postgres")
		t.Setenv("PAGODA_DATABASE_HOSTNAME", "127.0.0.1")
		t.Setenv("PAGODA_DATABASE_PORT", "1")
		t.Setenv("PAGODA_DATABASE_USER", "postgres")
		t.Setenv("PAGODA_DATABASE_PASSWORD", "postgres")

		message := mustPanicMessage(t, func() {
			_ = NewContainer(nil)
		})
		if !strings.Contains(message, "startup database service failure") {
			t.Fatalf("panic=%q should name database service failure", message)
		}
		if !strings.Contains(message, "PAGODA_DATABASE_HOSTNAME/PAGODA_DATABASE_PORT") {
			t.Fatalf("panic=%q should reference database host/port config", message)
		}
	})
}

func mustPanicMessage(t *testing.T, fn func()) (panicMessage string) {
	t.Helper()
	defer func() {
		recovered := recover()
		if recovered == nil {
			t.Fatalf("expected panic")
		}
		panicMessage = recovered.(string)
	}()
	fn()
	return ""
}
