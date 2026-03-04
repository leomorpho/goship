package ui_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/leomorpho/goship/apps/goship/foundation"
	"github.com/leomorpho/goship/apps/goship/web/middleware"
	"github.com/leomorpho/goship/apps/goship/views/web/components/gen"
	"github.com/leomorpho/goship/apps/goship/views/web/layouts/gen"
	"github.com/leomorpho/goship/apps/goship/views/web/pages/gen"
	"github.com/leomorpho/goship/apps/goship/web/ui"
	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/pkg/htmx"
	"github.com/leomorpho/goship/pkg/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/labstack/echo/v4"
)

var (
	c *foundation.Container
)

func TestMain(m *testing.M) {
	if err := chdirRepoRoot(); err != nil {
		panic(err)
	}

	// Set the environment to test
	config.SwitchEnvironment(config.EnvTest)

	// Create a new container
	c = foundation.NewContainer()

	// Run tests
	exitVal := m.Run()

	// Shutdown the container
	if err := c.Shutdown(); err != nil {
		panic(err)
	}

	os.Exit(exitVal)
}

func chdirRepoRoot() error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return os.Chdir(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return os.ErrNotExist
		}
		dir = parent
	}
}

func TestController_Redirect(t *testing.T) {
	c.Web.GET("/path/:first/and/:second", func(c echo.Context) error {
		return nil
	}).Name = "redirect-test"

	ctx, _ := tests.NewContext(c.Web, "/abc")
	ctr := ui.NewController(c)
	err := ctr.Redirect(ctx, "redirect-test", "one", "two")
	require.NoError(t, err)
	assert.Equal(t, "/path/one/and/two", ctx.Response().Header().Get(echo.HeaderLocation))
	assert.Equal(t, http.StatusFound, ctx.Response().Status)
}

func TestController_RenderToHTMLBlob(t *testing.T) {
	setup := func() (ui.Controller, ui.Page) {
		ctx, _ := tests.NewContext(c.Web, "/test/TestController_RenderPage")
		ctr := ui.NewController(c)

		p := ui.NewPage(ctx)
		p.Name = "healthcheck"
		p.Component = components.ToolTip("", "")
		p.Layout = layouts.Main
		p.Cache.Enabled = false

		return ctr, p
	}

	t.Run("successful rendering", func(t *testing.T) {
		ctr, p := setup()
		htmlBlob, err := ctr.RenderToHTMLBlob(context.Background(), p)
		require.NoError(t, err)
		assert.NotEmpty(t, htmlBlob)
	})

	t.Run("page name missing", func(t *testing.T) {
		ctr, p := setup()
		p.Name = ""
		ctx := context.Background()
		htmlBlob, err := ctr.RenderToHTMLBlob(ctx, p)
		assert.Error(t, err)
		assert.Empty(t, htmlBlob)
	})
}

func TestController_RenderPage(t *testing.T) {
	setup := func() (echo.Context, *httptest.ResponseRecorder, ui.Controller, ui.Page) {
		ctx, rec := tests.NewContext(c.Web, "/test/TestController_RenderPage")
		tests.InitSession(ctx)
		ctr := ui.NewController(c)

		p := ui.NewPage(ctx)
		p.Name = "home"
		p.Layout = layouts.Main
		p.Component = pages.LandingPage(&p)
		p.Cache.Enabled = false
		p.Headers["A"] = "b"
		p.Headers["C"] = "d"
		p.StatusCode = http.StatusCreated
		return ctx, rec, ctr, p
	}

	t.Run("missing name", func(t *testing.T) {
		// Rendering should fail if the Page has no name
		ctx, _, ctr, p := setup()
		p.Name = ""
		err := ctr.RenderPage(ctx, p)
		assert.Error(t, err)
	})

	t.Run("missing component", func(t *testing.T) {
		// Rendering should fail if the Page has no component
		ctx, _, ctr, p := setup()
		p.Name = "home"
		p.Component = nil
		err := ctr.RenderPage(ctx, p)
		assert.Error(t, err)
	})

	t.Run("no page cache", func(t *testing.T) {
		ctx, _, ctr, p := setup()
		err := ctr.RenderPage(ctx, p)
		require.NoError(t, err)

		// Check status code and headers
		assert.Equal(t, http.StatusCreated, ctx.Response().Status)
		for k, v := range p.Headers {
			assert.Equal(t, v, ctx.Response().Header().Get(k))
		}
	})

	t.Run("htmx rendering", func(t *testing.T) {
		ctx, _, ctr, p := setup()
		p.HTMX.Request.Enabled = true
		p.HTMX.Response = &htmx.Response{
			Trigger: "trigger",
		}
		err := ctr.RenderPage(ctx, p)
		require.NoError(t, err)

		// Check HTMX header
		assert.Equal(t, "trigger", ctx.Response().Header().Get(htmx.HeaderTrigger))
	})

	t.Run("page cache", func(t *testing.T) {
		if c.Cache == nil {
			t.Skip("cache service is not initialized in this runtime profile")
		}

		ctx, rec, ctr, p := setup()
		p.Cache.Enabled = true
		p.Cache.Tags = []string{"tag1"}
		err := ctr.RenderPage(ctx, p)
		require.NoError(t, err)

		// Fetch from the cache
		res, err := c.Cache.
			Get().
			Group(middleware.CachedPageGroup).
			Key(p.URL).
			Type(new(middleware.CachedPage)).
			Fetch(context.Background())
		require.NoError(t, err)

		// Compare the cached page
		cp, ok := res.(*middleware.CachedPage)
		require.True(t, ok)
		assert.Equal(t, p.URL, cp.URL)
		assert.Equal(t, p.Headers, cp.Headers)
		assert.Equal(t, p.StatusCode, cp.StatusCode)
		assert.Equal(t, rec.Body.Bytes(), cp.HTML)

		// Clear the tag
		err = c.Cache.
			Flush().
			Tags(p.Cache.Tags[0]).
			Execute(context.Background())
		require.NoError(t, err)

		// Refetch from the cache and expect no results
		_, err = c.Cache.
			Get().
			Group(middleware.CachedPageGroup).
			Key(p.URL).
			Type(new(middleware.CachedPage)).
			Fetch(context.Background())
		assert.Error(t, err)
	})
}
