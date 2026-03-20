package authsupport

import (
	"database/sql"
	"os"
	"strings"

	"github.com/leomorpho/goship/config"
)

func SelectStore(cfg *config.Config, db *sql.DB) authStore {
	choice := strings.ToLower(strings.TrimSpace(os.Getenv("PAGODA_AUTH_STORE")))
	switch choice {
	case "", "bob":
		if db == nil {
			return newUnavailableAuthStore()
		}
		return newBobAuthStore(db, cfg.Adapters.DB)
	default:
		if db != nil {
			return newBobAuthStore(db, cfg.Adapters.DB)
		}
		return newUnavailableAuthStore()
	}
}
