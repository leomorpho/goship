package page_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	frameworkctx "github.com/leomorpho/goship/framework/context"
	"github.com/leomorpho/goship/framework/web/page"
	"github.com/stretchr/testify/assert"
)

func TestNewBase(t *testing.T) {
	t.Parallel()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/?a=1", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	p := page.NewBase(ctx)
	assert.Same(t, ctx, p.Context)
	assert.NotNil(t, p.ToURL)
	assert.Equal(t, "/", p.Path)
	assert.Equal(t, "/?a=1", p.URL)
	assert.Equal(t, http.StatusOK, p.StatusCode)
	assert.True(t, p.IsHome)
	assert.False(t, p.IsAuth)
	assert.False(t, p.IsAdmin)
	assert.Empty(t, p.CSRF)
	assert.Empty(t, p.RequestID)
	assert.Empty(t, p.Headers)
	assert.False(t, p.Cache.Enabled)
	assert.False(t, p.IsIosDevice)

	ctx.Set(frameworkctx.AuthenticatedUserIDKey, 7)
	ctx.Set(frameworkctx.AuthenticatedUserIsAdminKey, true)
	ctx.Set(frameworkctx.IsFromIOSApp, true)
	ctx.Set(echomw.DefaultCSRFConfig.ContextKey, "csrf-token")

	p = page.NewBase(ctx)
	assert.True(t, p.IsAuth)
	assert.True(t, p.IsAdmin)
	assert.True(t, p.IsIosDevice)
	assert.Equal(t, "csrf-token", p.CSRF)
}
