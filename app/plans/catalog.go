package plans

import paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"

const (
	defaultFreePlanKey = "free"
	defaultPaidPlanKey = "pro"
)

// BuildCatalog defines the app-owned catalog wired into runtime composition.
func BuildCatalog() (paidsubscriptions.PlanCatalog, error) {
	return paidsubscriptions.NewStaticPlanCatalog(
		[]paidsubscriptions.Plan{
			{Key: defaultFreePlanKey, Paid: false},
			{Key: defaultPaidPlanKey, Paid: true},
		},
		defaultFreePlanKey,
		defaultPaidPlanKey,
	)
}
