package controllers

import (
	paidsubscriptions "github.com/leomorpho/goship-modules/paidsubscriptions"
)

func activePlanKey(service *paidsubscriptions.Service, pt *paidsubscriptions.ProductType) string {
	return service.ActivePlanKey(pt)
}

func isPaidPlanKey(service *paidsubscriptions.Service, planKey string) bool {
	return service.IsPaidPlanKey(planKey)
}
