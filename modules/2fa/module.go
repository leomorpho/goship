package twofa

import (
	"io/fs"

	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/framework/core"
)

type ModuleDeps struct {
	Controller ui.Controller
	Service    *Service
}

type Module struct {
	controller ui.Controller
	service    *Service
}

func NewModule(deps ModuleDeps) *Module {
	return &Module{
		controller: deps.Controller,
		service:    deps.Service,
	}
}

func (m *Module) ID() string {
	return "2fa"
}

func (m *Module) Migrations() fs.FS {
	return nil
}

func (m *Module) RegisterRoutes(r core.Router) error {
	registerRoutes(r, m.controller, m.service)
	return nil
}
