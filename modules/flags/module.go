package flags

import "io/fs"

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
	return nil
}

func (m *Module) Service() *Service {
	if m == nil {
		return nil
	}
	return m.service
}
