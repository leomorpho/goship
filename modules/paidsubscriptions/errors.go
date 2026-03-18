package paidsubscriptions

import "fmt"

func ErrPlanNotFound(planKey string) error {
	return fmt.Errorf("subscription plan %q not found in catalog", normalizePlanKey(planKey))
}

func ErrNoDefaultPaidPlan(planKey string) error {
	return fmt.Errorf("default trial plan %q is not paid; configure a paid default for billing activation", normalizePlanKey(planKey))
}
