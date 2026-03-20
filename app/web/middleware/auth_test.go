package middleware

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/leomorpho/goship/framework/context"
	"github.com/leomorpho/goship/framework/tests"

	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestLoadAuthenticatedUser(t *testing.T) {
	ctx, _ := tests.NewContext(c.Web, "/")
	tests.InitSession(ctx)
	mw := LoadAuthenticatedUser(c.Auth, nil, nil)

	// Not authenticated
	_ = tests.ExecuteMiddleware(ctx, mw)
	assert.Nil(t, ctx.Get(context.AuthenticatedUserIDKey))

	// Login
	err := c.Auth.Login(ctx, usr.ID)
	require.NoError(t, err)

	// Verify the midldeware returns the authenticated user
	_ = tests.ExecuteMiddleware(ctx, mw)
	authUserID, ok := ctx.Get(context.AuthenticatedUserIDKey).(int)
	require.True(t, ok)
	assert.Equal(t, usr.ID, authUserID)
	authUserName, ok := ctx.Get(context.AuthenticatedUserNameKey).(string)
	require.True(t, ok)
	assert.Equal(t, usr.Name, authUserName)
	authUserEmail, ok := ctx.Get(context.AuthenticatedUserEmailKey).(string)
	require.True(t, ok)
	assert.Equal(t, usr.Email, authUserEmail)
	if raw := ctx.Get(context.AuthenticatedProfileIDKey); raw != nil {
		profileID, ok := raw.(int)
		require.True(t, ok)
		assert.Positive(t, profileID)
	}
}

func TestRequireAuthentication(t *testing.T) {
	ctx, _ := tests.NewContext(c.Web, "/")
	tests.InitSession(ctx)

	// Not logged in
	err := tests.ExecuteMiddleware(ctx, RequireAuthentication())
	require.NoError(t, err)
	// TODO: not ideal for now. I would like to redirect with a meaningfull code
	// tests.AssertHTTPErrorCode(t, err, http.StatusUnauthorized)

	// Login
	err = c.Auth.Login(ctx, usr.ID)
	require.NoError(t, err)
	_ = tests.ExecuteMiddleware(ctx, LoadAuthenticatedUser(c.Auth, nil, nil))

	// Logged in
	err = tests.ExecuteMiddleware(ctx, RequireAuthentication())
	assert.Nil(t, err)
}

func TestRequireNoAuthentication(t *testing.T) {
	ctx, _ := tests.NewContext(c.Web, "/")
	tests.InitSession(ctx)

	// Not logged in
	err := tests.ExecuteMiddleware(ctx, RequireNoAuthentication())
	assert.Nil(t, err)

	// Login
	err = c.Auth.Login(ctx, usr.ID)
	require.NoError(t, err)
	_ = tests.ExecuteMiddleware(ctx, LoadAuthenticatedUser(c.Auth, nil, nil))

	// Logged in
	err = tests.ExecuteMiddleware(ctx, RequireNoAuthentication())
	assert.Nil(t, err)
}

func TestLoadValidPasswordToken(t *testing.T) {
	ctx, _ := tests.NewContext(c.Web, "/")
	tests.InitSession(ctx)

	// Missing user context
	err := tests.ExecuteMiddleware(ctx, LoadValidPasswordToken(c.Auth))
	tests.AssertHTTPErrorCode(t, err, http.StatusInternalServerError)

	// Add user and password token context but no token and expect a redirect
	ctx.SetParamNames("user", "password_token")
	ctx.SetParamValues(fmt.Sprintf("%d", usr.ID), "1")
	_ = tests.ExecuteMiddleware(ctx, LoadUser(c.Auth))
	err = tests.ExecuteMiddleware(ctx, LoadValidPasswordToken(c.Auth))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusFound, ctx.Response().Status)

	// Add user context and invalid password token and expect a redirect
	ctx.SetParamNames("user", "password_token", "token")
	ctx.SetParamValues(fmt.Sprintf("%d", usr.ID), "1", "faketoken")
	_ = tests.ExecuteMiddleware(ctx, LoadUser(c.Auth))
	err = tests.ExecuteMiddleware(ctx, LoadValidPasswordToken(c.Auth))
	assert.NoError(t, err)
	assert.Equal(t, http.StatusFound, ctx.Response().Status)

	// Create a valid token
	token, tokenID, err := c.Auth.GeneratePasswordResetToken(ctx, usr.ID)
	require.NoError(t, err)

	// Add user and valid password token
	ctx.SetParamNames("user", "password_token", "token")
	ctx.SetParamValues(fmt.Sprintf("%d", usr.ID), fmt.Sprintf("%d", tokenID), token)
	_ = tests.ExecuteMiddleware(ctx, LoadUser(c.Auth))
	err = tests.ExecuteMiddleware(ctx, LoadValidPasswordToken(c.Auth))
	assert.Nil(t, err)
}

func TestRequireAdmin(t *testing.T) {
	ctx, _ := tests.NewContext(c.Web, "/")
	tests.InitSession(ctx)

	err := tests.ExecuteMiddleware(ctx, RequireAdmin())
	tests.AssertHTTPErrorCode(t, err, http.StatusUnauthorized)

	ctx.Set(context.AuthenticatedUserIDKey, usr.ID)
	ctx.Set(context.AuthenticatedUserIsAdminKey, false)
	err = tests.ExecuteMiddleware(ctx, RequireAdmin())
	tests.AssertHTTPErrorCode(t, err, http.StatusForbidden)

	ctx.Set(context.AuthenticatedUserIsAdminKey, true)
	err = tests.ExecuteMiddleware(ctx, RequireAdmin())
	require.NoError(t, err)
}

func TestUserIsAdmin(t *testing.T) {
	t.Setenv("PAGODA_ADMIN_EMAILS", "admin@example.com, owner@example.com")
	assert.True(t, userIsAdmin("admin@example.com"))
	assert.True(t, userIsAdmin("Owner@example.com"))
	assert.False(t, userIsAdmin("user@example.com"))
}
