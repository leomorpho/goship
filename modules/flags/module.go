package flags

import "context"
import "io/fs"

import dbmigrate "github.com/leomorpho/goship/modules/flags/db/migrate"

const ModuleID = "flags"

type Module struct {
	service *Service
	syncer  *Syncer
}

func NewModule(service *Service, syncer *Syncer) *Module {
	return &Module{service: service, syncer: syncer}
}

func (m *Module) ID() string {
	return ModuleID
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

func (m *Module) Start(ctx context.Context) error {
	if m == nil || m.syncer == nil {
		return nil
	}
	_, err := m.syncer.Sync(ctx)
	return err
}
