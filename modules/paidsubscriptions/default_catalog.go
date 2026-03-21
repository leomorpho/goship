package paidsubscriptions

const (
	DefaultFreePlanKey = "free"
	DefaultPaidPlanKey = "pro"
)

// BuildDefaultCatalog defines the canonical starter subscription catalog.
func BuildDefaultCatalog() (PlanCatalog, error) {
	return NewStaticPlanCatalog(
		[]Plan{
			{Key: DefaultFreePlanKey, Paid: false},
			{Key: DefaultPaidPlanKey, Paid: true},
		},
		DefaultFreePlanKey,
		DefaultPaidPlanKey,
	)
}
