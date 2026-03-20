package foundation

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/leomorpho/goship/framework/tests"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

func TestAuthClient_Auth(t *testing.T) {
	assertNoAuth := func() {
		_, err := c.Auth.GetAuthenticatedUserID(ctx)
		assert.True(t, errors.Is(err, NotAuthenticatedError{}))
		_, err = c.Auth.GetAuthenticatedIdentity(ctx)
		assert.True(t, errors.Is(err, NotAuthenticatedError{}))
	}

	assertNoAuth()

	err := c.Auth.Login(ctx, usr.ID)
	require.NoError(t, err)

	uid, err := c.Auth.GetAuthenticatedUserID(ctx)
	require.NoError(t, err)
	assert.Equal(t, usr.ID, uid)

	u, err := c.Auth.GetAuthenticatedIdentity(ctx)
	require.NoError(t, err)
	assert.Equal(t, u.UserID, usr.ID)
	assert.Equal(t, u.UserName, usr.Name)
	assert.Equal(t, u.UserEmail, usr.Email)

	err = c.Auth.Logout(ctx)
	require.NoError(t, err)

	assertNoAuth()
}

func TestAuthClient_FindUserRecordByEmail(t *testing.T) {
	u, err := c.Auth.FindUserRecordByEmail(ctx, usr.Email)
	require.NoError(t, err)
	assert.Equal(t, usr.ID, u.UserID)
	assert.Equal(t, usr.Name, u.Name)
	assert.Equal(t, usr.Email, u.Email)
	assert.Equal(t, usr.Password, u.Password)
	assert.Equal(t, usr.Verified, u.IsVerified)
}

func TestAuthClient_AuthenticateUserByEmailPassword(t *testing.T) {
	password := "password"
	hash, err := c.Auth.HashPassword(password)
	require.NoError(t, err)
	loginUser, err := tests.CreateUserDB(
		ctx.Request().Context(),
		c.Database,
		"Auth Login User",
		fmt.Sprintf("auth-login-%d@localhost.localhost", time.Now().UnixNano()),
		hash,
		true,
	)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		u, err := c.Auth.AuthenticateUserByEmailPassword(ctx, loginUser.Email, password)
		require.NoError(t, err)
		assert.Equal(t, loginUser.ID, u.UserID)
	})

	t.Run("bad email", func(t *testing.T) {
		_, err := c.Auth.AuthenticateUserByEmailPassword(ctx, "missing@example.com", "password")
		assert.Error(t, err)
		_, ok := err.(InvalidCredentialsError)
		assert.True(t, ok)
	})

	t.Run("bad password", func(t *testing.T) {
		_, err := c.Auth.AuthenticateUserByEmailPassword(ctx, loginUser.Email, "not-the-password")
		assert.Error(t, err)
		_, ok := err.(InvalidCredentialsError)
		assert.True(t, ok)
	})
}

func TestAuthClient_PasswordHashing(t *testing.T) {
	pw := "testcheckpassword"
	hash, err := c.Auth.HashPassword(pw)
	assert.NoError(t, err)
	assert.NotEqual(t, hash, pw)
	err = c.Auth.CheckPassword(pw, hash)
	assert.NoError(t, err)
}

func TestAuthClient_GeneratePasswordResetToken(t *testing.T) {
	token, tokenID, err := c.Auth.GeneratePasswordResetToken(ctx, usr.ID)
	require.NoError(t, err)
	assert.Len(t, token, c.Config.App.PasswordToken.Length)
	var tokenHash string
	err = c.Database.QueryRowContext(ctx.Request().Context(),
		`SELECT hash FROM password_tokens WHERE id = $1`, tokenID).Scan(&tokenHash)
	require.NoError(t, err)
	assert.NoError(t, c.Auth.CheckPassword(token, tokenHash))
}

func TestAuthClient_GetValidPasswordToken(t *testing.T) {
	// Check that a fake token is not valid
	err := c.Auth.GetValidPasswordToken(ctx, usr.ID, 1, "faketoken")
	assert.Error(t, err)

	// Generate a valid token and check that it is accepted
	token, tokenID, err := c.Auth.GeneratePasswordResetToken(ctx, usr.ID)
	require.NoError(t, err)
	err = c.Auth.GetValidPasswordToken(ctx, usr.ID, tokenID, token)
	require.NoError(t, err)

	// Expire the token by pushing the date far enough back
	res, err := c.Database.ExecContext(
		ctx.Request().Context(),
		`UPDATE password_tokens SET created_at = $1 WHERE id = $2`,
		time.Now().Add(-(c.Config.App.PasswordToken.Expiration + time.Hour)),
		tokenID,
	)
	require.NoError(t, err)
	count64, err := res.RowsAffected()
	require.NoError(t, err)
	count := int(count64)
	require.Equal(t, 1, count)

	// Expired tokens should not be valid
	err = c.Auth.GetValidPasswordToken(ctx, usr.ID, tokenID, token)
	assert.Error(t, err)
}

func TestAuthClient_DeletePasswordTokens(t *testing.T) {
	// Create three tokens for the user
	for i := 0; i < 3; i++ {
		_, _, err := c.Auth.GeneratePasswordResetToken(ctx, usr.ID)
		require.NoError(t, err)
	}

	// Delete all tokens for the user
	err := c.Auth.DeletePasswordTokens(ctx, usr.ID)
	require.NoError(t, err)

	// Check that no tokens remain
	var count int
	err = c.Database.QueryRowContext(
		ctx.Request().Context(),
		`SELECT COUNT(*) FROM password_tokens WHERE password_token_user = $1`,
		usr.ID,
	).Scan(&count)

	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestAuthClient_RandomToken(t *testing.T) {
	length := c.Config.App.PasswordToken.Length
	a, err := c.Auth.RandomToken(length)
	require.NoError(t, err)
	b, err := c.Auth.RandomToken(length)
	require.NoError(t, err)
	assert.Len(t, a, length)
	assert.Len(t, b, length)
	assert.NotEqual(t, a, b)
}

func TestAuthClient_EmailVerificationToken(t *testing.T) {
	t.Run("valid token", func(t *testing.T) {
		email := "test@localhost.com"
		token, err := c.Auth.GenerateEmailVerificationToken(email)
		require.NoError(t, err)

		tokenEmail, err := c.Auth.ValidateEmailVerificationToken(token)
		require.NoError(t, err)
		assert.Equal(t, email, tokenEmail)
	})

	t.Run("invalid token", func(t *testing.T) {
		badToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJlbWFpbCI6InRlc3RAbG9jYWxob3N0LmNvbSIsImV4cCI6MTkxNzg2NDAwMH0.ScJCpfEEzlilKfRs_aVouzwPNKI28M3AIm-hyImQHUQ"
		_, err := c.Auth.ValidateEmailVerificationToken(badToken)
		assert.Error(t, err)
	})

	t.Run("expired token", func(t *testing.T) {
		c.Config.App.EmailVerificationTokenExpiration = -time.Hour
		email := "test@localhost.com"
		token, err := c.Auth.GenerateEmailVerificationToken(email)
		require.NoError(t, err)

		_, err = c.Auth.ValidateEmailVerificationToken(token)
		assert.Error(t, err)

		c.Config.App.EmailVerificationTokenExpiration = time.Hour * 12
	})
}
