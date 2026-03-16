package flags

import "io/fs"

import dbmigrate "github.com/leomorpho/goship/modules/flags/db/migrate"

type Module struct {
	service *Service
}

func NewModule(service *Service) *Module {
	return &Module{service: service}
}

func (m *Module) ID() string {
	return "flags"
}

func (m *Module) Migrations() fs.FS {
	return dbmigrate.Migrations()
}

func (m *Module) Service() *Service {
	if m == nil {
		return nil
	}
	return m.service
}
