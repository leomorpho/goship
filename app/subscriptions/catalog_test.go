package subscriptions

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildPlanCatalog(t *testing.T) {
	t.Parallel()

	catalog, err := BuildPlanCatalog()
	require.NoError(t, err)
	require.NotNil(t, catalog)

	free, ok := catalog.PlanByKey(PlanFreeKey)
	require.True(t, ok)
	require.False(t, free.Paid)

	pro, ok := catalog.PlanByKey(PlanProKey)
	require.True(t, ok)
	require.True(t, pro.Paid)

	require.Equal(t, PlanFreeKey, catalog.FreePlanKey())
	require.Equal(t, PlanProKey, catalog.DefaultTrialPlanKey())
}
