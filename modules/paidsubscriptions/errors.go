package paidsubscriptions

import "fmt"

func ErrPlanNotFound(planKey string) error {
	return fmt.Errorf("subscription plan %q not found in catalog", normalizePlanKey(planKey))
}
