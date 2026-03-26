package emailsubscriptions

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

// Contract returns the canonical install contract for the emailsubscriptions battery.
func Contract() InstallContract {
	return InstallContract{
		Routes: []string{
			"modules/notifications/routes/routes.go",
		},
		Config: []string{
			"config/modules.yaml",
			"go.mod",
			"go.work",
		},
		Jobs: []string{
			"modules/emailsubscriptions/service.go",
			"modules/emailsubscriptions/store_sql.go",
			"modules/emailsubscriptions/catalog.go",
		},
		Migrations: []string{
			"modules/emailsubscriptions/db/migrate/migrations",
		},
		Tests: []string{
			"modules/emailsubscriptions/service_test.go",
			"modules/emailsubscriptions/store_sql_test.go",
			"modules/emailsubscriptions/store_sql_integration_test.go",
			"modules/emailsubscriptions/catalog_test.go",
			"modules/notifications/routes/routes_contract_test.go",
		},
	}
}

// New is the module entrypoint used by app wiring.
func New(store Store) *Service {
	return NewService(store)
}
