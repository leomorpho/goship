package subscriptions

import paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"

const (
	PlanFreeKey = "free"
	PlanProKey  = "pro"
)

// BuildPlanCatalog defines app-owned subscription plans wired into the module.
// Keep this in app scope so projects can customize plans without changing module internals.
func BuildPlanCatalog() (paidsubscriptions.PlanCatalog, error) {
	return paidsubscriptions.NewStaticPlanCatalog(
		[]paidsubscriptions.Plan{
			{Key: PlanFreeKey, Paid: false},
			{Key: PlanProKey, Paid: true},
		},
		PlanFreeKey,
		PlanProKey, // default onboarding trial plan
	)
}
