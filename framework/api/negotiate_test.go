package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestIsAPIRequestDetectsAcceptHeader(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/posts/1", nil)
	req.Header.Set(echo.HeaderAccept, "text/html, application/json")
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.SetPath("/posts/1")

	if !IsAPIRequest(ctx) {
		t.Fatal("IsAPIRequest() = false, want true")
	}
}

func TestIsAPIRequestDetectsAPIPrefix(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/posts/1", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.SetPath("/api/v1/posts/:id")

	if !IsAPIRequest(ctx) {
		t.Fatal("IsAPIRequest() = false, want true")
	}
}

func TestIsAPIRequestRejectsHTMLPageRequests(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/posts/1", nil)
	req.Header.Set(echo.HeaderAccept, "text/html")
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)
	ctx.SetPath("/posts/:id")

	if IsAPIRequest(ctx) {
		t.Fatal("IsAPIRequest() = true, want false")
	}
}
