package emailsubscriptions

const ModuleID = "emailsubscriptions"

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

// Contract returns the canonical install contract for the email subscriptions battery.
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
			"modules/emailsubscriptions/module.go",
		},
		Tests: []string{
			"modules/emailsubscriptions/module_test.go",
		},
	}
}
