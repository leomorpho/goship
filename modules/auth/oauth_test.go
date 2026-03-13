package auth

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/foundation"
	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/tests"
	profilesvc "github.com/leomorpho/goship/modules/profile"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/oauth2"
	_ "modernc.org/sqlite"
)

type fakeOAuthProvider struct {
	name string
	user *OAuthUser
}

func (p fakeOAuthProvider) Name() string {
	return p.name
}

func (p fakeOAuthProvider) Config() *oauth2.Config {
	return &oauth2.Config{}
}

func (p fakeOAuthProvider) FetchUser(context.Context, *http.Client) (*OAuthUser, error) {
	return p.user, nil
}

type fakeOAuthAuthClient struct {
	db *sql.DB
}

func (f fakeOAuthAuthClient) RandomToken(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("invalid length")
	}
	return "0123456789abcdef0123456789abcdef", nil
}

func (f fakeOAuthAuthClient) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (f fakeOAuthAuthClient) GetIdentityByUserID(ctx context.Context, userID int) (*foundation.AuthIdentity, error) {
	row := f.db.QueryRowContext(ctx, `
		SELECT u.id, u.name, u.email, p.id, COALESCE(p.fully_onboarded, false)
		FROM users u
		LEFT JOIN profiles p ON p.user_profile = u.id
		WHERE u.id = ?`, userID)
	identity := foundation.AuthIdentity{}
	identity.HasProfile = false
	var profileID sql.NullInt64
	var fullyOnboarded sql.NullBool
	if err := row.Scan(&identity.UserID, &identity.UserName, &identity.UserEmail, &profileID, &fullyOnboarded); err != nil {
		return nil, err
	}
	if profileID.Valid {
		identity.HasProfile = true
		identity.ProfileID = int(profileID.Int64)
		identity.ProfileFullyOnboarded = fullyOnboarded.Valid && fullyOnboarded.Bool
	}
	return &identity, nil
}

func TestOAuthServiceHandleCallback_CreatesUserProfileAndAccount(t *testing.T) {
	db := newOAuthTestDB(t)
	cfg := oauthTestConfig()
	authClient := fakeOAuthAuthClient{db: db}
	profileService := profilesvc.NewProfileServiceWithDBDeps(db, "sqlite", nil, nil, nil)
	service := NewOAuthService(cfg, db, authClient, Deps{ProfileService: *profileService})
	service.providers = map[string]OAuthProvider{
		"github": fakeOAuthProvider{
			name: "github",
			user: &OAuthUser{
				ProviderID: "gh-123",
				Email:      "new-user@example.com",
				Name:       "New User",
			},
		},
	}
	service.exchangeCode = func(context.Context, OAuthProvider, string) (*oauth2.Token, error) {
		return &oauth2.Token{AccessToken: "provider-access-token"}, nil
	}

	result, err := service.HandleCallback(context.Background(), "github", "valid-code")
	if err != nil {
		t.Fatalf("HandleCallback() error = %v", err)
	}
	if !result.NewUser {
		t.Fatalf("expected new user result")
	}
	if result.UserID <= 0 || result.ProfileID <= 0 {
		t.Fatalf("expected user and profile IDs, got %#v", result)
	}

	var userCount int
	if err := db.QueryRow(`SELECT COUNT(1) FROM users`).Scan(&userCount); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if userCount != 1 {
		t.Fatalf("expected 1 user, got %d", userCount)
	}

	var storedToken string
	if err := db.QueryRow(`SELECT token FROM oauth_accounts WHERE provider = 'github' AND provider_id = 'gh-123'`).Scan(&storedToken); err != nil {
		t.Fatalf("query oauth account: %v", err)
	}
	if storedToken == "provider-access-token" || storedToken == "" {
		t.Fatalf("expected encrypted oauth token, got %q", storedToken)
	}
}

func TestOAuthServiceHandleCallback_LinksExistingUserByEmail(t *testing.T) {
	db := newOAuthTestDB(t)
	cfg := oauthTestConfig()
	authClient := fakeOAuthAuthClient{db: db}
	profileService := profilesvc.NewProfileServiceWithDBDeps(db, "sqlite", nil, nil, nil)

	passwordHash, err := authClient.HashPassword("password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	registration, err := profileService.RegisterUserWithProfile(
		context.Background(),
		"Existing User",
		"existing@example.com",
		passwordHash,
		time.Date(1990, time.January, 1, 0, 0, 0, 0, time.UTC),
		nil,
	)
	if err != nil {
		t.Fatalf("RegisterUserWithProfile() error = %v", err)
	}

	service := NewOAuthService(cfg, db, authClient, Deps{ProfileService: *profileService})
	service.providers = map[string]OAuthProvider{
		"github": fakeOAuthProvider{
			name: "github",
			user: &OAuthUser{
				ProviderID: "gh-999",
				Email:      "existing@example.com",
				Name:       "Existing User",
			},
		},
	}
	service.exchangeCode = func(context.Context, OAuthProvider, string) (*oauth2.Token, error) {
		return &oauth2.Token{AccessToken: "provider-access-token"}, nil
	}

	result, err := service.HandleCallback(context.Background(), "github", "valid-code")
	if err != nil {
		t.Fatalf("HandleCallback() error = %v", err)
	}
	if result.NewUser {
		t.Fatalf("expected existing user link, got new user")
	}
	if result.UserID != registration.UserID {
		t.Fatalf("expected user ID %d, got %d", registration.UserID, result.UserID)
	}

	var userCount int
	if err := db.QueryRow(`SELECT COUNT(1) FROM users`).Scan(&userCount); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if userCount != 1 {
		t.Fatalf("expected no extra user rows, got %d", userCount)
	}
}

func TestConsumeOAuthState_ValidatesAndClearsSession(t *testing.T) {
	e := echo.New()
	ctx, _ := tests.NewContext(e, "/auth/oauth/github/callback")
	tests.InitSession(ctx)

	sess, err := session.Get("session", ctx)
	if err != nil {
		t.Fatalf("session.Get() error = %v", err)
	}
	sess.Values["oauth_state"] = "expected-state"
	sess.Values["oauth_provider"] = "github"
	if err := sess.Save(ctx.Request(), ctx.Response()); err != nil {
		t.Fatalf("sess.Save() error = %v", err)
	}

	if err := consumeOAuthState(ctx, "github", "expected-state"); err != nil {
		t.Fatalf("consumeOAuthState() error = %v", err)
	}
	if err := consumeOAuthState(ctx, "github", "expected-state"); !errors.Is(err, errOAuthStateInvalid) {
		t.Fatalf("expected cleared session to fail with errOAuthStateInvalid, got %v", err)
	}
}

func oauthTestConfig() *config.Config {
	cfg := &config.Config{}
	cfg.HTTP.Domain = "http://localhost:8000"
	cfg.Adapters.DB = "sqlite"
	cfg.App.EncryptionKey = "test-encryption-key"
	cfg.App.AppEncryptionKey = "test-app-encryption-key"
	cfg.OAuth.GitHub.ClientID = "github-client-id"
	cfg.OAuth.GitHub.ClientSecret = "github-client-secret"
	return cfg
}

func newOAuthTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", "file:"+t.Name()+"?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	schema := []string{
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			name TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			password TEXT NOT NULL,
			verified BOOLEAN NOT NULL DEFAULT FALSE
		);`,
		`CREATE TABLE profiles (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			bio TEXT NOT NULL,
			birthdate DATETIME NOT NULL,
			age INTEGER NOT NULL,
			fully_onboarded BOOLEAN NOT NULL DEFAULT FALSE,
			phone_verified BOOLEAN NOT NULL DEFAULT FALSE,
			user_profile INTEGER NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE oauth_accounts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			provider TEXT NOT NULL,
			provider_id TEXT NOT NULL,
			email TEXT,
			token TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(provider, provider_id)
		);`,
	}
	for _, stmt := range schema {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("db.Exec() error = %v", err)
		}
	}
	return db
}
