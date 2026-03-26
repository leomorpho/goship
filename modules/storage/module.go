package storage

import "github.com/leomorpho/goship/framework/core"

const ModuleID = "storage"

// InstallContract declares the expected install ownership surfaces for this battery.
type InstallContract struct {
	Routes     []string
	Config     []string
	Assets     []string
	Jobs       []string
	Templates  []string
	Migrations []string
	Tests      []string
}

// Contract returns the canonical install contract for the storage battery.
func Contract() InstallContract {
	return InstallContract{
		Config: []string{
			"config/modules.yaml",
			"go.mod",
			"go.work",
		},
		Jobs: []string{
			"modules/storage/module.go",
		},
		Migrations: []string{
			"modules/storage/db/migrate/migrations",
		},
		Tests: []string{
			"modules/storage/module_test.go",
		},
	}
}

// Module exposes the installable storage battery around the core blob seam.
type Module struct {
	blob core.BlobStorage
}

// New wires a storage battery against the app-facing blob storage seam.
func New(blob core.BlobStorage) *Module {
	return &Module{blob: blob}
}

// ID returns the canonical module identifier used by ship module commands.
func (m *Module) ID() string {
	return ModuleID
}

// BlobStorage exposes the configured blob storage seam for higher-level consumers.
func (m *Module) BlobStorage() core.BlobStorage {
	if m == nil {
		return nil
	}
	return m.blob
}
