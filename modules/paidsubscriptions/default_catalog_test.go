package paidsubscriptions

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildDefaultCatalog(t *testing.T) {
	t.Parallel()

	catalog, err := BuildDefaultCatalog()
	require.NoError(t, err)
	require.NotNil(t, catalog)

	free, ok := catalog.PlanByKey(DefaultFreePlanKey)
	require.True(t, ok)
	require.False(t, free.Paid)

	paid, ok := catalog.PlanByKey(DefaultPaidPlanKey)
	require.True(t, ok)
	require.True(t, paid.Paid)

	require.Equal(t, DefaultFreePlanKey, catalog.FreePlanKey())
	require.Equal(t, DefaultPaidPlanKey, catalog.DefaultTrialPlanKey())
}
