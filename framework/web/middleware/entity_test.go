package middleware

import (
	"fmt"
	"testing"

	"github.com/leomorpho/goship/framework/appcontext"
	"github.com/leomorpho/goship/framework/tests"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadUser(t *testing.T) {
	ctx, _ := tests.NewContext(c.Web, "/")
	ctx.SetParamNames("user")
	ctx.SetParamValues(fmt.Sprintf("%d", usr.ID))
	_ = tests.ExecuteMiddleware(ctx, LoadUser(c.Auth))
	authUserID, ok := ctx.Get(appcontext.AuthenticatedUserIDKey).(int)
	require.True(t, ok)
	assert.Equal(t, usr.ID, authUserID)
	authUserEmail, ok := ctx.Get(appcontext.AuthenticatedUserEmailKey).(string)
	require.True(t, ok)
	assert.Equal(t, usr.Email, authUserEmail)
}
