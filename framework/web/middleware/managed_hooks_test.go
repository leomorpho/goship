package middleware

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	frameworksecurity "github.com/leomorpho/goship/framework/security"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireManagedHookSignature_AllowsValidSignedRequest(t *testing.T) {
	t.Setenv(frameworksecurity.ManagedHooksNonceStorePathEnv, filepath.Join(t.TempDir(), "nonces.json"))
	e := echo.New()
	verifier := frameworksecurity.NewManagedHookVerifier("secret", 5*time.Minute, 5*time.Minute)

	var called bool
	h := RequireManagedHookSignature(verifier)(func(c echo.Context) error {
		called = true
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/managed/status", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	ts := time.Now().UTC().Unix()
	nonce := "nonce-1"
	sig := frameworksecurity.SignManagedHookRequest("secret", req.Method, "/managed/status", ts, nonce, []byte{})
	req.Header.Set(frameworksecurity.HeaderManagedTimestamp, strconv.FormatInt(ts, 10))
	req.Header.Set(frameworksecurity.HeaderManagedNonce, nonce)
	req.Header.Set(frameworksecurity.HeaderManagedSignature, sig)

	err := h(ctx)
	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRequireManagedHookSignature_RejectsUnsignedRequest(t *testing.T) {
	t.Setenv(frameworksecurity.ManagedHooksNonceStorePathEnv, filepath.Join(t.TempDir(), "nonces.json"))
	e := echo.New()
	verifier := frameworksecurity.NewManagedHookVerifier("secret", 5*time.Minute, 5*time.Minute)
	h := RequireManagedHookSignature(verifier)(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/managed/status", nil)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	err := h(ctx)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
