package foundation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildSubscriptionPlanCatalog(t *testing.T) {
	t.Parallel()

	catalog, err := BuildSubscriptionPlanCatalog()
	require.NoError(t, err)
	require.NotNil(t, catalog)

	free, ok := catalog.PlanByKey("free")
	require.True(t, ok)
	require.False(t, free.Paid)

	paid, ok := catalog.PlanByKey("pro")
	require.True(t, ok)
	require.True(t, paid.Paid)

	require.Equal(t, "free", catalog.FreePlanKey())
	require.Equal(t, "pro", catalog.DefaultTrialPlanKey())
}
