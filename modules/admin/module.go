package admin

import (
	"database/sql"
	"io/fs"
	"sync"

	"github.com/leomorpho/goship/framework/web/ui"
	"github.com/leomorpho/goship/framework/core"
	"github.com/leomorpho/goship/modules/auditlog"
	"github.com/leomorpho/goship/modules/flags"
)

type ModuleDeps struct {
	Controller ui.Controller
	DB         *sql.DB
	AuditLogs  *auditlog.Service
	Flags      *flags.Service
}

type Module struct {
	controller ui.Controller
	db         *sql.DB
	auditLogs  *auditlog.Service
	flags      *flags.Service
}

var registerDefaultsOnce sync.Once

func New(deps ModuleDeps) *Module {
	registerDefaultsOnce.Do(registerDefaultResources)
	return &Module{
		controller: deps.Controller,
		db:         deps.DB,
		auditLogs:  deps.AuditLogs,
		flags:      deps.Flags,
	}
}

func (m *Module) ID() string {
	return "admin"
}

func (m *Module) Migrations() fs.FS {
	return nil
}

func (m *Module) RegisterRoutes(r core.Router) error {
	return registerRoutes(r, m.controller, m.db, m.auditLogs, m.flags)
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
