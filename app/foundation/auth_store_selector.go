package foundation

import (
	"database/sql"
	"os"
	"strings"

	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/db/ent"
)

func selectAuthStore(cfg *config.Config, orm *ent.Client, db *sql.DB) authStore {
	choice := strings.ToLower(strings.TrimSpace(os.Getenv("PAGODA_AUTH_STORE")))
	switch choice {
	case "ent":
		return newEntAuthStore(orm)
	case "", "bob":
		if db == nil {
			return newEntAuthStore(orm)
		}
		return newBobAuthStore(db, cfg.Adapters.DB)
	default:
		return newEntAuthStore(orm)
	}
}
