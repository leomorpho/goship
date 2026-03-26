package paidsubscriptions

const ModuleID = "paidsubscriptions"

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

// Contract returns the canonical install contract for the paidsubscriptions battery.
func Contract() InstallContract {
	return InstallContract{
		Routes: []string{
			"modules/paidsubscriptions/routes/routes.go",
		},
		Config: []string{
			"config/modules.yaml",
			"go.mod",
			"go.work",
			".env.example",
		},
		Jobs: []string{
			"modules/paidsubscriptions/service.go",
			"modules/paidsubscriptions/store_sql.go",
			"modules/paidsubscriptions/plan_catalog.go",
		},
		Migrations: []string{
			"modules/paidsubscriptions/db/migrate/migrations",
		},
		Tests: []string{
			"modules/paidsubscriptions/service_test.go",
			"modules/paidsubscriptions/store_sql_test.go",
			"modules/paidsubscriptions/store_sql_integration_test.go",
		},
	}
}

// New is the module entrypoint used by app wiring.
func New(store Store) *Service {
	return NewService(store)
}
