package admin

import (
	"database/sql"
	"io/fs"
	"sync"

	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/framework/core"
)

type ModuleDeps struct {
	Controller ui.Controller
	DB         *sql.DB
}

type Module struct {
	controller ui.Controller
	db         *sql.DB
}

var registerDefaultsOnce sync.Once

func New(deps ModuleDeps) *Module {
	registerDefaultsOnce.Do(registerDefaultResources)
	return &Module{
		controller: deps.Controller,
		db:         deps.DB,
	}
}

func (m *Module) ID() string {
	return "admin"
}

func (m *Module) Migrations() fs.FS {
	return nil
}

func (m *Module) RegisterRoutes(r core.Router) error {
	return registerRoutes(r, m.controller, m.db)
}

type User struct {
	ID        int
	Name      string `validate:"required"`
	Email     string `admin:"email" validate:"required"`
	Password  string
	Verified  bool
	CreatedAt string `admin:"readonly"`
	UpdatedAt string `admin:"readonly"`
}

func registerDefaultResources() {
	Register[User](ResourceConfig{
		TableName: "users",
		Sensitive: []string{"Password"},
		ReadOnly:  []string{"ID", "CreatedAt", "UpdatedAt"},
	})
}
