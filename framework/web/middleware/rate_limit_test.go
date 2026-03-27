package middleware

import (
	"net/http"
	"testing"
	"time"

	"github.com/leomorpho/goship/framework/appcontext"
	"github.com/leomorpho/goship/framework/ratelimit"
	"github.com/leomorpho/goship/framework/testkit"
)

func TestRateLimit_RejectsAfterLimit(t *testing.T) {
	store, err := ratelimit.NewOtterStore(128)
	if err != nil {
		t.Fatalf("NewOtterStore error = %v", err)
	}
	t.Cleanup(store.Close)

	mw := RateLimit(store, 1, time.Minute)

	ctx, _ := tests.NewContext(c.Web, "/user/login")
	ctx.SetPath("/user/login")
	ctx.Request().Method = http.MethodPost
	ctx.Request().RemoteAddr = "198.51.100.10:3210"

	if err := tests.ExecuteMiddleware(ctx, mw); err != nil {
		t.Fatalf("first request err = %v, want nil", err)
	}
	err = tests.ExecuteMiddleware(ctx, mw)
	tests.AssertHTTPErrorCode(t, err, http.StatusTooManyRequests)
	if got := ctx.Response().Header().Get("Retry-After"); got == "" {
		t.Fatal("Retry-After header missing on limited request")
	}
}

func TestRateLimit_KeysAuthenticatedUsersSeparately(t *testing.T) {
	store, err := ratelimit.NewOtterStore(128)
	if err != nil {
		t.Fatalf("NewOtterStore error = %v", err)
	}
	t.Cleanup(store.Close)

	mw := RateLimit(store, 1, time.Minute)

	ctxA, _ := tests.NewContext(c.Web, "/user/login")
	ctxA.SetPath("/user/login")
	ctxA.Request().Method = http.MethodPost
	ctxA.Request().RemoteAddr = "198.51.100.10:3210"
	ctxA.Set(appcontext.AuthenticatedUserIDKey, 101)

	ctxB, _ := tests.NewContext(c.Web, "/user/login")
	ctxB.SetPath("/user/login")
	ctxB.Request().Method = http.MethodPost
	ctxB.Request().RemoteAddr = "198.51.100.10:3210"
	ctxB.Set(appcontext.AuthenticatedUserIDKey, 202)

	if err := tests.ExecuteMiddleware(ctxA, mw); err != nil {
		t.Fatalf("user A request err = %v, want nil", err)
	}
	if err := tests.ExecuteMiddleware(ctxB, mw); err != nil {
		t.Fatalf("user B request err = %v, want nil", err)
	}
}
