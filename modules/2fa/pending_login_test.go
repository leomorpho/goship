package twofa

import (
	"errors"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/framework/tests"
)

func TestPendingUserCookieRoundTrip(t *testing.T) {
	e := echo.New()
	ctx, rec := tests.NewContext(e, "/auth/2fa/verify")

	if err := SetPendingUserCookie(ctx, "secret", 42); err != nil {
		t.Fatalf("SetPendingUserCookie() error = %v", err)
	}
	for _, cookie := range rec.Result().Cookies() {
		ctx.Request().AddCookie(cookie)
	}

	got, err := PendingUserIDFromCookie(ctx, "secret")
	if err != nil {
		t.Fatalf("PendingUserIDFromCookie() error = %v", err)
	}
	if got != 42 {
		t.Fatalf("expected user id 42, got %d", got)
	}
}

func TestPendingUserCookieRejectsWrongSecret(t *testing.T) {
	e := echo.New()
	ctx, rec := tests.NewContext(e, "/auth/2fa/verify")

	if err := SetPendingUserCookie(ctx, "secret", 42); err != nil {
		t.Fatalf("SetPendingUserCookie() error = %v", err)
	}
	for _, cookie := range rec.Result().Cookies() {
		ctx.Request().AddCookie(cookie)
	}

	_, err := PendingUserIDFromCookie(ctx, "wrong-secret")
	if !errors.Is(err, ErrPendingUserCookieInvalid) {
		t.Fatalf("expected ErrPendingUserCookieInvalid, got %v", err)
	}
}
