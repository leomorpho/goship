package foundation

import paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"

const (
	defaultFreePlanKey = "free"
	defaultPaidPlanKey = "pro"
)

// BuildSubscriptionPlanCatalog defines the app-owned catalog wired into runtime composition.
func BuildSubscriptionPlanCatalog() (paidsubscriptions.PlanCatalog, error) {
	return paidsubscriptions.NewStaticPlanCatalog(
		[]paidsubscriptions.Plan{
			{Key: defaultFreePlanKey, Paid: false},
			{Key: defaultPaidPlanKey, Paid: true},
		},
		defaultFreePlanKey,
		defaultPaidPlanKey,
	)
}
