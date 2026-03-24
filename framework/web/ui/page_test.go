package ui_test

import (
	"net/http"
	"testing"

	echomw "github.com/labstack/echo/v4/middleware"
	"github.com/leomorpho/goship/framework/context"
	"github.com/leomorpho/goship/framework/repos/uxflashmessages"
	"github.com/leomorpho/goship/framework/tests"
	"github.com/leomorpho/goship/framework/web/ui"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPage(t *testing.T) {
	ctx, _ := tests.NewContext(c.Web, "/")
	p := ui.NewPage(ctx)
	assert.Same(t, ctx, p.Context)
	assert.NotNil(t, p.ToURL)
	assert.Equal(t, "/", p.Path)
	assert.Equal(t, "/", p.URL)
	assert.Equal(t, http.StatusOK, p.StatusCode)
	assert.Equal(t, ui.NewPager(ctx, ui.DefaultItemsPerPage), p.Pager)
	assert.Empty(t, p.Headers)
	assert.True(t, p.IsHome)
	assert.False(t, p.IsAuth)
	assert.Empty(t, p.CSRF)
	assert.Empty(t, p.RequestID)
	assert.False(t, p.Cache.Enabled)

	ctx, _ = tests.NewContext(c.Web, "/abc?def=123")
	usr, err := tests.CreateRandomUserDB(c.Database)
	require.NoError(t, err)
	ctx.Set(context.AuthenticatedUserIDKey, usr.ID)
	ctx.Set(context.AuthenticatedUserNameKey, usr.Name)
	ctx.Set(context.AuthenticatedUserEmailKey, usr.Email)
	ctx.Set(echomw.DefaultCSRFConfig.ContextKey, "csrf")
	p = ui.NewPage(ctx)
	assert.Equal(t, "/abc", p.Path)
	assert.Equal(t, "/abc?def=123", p.URL)
	assert.False(t, p.IsHome)
	assert.True(t, p.IsAuth)
	require.NotNil(t, p.AuthUser)
	assert.Equal(t, usr.ID, p.AuthUser.ID)
	assert.Equal(t, usr.Name, p.AuthUser.Name)
	assert.Equal(t, usr.Email, p.AuthUser.Email)
	assert.Equal(t, "csrf", p.CSRF)
}

func TestPage_GetMessages(t *testing.T) {
	ctx, _ := tests.NewContext(c.Web, "/")
	tests.InitSession(ctx)
	p := ui.NewPage(ctx)

	// Set messages
	msgTests := make(map[uxflashmessages.Type][]string)
	msgTests[uxflashmessages.TypeWarning] = []string{
		"abc",
		"def",
	}
	msgTests[uxflashmessages.TypeInfo] = []string{
		"123",
		"456",
	}
	for typ, values := range msgTests {
		for _, value := range values {
			uxflashmessages.Set(ctx, typ, value)
		}
	}

	// Get the messages
	for typ, values := range msgTests {
		msgs := p.GetMessages(typ)

		for i, message := range msgs {
			assert.Equal(t, values[i], string(message))
		}
	}
}

func TestPage_DesignTokenRecipes(t *testing.T) {
	ctx, _ := tests.NewContext(c.Web, "/")
	p := ui.NewPage(ctx)

	assert.Equal(t, "gs-page", p.StarterPageClass())
	assert.Equal(t, "gs-panel", p.StarterPanelClass())
	assert.Equal(t, "gs-title", p.StarterTitleClass())
	assert.Equal(t, "gs-text", p.StarterTextClass())
	assert.Equal(t, "gs-button gs-button-primary", p.StarterPrimaryActionClass())
	assert.Equal(t, "gs-button gs-button-secondary", p.StarterSecondaryActionClass())
	assert.Equal(t, "gs-kicker", p.StarterKickerClass())
	assert.Equal(t, "gs-stack", p.StarterStackClass())
	assert.Equal(t, "gs-color-muted", p.StarterMutedColorClass())
	assert.Equal(t, "gs-elevation-float", p.StarterElevationClass())
}
