package pwa

import "io/fs"

const ModuleID = "pwa"

type Module struct {
	service *RouteService
	assets  *assetService
}

func NewModule(service *RouteService) *Module {
	return &Module{
		service: service,
		assets:  newAssetService(),
	}
}

func (m *Module) ID() string {
	return ModuleID
}

func (m *Module) Migrations() fs.FS {
	return nil
}
