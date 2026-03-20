package i18n

import (
	"testing"

	"github.com/leomorpho/goship/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitI18n_DisabledModeLeavesServiceUnset(t *testing.T) {
	got := New(&config.Config{
		I18n: config.I18nConfig{
			Enabled:         false,
			DefaultLanguage: "en",
		},
	})
	assert.Nil(t, got)
}

func TestInitI18n_UsesConfiguredDefaultLanguage(t *testing.T) {
	got := New(&config.Config{
		I18n: config.I18nConfig{
			Enabled:         true,
			DefaultLanguage: "fr",
		},
	})
	require.NotNil(t, got)
	assert.Equal(t, "fr", got.DefaultLanguage())
}
