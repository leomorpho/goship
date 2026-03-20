package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	customctx "github.com/leomorpho/goship/framework/context"
)

func TestAuthenticatedUserEmail(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	if _, err := authenticatedUserEmail(ctx); err == nil {
		t.Fatal("expected error when user email is missing from context")
	}

	ctx.Set(customctx.AuthenticatedUserEmailKey, "")
	if _, err := authenticatedUserEmail(ctx); err == nil {
		t.Fatal("expected error for empty user email")
	}

	ctx.Set(customctx.AuthenticatedUserEmailKey, "user@example.com")
	email, err := authenticatedUserEmail(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if email != "user@example.com" {
		t.Fatalf("email = %q, want %q", email, "user@example.com")
	}
}
