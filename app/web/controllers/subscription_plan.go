package controllers

import (
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
	appsubscriptions "github.com/leomorpho/goship/app/subscriptions"
)

func activePlanKey(pt *paidsubscriptions.ProductType) string {
	if pt == nil || pt.Value == "" {
		return appsubscriptions.PlanFreeKey
	}
	return pt.Value
}

func isPaidPlanKey(planKey string) bool {
	return planKey != appsubscriptions.PlanFreeKey
}
