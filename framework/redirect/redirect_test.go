package redirect

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/framework/htmx"
)

func TestRedirectGo_NonHTMX(t *testing.T) {
	e := echo.New()
	e.GET("/users/:id", func(c echo.Context) error { return nil }).Name = "user_show"

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	err := New(ctx).
		Route("user_show").
		Params(42).
		Query(url.Values{"tab": []string{"profile"}}).
		Go()
	if err != nil {
		t.Fatalf("redirect failed: %v", err)
	}

	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	if got := rec.Header().Get(echo.HeaderLocation); got != "/users/42?tab=profile" {
		t.Fatalf("location = %q, want %q", got, "/users/42?tab=profile")
	}
}

func TestRedirectGo_HTMXBoosted(t *testing.T) {
	e := echo.New()
	e.GET("/home", func(c echo.Context) error { return nil }).Name = "home"

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(htmx.HeaderRequest, "true")
	req.Header.Set(htmx.HeaderBoosted, "true")
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	err := New(ctx).Route("home").Go()
	if err != nil {
		t.Fatalf("redirect failed: %v", err)
	}

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got := rec.Header().Get(htmx.HeaderRedirect); got != "/home" {
		t.Fatalf("hx-redirect = %q, want %q", got, "/home")
	}
}
