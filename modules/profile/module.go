package profiles

import (
	"io/fs"

	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/framework/core"
)

type ModuleDeps struct {
	Controller     ui.Controller
	ProfileService *ProfileService
	MaxFileSizeMB  int64
}

type Module struct {
	service *routeService
}

func NewModule(deps ModuleDeps) *Module {
	return &Module{
		service: newRouteService(deps),
	}
}

func (m *Module) ID() string {
	return "profile"
}

func (m *Module) Migrations() fs.FS {
	return nil
}

func (m *Module) RegisterRoutes(r core.Router) error {
	registerRoutes(r, m.service)
	return nil
}
