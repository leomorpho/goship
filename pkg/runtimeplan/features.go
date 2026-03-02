package runtimeplan

// WebFeatures describes runtime-dependent feature exposure for web routing.
type WebFeatures struct {
	EnablePageCache bool
	EnableRealtime  bool
}

// ResolveWebFeatures computes web feature flags from runtime plan and available dependencies.
func ResolveWebFeatures(plan Plan, hasCache, hasNotifier bool) WebFeatures {
	if !plan.RunWeb {
		return WebFeatures{}
	}

	return WebFeatures{
		EnablePageCache: hasCache,
		EnableRealtime:  hasNotifier && plan.Adapters.PubSub != "",
	}
}

