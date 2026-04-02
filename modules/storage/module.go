package storage

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
		Tests: []string{
			"modules/storage/module_test.go",
		},
	}
}

// Module is the standalone storage battery entrypoint.
type Module struct{}

// New returns the storage battery module entrypoint.
func New() *Module {
	return &Module{}
}
