package foundation

import (
	stdcontext "context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/gorilla/sessions"
	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/db/ent"
	"github.com/leomorpho/goship/framework/context"
	"github.com/leomorpho/goship/framework/dberrors"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

const (
	// authSessionName stores the name of the session which contains authentication data
	authSessionName = "ua"

	// authSessionKeyUserID stores the key used to store the user ID in the session
	authSessionKeyUserID = "user_id"

	// authSessionKeyAuthenticated stores the key used to store the authentication status in the session
	authSessionKeyAuthenticated = "authenticated"
)

// NotAuthenticatedError is an error returned when a user is not authenticated
type NotAuthenticatedError struct{}

// Error implements the error interface.
func (e NotAuthenticatedError) Error() string {
	return "user not authenticated"
}

// InvalidPasswordTokenError is an error returned when an invalid token is provided
type InvalidPasswordTokenError struct{}

// Error implements the error interface.
func (e InvalidPasswordTokenError) Error() string {
	return "invalid password token"
}

// InvalidCredentialsError is returned when email/password authentication fails.
type InvalidCredentialsError struct{}

// Error implements the error interface.
func (e InvalidCredentialsError) Error() string {
	return "invalid credentials"
}

// AuthClient is the client that handles authentication requests
type AuthClient struct {
	config *config.Config
	store  authStore
}

// NewAuthClient creates a new authentication client
func NewAuthClient(cfg *config.Config, orm *ent.Client, db *sql.DB) *AuthClient {
	return &AuthClient{
		config: cfg,
		store:  selectAuthStore(cfg, orm, db),
	}
}

// Login logs in a user of a given ID
func (c *AuthClient) Login(ctx echo.Context, userID int) error {

	sess, err := session.Get(authSessionName, ctx)
	if err != nil {
		return err
	}

	sess.Values[authSessionKeyUserID] = userID
	sess.Values[authSessionKeyAuthenticated] = true
	return sess.Save(ctx.Request(), ctx.Response())
}

// Logout logs the requesting user out
func (c *AuthClient) Logout(ctx echo.Context) error {
	sess, err := session.Get(authSessionName, ctx)
	if err != nil {
		return err
	}

	// Overwrite session values
	sess.Values[authSessionKeyAuthenticated] = false

	// TODO: not quite sure why, but resetting the cookie is not needed in the vanilla
	// starter kit from Pagoda. Not sure which one of my changes broke that.
	// Set the cookie to expire immediately
	sess.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   -1, // Set MaxAge to -1 to delete the session
		HttpOnly: true,
	}

	return sess.Save(ctx.Request(), ctx.Response())
}

// GetAuthenticatedUserID returns the authenticated user's ID, if the user is logged in
func (c *AuthClient) GetAuthenticatedUserID(ctx echo.Context) (int, error) {
	sess, err := session.Get(authSessionName, ctx)
	if err != nil {
		return 0, err
	}

	if sess.Values[authSessionKeyAuthenticated] == true {
		return sess.Values[authSessionKeyUserID].(int), nil
	}

	return 0, NotAuthenticatedError{}
}

// GetAuthenticatedIdentity returns the authenticated identity if the user is logged in.
func (c *AuthClient) GetAuthenticatedIdentity(ctx echo.Context) (*AuthIdentity, error) {
	if userID, err := c.GetAuthenticatedUserID(ctx); err == nil {
		return c.store.GetIdentityByUserID(ctx.Request().Context(), userID)
	}

	return nil, NotAuthenticatedError{}
}

// GetIdentityByUserID returns an auth identity for an explicit user ID.
func (c *AuthClient) GetIdentityByUserID(ctx stdcontext.Context, userID int) (*AuthIdentity, error) {
	return c.store.GetIdentityByUserID(ctx, userID)
}

// FindUserRecordByEmail returns an auth user record by email (case-insensitive).
func (c *AuthClient) FindUserRecordByEmail(ctx echo.Context, email string) (*AuthUserRecord, error) {
	return c.store.GetUserRecordByEmail(ctx.Request().Context(), email)
}

// SetLastOnlineTimestamp sets the last online time for a user
func (c *AuthClient) SetLastOnlineTimestamp(ctx echo.Context, userID int) error {
	return c.store.CreateLastSeenOnline(ctx.Request().Context(), userID, time.Now())
}

// AuthenticateUserByEmailPassword authenticates credentials and returns the user record on success.
func (c *AuthClient) AuthenticateUserByEmailPassword(
	ctx echo.Context,
	email string,
	password string,
) (*AuthUserRecord, error) {
	userRecord, err := c.FindUserRecordByEmail(ctx, email)
	switch {
	case dberrors.IsNotFound(err):
		return nil, InvalidCredentialsError{}
	case err != nil:
		return nil, err
	}

	if err := c.CheckPassword(password, userRecord.Password); err != nil {
		return nil, InvalidCredentialsError{}
	}

	return userRecord, nil
}

func (c *AuthClient) GetUserDisplayNameByUserID(ctx echo.Context, userID int) (string, error) {
	return c.store.GetUserDisplayNameByUserID(ctx.Request().Context(), userID)
}

func (c *AuthClient) SetUserDisplayNameByUserID(ctx echo.Context, userID int, displayName string) error {
	return c.store.UpdateUserDisplayNameByUserID(ctx.Request().Context(), userID, displayName)
}

func (c *AuthClient) SetUserPasswordHashByUserID(ctx echo.Context, userID int, passwordHash string) error {
	return c.store.UpdateUserPasswordHashByUserID(ctx.Request().Context(), userID, passwordHash)
}

func (c *AuthClient) MarkUserVerifiedByUserID(ctx echo.Context, userID int) error {
	return c.store.MarkUserVerifiedByUserID(ctx.Request().Context(), userID)
}

// HashPassword returns a hash of a given password
func (c *AuthClient) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword check if a given password matches a given hash
func (c *AuthClient) CheckPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// GeneratePasswordResetToken generates a password reset token for a given user.
// For security purposes, the token itself is not stored in the database but rather
// a hash of the token, exactly how passwords are handled. This method returns both
// the generated token as well as the created password token ID.
func (c *AuthClient) GeneratePasswordResetToken(ctx echo.Context, userID int) (string, int, error) {
	// Generate the token, which is what will go in the URL, but not the database
	token, err := c.RandomToken(c.config.App.PasswordToken.Length)
	if err != nil {
		return "", 0, err
	}

	// Hash the token, which is what will be stored in the database
	hash, err := c.HashPassword(token)
	if err != nil {
		return "", 0, err
	}

	// Create and save the password reset token
	tokenID, err := c.store.CreatePasswordToken(ctx.Request().Context(), userID, hash)
	if err != nil {
		return "", 0, err
	}
	return token, tokenID, nil
}

// GetValidPasswordToken validates a non-expired password token for a given user/token ID combination.
// Since the raw token is not stored in the database for security purposes, the provided token is checked
// against the stored hash.
func (c *AuthClient) GetValidPasswordToken(ctx echo.Context, userID, tokenID int, token string) error {
	// Ensure expired tokens are never returned
	expiration := time.Now().Add(-c.config.App.PasswordToken.Expiration)

	hash, err := c.store.GetPasswordTokenHash(ctx.Request().Context(), userID, tokenID, expiration)
	switch {
	case dberrors.IsNotFound(err):
	case err == nil:
		// Check the token for a hash match
		if err := c.CheckPassword(token, hash); err == nil {
			return nil
		}
	default:
		if !context.IsCanceledError(err) {
			return err
		}
	}

	return InvalidPasswordTokenError{}
}

// DeletePasswordTokens deletes all password tokens in the database for a belonging to a given user.
// This should be called after a successful password reset.
func (c *AuthClient) DeletePasswordTokens(ctx echo.Context, userID int) error {
	return c.store.DeletePasswordTokens(ctx.Request().Context(), userID)
}

// RandomToken generates a random token string of a given length
func (c *AuthClient) RandomToken(length int) (string, error) {
	b := make([]byte, (length/2)+1)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	return token[:length], nil
}

// GenerateEmailVerificationToken generates an email verification token for a given email address using JWT which
// is set to expire based on the duration stored in configuration
func (c *AuthClient) GenerateEmailVerificationToken(email string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
		"exp":   time.Now().Add(c.config.App.EmailVerificationTokenExpiration).Unix(),
	})

	return token.SignedString([]byte(c.config.App.EncryptionKey))
}

// ValidateEmailVerificationToken validates an email verification token and returns the associated email address if
// the token is valid and has not expired
func (c *AuthClient) ValidateEmailVerificationToken(token string) (string, error) {
	t, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}

		return []byte(c.config.App.EncryptionKey), nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := t.Claims.(jwt.MapClaims); ok && t.Valid {
		return claims["email"].(string), nil
	}

	return "", errors.New("invalid or expired token")
}
