package ai

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewModule_UnsupportedDriverReturnsUnavailableService(t *testing.T) {
	module := NewModule(NewService(NewUnavailableProvider(`unsupported AI driver "openrouter"`), nil), nil)
	require.NotNil(t, module)
	require.NotNil(t, module.Service())

	_, err := module.Service().Complete(context.Background(), Request{
		Messages: []Message{{Role: "user", Content: "hello"}},
	})
	require.ErrorContains(t, err, "unsupported AI driver")
}

func TestNewModule_MissingAPIKeyReturnsUnavailableService(t *testing.T) {
	module := NewModule(NewService(NewUnavailableProvider("missing ANTHROPIC_API_KEY"), nil), nil)
	require.NotNil(t, module)
	require.NotNil(t, module.Service())

	_, err := module.Service().Complete(context.Background(), Request{
		Messages: []Message{{Role: "user", Content: "hello"}},
	})
	require.ErrorContains(t, err, "missing ANTHROPIC_API_KEY")
}
