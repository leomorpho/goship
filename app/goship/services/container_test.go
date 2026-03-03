package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewContainer(t *testing.T) {
	c := NewContainer()
	t.Cleanup(func() {
		require.NoError(t, c.Shutdown())
	})

	assert.NotNil(t, c.Web)
	assert.NotNil(t, c.Config)
	assert.NotNil(t, c.Validator)
	assert.NotNil(t, c.Database)
	assert.NotNil(t, c.ORM)
	assert.NotNil(t, c.Mail)
	assert.NotNil(t, c.Auth)
	assert.Nil(t, c.Cache)
	assert.Nil(t, c.Tasks)
	assert.Nil(t, c.Notifier)
	assert.NotNil(t, c.CoreCache)
	assert.NotNil(t, c.CoreJobs)
	assert.NotNil(t, c.CorePubSub)
	assert.NotEmpty(t, c.Adapters.Selection.DB)
	assert.NotEmpty(t, c.Adapters.Selection.Cache)
	assert.NotEmpty(t, c.Adapters.Selection.Jobs)
	assert.NotEmpty(t, c.Adapters.Selection.PubSub)
}

func TestContainerShutdownNilSafe(t *testing.T) {
	c := &Container{}
	assert.NoError(t, c.Shutdown())
}
