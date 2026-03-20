package paidsubscriptions

import (
	"fmt"
	"strings"
)

// Plan describes a subscription plan exposed by the app.
type Plan struct {
	Key  string
	Paid bool
}

// PlanCatalog defines how the app exposes available plans to the module.
type PlanCatalog interface {
	PlanByKey(key string) (Plan, bool)
	FreePlanKey() string
	DefaultTrialPlanKey() string
}

type StaticPlanCatalog struct {
	plans            map[string]Plan
	freePlanKey      string
	defaultTrialPlan string
}

func NewStaticPlanCatalog(plans []Plan, freePlanKey, defaultTrialPlanKey string) (*StaticPlanCatalog, error) {
	if len(plans) == 0 {
		return nil, fmt.Errorf("plans cannot be empty")
	}
	byKey := make(map[string]Plan, len(plans))
	for _, plan := range plans {
		key := normalizePlanKey(plan.Key)
		if key == "" {
			return nil, fmt.Errorf("plan key cannot be empty")
		}
		plan.Key = key
		if _, exists := byKey[key]; exists {
			return nil, fmt.Errorf("duplicate plan key %q", key)
		}
		byKey[key] = plan
	}

	freeKey := normalizePlanKey(freePlanKey)
	if _, ok := byKey[freeKey]; !ok {
		return nil, fmt.Errorf("free plan key %q not found in catalog", freeKey)
	}

	trialKey := normalizePlanKey(defaultTrialPlanKey)
	if trialKey != "" {
		if _, ok := byKey[trialKey]; !ok {
			return nil, fmt.Errorf("default trial plan key %q not found in catalog", trialKey)
		}
	}

	return &StaticPlanCatalog{
		plans:            byKey,
		freePlanKey:      freeKey,
		defaultTrialPlan: trialKey,
	}, nil
}

func (c *StaticPlanCatalog) PlanByKey(key string) (Plan, bool) {
	plan, ok := c.plans[normalizePlanKey(key)]
	return plan, ok
}

func (c *StaticPlanCatalog) FreePlanKey() string {
	return c.freePlanKey
}

func (c *StaticPlanCatalog) DefaultTrialPlanKey() string {
	return c.defaultTrialPlan
}

func mustDefaultPlanCatalog() PlanCatalog {
	catalog, err := NewStaticPlanCatalog(
		[]Plan{
			{Key: ProductTypeFree.Value, Paid: false},
			{Key: ProductTypePro.Value, Paid: true},
		},
		ProductTypeFree.Value,
		ProductTypePro.Value,
	)
	if err != nil {
		panic(err)
	}
	return catalog
}

func normalizePlanKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}
