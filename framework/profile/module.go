package profiles

import (
	"io/fs"

	"github.com/leomorpho/goship/framework/core"
	"github.com/leomorpho/goship/framework/web/ui"
)

type ModuleDeps struct {
	Controller     ui.Controller
	ProfileService *ProfileService
	MaxFileSizeMB  int64
}

const ModuleID = "profile"

type Module struct {
	service *routeService
}

func NewModule(deps ModuleDeps) *Module {
	return &Module{
		service: newRouteService(deps),
	}
}

func (m *Module) ID() string {
	return ModuleID
}

func (m *Module) Migrations() fs.FS {
	return nil
}

func (m *Module) RegisterRoutes(r core.Router) error {
	registerRoutes(r, m.service)
	return nil
}
