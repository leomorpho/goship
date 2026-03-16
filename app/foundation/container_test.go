package foundation

import (
	"context"
	"testing"

	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/events/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewContainer(t *testing.T) {
	t.Setenv("PAGODA_APP_ENVIRONMENT", string(config.EnvTest))

	c := NewContainer()
	t.Cleanup(func() {
		require.NoError(t, c.Shutdown())
	})

	assert.NotNil(t, c.Web)
	assert.NotNil(t, c.Config)
	assert.NotNil(t, c.Validator)
	assert.NotNil(t, c.Database)
	assert.NotNil(t, c.Mail)
	assert.NotNil(t, c.Auth)
	assert.NotNil(t, c.AI)
	assert.NotNil(t, c.EventBus)
	if c.Adapters.Selection.Cache == "redis" || c.Adapters.Selection.Cache == "otter" || c.Adapters.Selection.Cache == "memory" {
		assert.NotNil(t, c.Cache)
	} else {
		assert.Nil(t, c.Cache)
	}
	assert.Nil(t, c.Notifier)
	assert.NotNil(t, c.CoreCache)
	assert.NotNil(t, c.CoreJobs)
	assert.NotNil(t, c.CoreJobsInspector)
	assert.NotNil(t, c.CorePubSub)
	assert.NotEmpty(t, c.Adapters.Selection.DB)
	assert.NotEmpty(t, c.Adapters.Selection.Cache)
	assert.NotEmpty(t, c.Adapters.Selection.Jobs)
	assert.NotEmpty(t, c.Adapters.Selection.PubSub)
	_, err := c.Database.Exec(`
		INSERT INTO users (id, created_at, updated_at, name, email, password, verified)
		VALUES (1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'tester', 'tester@example.com', 'password', 1)
	`)
	require.NoError(t, err)
	assert.NoError(t, c.EventBus.Publish(context.Background(), types.UserLoggedIn{UserID: 1}))
}

func TestContainerShutdownNilSafe(t *testing.T) {
	c := &Container{}
	assert.NoError(t, c.Shutdown())
}
