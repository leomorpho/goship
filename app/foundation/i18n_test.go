package foundation

import (
	"testing"

	"github.com/leomorpho/goship/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitI18n_DisabledModeLeavesServiceUnset(t *testing.T) {
	c := &Container{
		Config: &config.Config{
			I18n: config.I18nConfig{
				Enabled:         false,
				DefaultLanguage: "en",
			},
		},
	}

	c.initI18n()
	assert.Nil(t, c.I18n)
}

func TestInitI18n_UsesConfiguredDefaultLanguage(t *testing.T) {
	c := &Container{
		Config: &config.Config{
			I18n: config.I18nConfig{
				Enabled:         true,
				DefaultLanguage: "fr",
			},
		},
	}

	c.initI18n()
	require.NotNil(t, c.I18n)
	assert.Equal(t, "fr", c.I18n.DefaultLanguage())
}
