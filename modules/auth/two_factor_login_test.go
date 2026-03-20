package auth

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/leomorpho/goship/app/foundation"
	"github.com/leomorpho/goship/app/web/routenames"
	"github.com/leomorpho/goship/app/web/ui"
	"github.com/leomorpho/goship/config"
	frameworkvalidation "github.com/leomorpho/goship/framework/web/validation"
	"github.com/leomorpho/goship/modules/authsupport"
	"golang.org/x/crypto/bcrypt"
)

type fakeLoginAuthStore struct {
	userRecord *authsupport.AuthUserRecord
	identity   *authsupport.AuthIdentity
}

func (f fakeLoginAuthStore) GetIdentityByUserID(context.Context, int) (*authsupport.AuthIdentity, error) {
	if f.identity == nil {
		return nil, sql.ErrNoRows
	}
	return f.identity, nil
}

func (f fakeLoginAuthStore) GetUserRecordByEmail(context.Context, string) (*authsupport.AuthUserRecord, error) {
	if f.userRecord == nil {
		return nil, sql.ErrNoRows
	}
	return f.userRecord, nil
}

func (f fakeLoginAuthStore) GetUserDisplayNameByUserID(context.Context, int) (string, error) {
	return "", errors.New("not implemented")
}

func (f fakeLoginAuthStore) UpdateUserDisplayNameByUserID(context.Context, int, string) error {
	return errors.New("not implemented")
}

func (f fakeLoginAuthStore) UpdateUserPasswordHashByUserID(context.Context, int, string) error {
	return errors.New("not implemented")
}

func (f fakeLoginAuthStore) MarkUserVerifiedByUserID(context.Context, int) error {
	return errors.New("not implemented")
}

func (f fakeLoginAuthStore) CreateLastSeenOnline(context.Context, int, time.Time) error {
	return errors.New("not implemented")
}

func (f fakeLoginAuthStore) CreatePasswordToken(context.Context, int, string) (int, error) {
	return 0, errors.New("not implemented")
}

func (f fakeLoginAuthStore) GetPasswordTokenHash(context.Context, int, int, time.Time) (string, error) {
	return "", errors.New("not implemented")
}

func (f fakeLoginAuthStore) DeletePasswordTokens(context.Context, int) error {
	return errors.New("not implemented")
}

type fakeTwoFactorGate struct {
	enabled   bool
	beginUser int
}

func (f *fakeTwoFactorGate) IsEnabled(context.Context, int) (bool, error) {
	return f.enabled, nil
}

func (f *fakeTwoFactorGate) BeginPendingLogin(_ echo.Context, userID int) error {
	f.beginUser = userID
	return nil
}

func TestPostLogin_RedirectsToTwoFactorVerificationWithoutCreatingSession(t *testing.T) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("super-secret"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("GenerateFromPassword() error = %v", err)
	}

	cfg := configForTwoFactorLoginTest()
	authClient := authsupport.NewAuthClient(cfg, fakeLoginAuthStore{
		userRecord: &authsupport.AuthUserRecord{
			UserID:   42,
			Email:    "user@example.com",
			Password: string(passwordHash),
		},
		identity: &authsupport.AuthIdentity{
			UserID:                42,
			UserEmail:             "user@example.com",
			HasProfile:            true,
			ProfileID:             7,
			ProfileFullyOnboarded: true,
		},
	})

	e := echo.New()
	e.Validator = frameworkvalidation.NewValidator()
	e.GET("/auth/2fa/verify", func(c echo.Context) error { return c.NoContent(http.StatusOK) }).Name = routenames.RouteNameTwoFactorVerify

	container := &foundation.Container{
		Config: cfg,
		Auth:   authClient,
		Web:    e,
		Logger: e.Logger,
	}
	twoFactor := &fakeTwoFactorGate{enabled: true}
	service := &Service{
		ctr:       ui.NewController(container),
		twoFactor: twoFactor,
	}

	form := url.Values{}
	form.Set("email", "user@example.com")
	form.Set("password", "super-secret")

	req := httptest.NewRequest(http.MethodPost, "/user/login", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	sessionMiddleware := session.Middleware(sessions.NewCookieStore([]byte("secret")))
	if err := sessionMiddleware(func(c echo.Context) error {
		return service.postLogin(c)
	})(ctx); err != nil {
		t.Fatalf("postLogin() error = %v", err)
	}

	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, rec.Code)
	}
	if rec.Header().Get("Location") != e.Reverse(routenames.RouteNameTwoFactorVerify) {
		t.Fatalf("expected redirect to %q, got %q", e.Reverse(routenames.RouteNameTwoFactorVerify), rec.Header().Get("Location"))
	}
	if twoFactor.beginUser != 42 {
		t.Fatalf("expected pending 2fa user 42, got %d", twoFactor.beginUser)
	}

	authSession, err := session.Get("ua", ctx)
	if err != nil {
		t.Fatalf("session.Get(auth) error = %v", err)
	}
	if authSession.Values["authenticated"] == true {
		t.Fatalf("expected no authenticated session before 2fa verification")
	}
}

func configForTwoFactorLoginTest() *config.Config {
	cfg := &config.Config{}
	cfg.App.EncryptionKey = "test-encryption-key"
	return cfg
}
