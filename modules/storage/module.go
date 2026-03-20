package storage

import "github.com/leomorpho/goship/framework/core"

const ModuleID = "storage"

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
