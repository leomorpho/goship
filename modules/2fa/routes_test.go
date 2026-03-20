package twofa

import (
	"context"
	"database/sql"
	"encoding/json"
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
	"github.com/leomorpho/goship/modules/authsupport"
	"github.com/pquerna/otp/totp"
)

type fakeTwoFactorAuthStore struct {
	identity *authsupport.AuthIdentity
}

func (f fakeTwoFactorAuthStore) GetIdentityByUserID(context.Context, int) (*authsupport.AuthIdentity, error) {
	if f.identity == nil {
		return nil, sql.ErrNoRows
	}
	return f.identity, nil
}

func (f fakeTwoFactorAuthStore) GetUserRecordByEmail(context.Context, string) (*authsupport.AuthUserRecord, error) {
	return nil, errors.New("not implemented")
}

func (f fakeTwoFactorAuthStore) GetUserDisplayNameByUserID(context.Context, int) (string, error) {
	return "", errors.New("not implemented")
}

func (f fakeTwoFactorAuthStore) UpdateUserDisplayNameByUserID(context.Context, int, string) error {
	return errors.New("not implemented")
}

func (f fakeTwoFactorAuthStore) UpdateUserPasswordHashByUserID(context.Context, int, string) error {
	return errors.New("not implemented")
}

func (f fakeTwoFactorAuthStore) MarkUserVerifiedByUserID(context.Context, int) error {
	return errors.New("not implemented")
}

func (f fakeTwoFactorAuthStore) CreateLastSeenOnline(context.Context, int, time.Time) error {
	return errors.New("not implemented")
}

func (f fakeTwoFactorAuthStore) CreatePasswordToken(context.Context, int, string) (int, error) {
	return 0, errors.New("not implemented")
}

func (f fakeTwoFactorAuthStore) GetPasswordTokenHash(context.Context, int, int, time.Time) (string, error) {
	return "", errors.New("not implemented")
}

func (f fakeTwoFactorAuthStore) DeletePasswordTokens(context.Context, int) error {
	return errors.New("not implemented")
}

func TestPostVerify_CompletesLoginAndRedirectsHome(t *testing.T) {
	cfg := &config.Config{}
	cfg.App.EncryptionKey = "test-encryption-key"

	secret, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "GoShip",
		AccountName: "user@example.com",
	})
	if err != nil {
		t.Fatalf("totp.Generate() error = %v", err)
	}
	code, err := totp.GenerateCode(secret.Secret(), time.Now().UTC())
	if err != nil {
		t.Fatalf("totp.GenerateCode() error = %v", err)
	}

	service := NewService(&fakeStore{
		settings: UserSettings{
			UserID:          42,
			Email:           "user@example.com",
			TOTPEnabled:     true,
			EncryptedSecret: mustEncrypt(t, cfg.App.EncryptionKey, secret.Secret()),
		},
	}, "GoShip", cfg.App.EncryptionKey)

	e := echo.New()
	e.GET("/user/login", func(c echo.Context) error { return c.NoContent(http.StatusOK) }).Name = routenames.RouteNameLogin
	e.GET("/auth/homeFeed", func(c echo.Context) error { return c.NoContent(http.StatusOK) }).Name = routenames.RouteNameHomeFeed
	e.GET("/welcome/preferences", func(c echo.Context) error { return c.NoContent(http.StatusOK) }).Name = routenames.RouteNamePreferences
	e.POST("/auth/2fa/verify", func(c echo.Context) error { return c.NoContent(http.StatusOK) }).Name = routenames.RouteNameTwoFactorVerifySubmit

	authClient := authsupport.NewAuthClient(cfg, fakeTwoFactorAuthStore{
		identity: &authsupport.AuthIdentity{
			UserID:                42,
			UserEmail:             "user@example.com",
			HasProfile:            true,
			ProfileID:             7,
			ProfileFullyOnboarded: true,
		},
	})
	container := &foundation.Container{
		Config: cfg,
		Auth:   authClient,
		Web:    e,
		Logger: e.Logger,
	}

	form := url.Values{}
	form.Set("code", code)

	req := httptest.NewRequest(http.MethodPost, "/auth/2fa/verify", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	mw := session.Middleware(sessions.NewCookieStore([]byte("secret")))
	if err := mw(func(c echo.Context) error {
		if err := SetPendingUserCookie(c, cfg.App.EncryptionKey, 42); err != nil {
			return err
		}
		for _, cookie := range rec.Result().Cookies() {
			c.Request().AddCookie(cookie)
		}
		return postVerify(ui.NewController(container), service)(c)
	})(ctx); err != nil {
		t.Fatalf("postVerify() error = %v", err)
	}

	if rec.Code != http.StatusFound {
		t.Fatalf("expected status %d, got %d", http.StatusFound, rec.Code)
	}
	if rec.Header().Get("Location") != e.Reverse(routenames.RouteNameHomeFeed) {
		t.Fatalf("expected redirect to %q, got %q", e.Reverse(routenames.RouteNameHomeFeed), rec.Header().Get("Location"))
	}

	authSession, err := session.Get("ua", ctx)
	if err != nil {
		t.Fatalf("session.Get(auth) error = %v", err)
	}
	if authSession.Values["authenticated"] != true {
		t.Fatalf("expected authenticated session after 2fa verification")
	}

	clearSeen := false
	for _, raw := range rec.Header().Values("Set-Cookie") {
		if strings.Contains(raw, pendingUserCookieName+"=") {
			clearSeen = true
			break
		}
	}
	if !clearSeen {
		t.Fatalf("expected response to overwrite pending 2fa cookie")
	}
}

func TestPostVerify_AcceptsBackupCodeAndConsumesIt(t *testing.T) {
	cfg := &config.Config{}
	cfg.App.EncryptionKey = "test-encryption-key"

	store := &fakeStore{}
	service := NewService(store, "GoShip", cfg.App.EncryptionKey)
	hash, err := service.HashBackupCode("BK-ABCD-EFGH")
	if err != nil {
		t.Fatalf("HashBackupCode() error = %v", err)
	}
	store.settings = UserSettings{
		UserID:           42,
		Email:            "user@example.com",
		TOTPEnabled:      true,
		EncryptedSecret:  mustEncrypt(t, cfg.App.EncryptionKey, "unused-secret"),
		BackupCodeHashes: []string{hash},
	}

	e := echo.New()
	e.GET("/user/login", func(c echo.Context) error { return c.NoContent(http.StatusOK) }).Name = routenames.RouteNameLogin
	e.GET("/auth/homeFeed", func(c echo.Context) error { return c.NoContent(http.StatusOK) }).Name = routenames.RouteNameHomeFeed
	e.GET("/welcome/preferences", func(c echo.Context) error { return c.NoContent(http.StatusOK) }).Name = routenames.RouteNamePreferences
	e.POST("/auth/2fa/verify", func(c echo.Context) error { return c.NoContent(http.StatusOK) }).Name = routenames.RouteNameTwoFactorVerifySubmit

	authClient := authsupport.NewAuthClient(cfg, fakeTwoFactorAuthStore{
		identity: &authsupport.AuthIdentity{
			UserID:                42,
			UserEmail:             "user@example.com",
			HasProfile:            true,
			ProfileID:             7,
			ProfileFullyOnboarded: true,
		},
	})
	container := &foundation.Container{
		Config: cfg,
		Auth:   authClient,
		Web:    e,
		Logger: e.Logger,
	}

	form := url.Values{}
	form.Set("code", "BK-ABCD-EFGH")

	req := httptest.NewRequest(http.MethodPost, "/auth/2fa/verify", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	rec := httptest.NewRecorder()
	ctx := e.NewContext(req, rec)

	mw := session.Middleware(sessions.NewCookieStore([]byte("secret")))
	if err := mw(func(c echo.Context) error {
		if err := SetPendingUserCookie(c, cfg.App.EncryptionKey, 42); err != nil {
			return err
		}
		for _, cookie := range rec.Result().Cookies() {
			c.Request().AddCookie(cookie)
		}
		return postVerify(ui.NewController(container), service)(c)
	})(ctx); err != nil {
		t.Fatalf("postVerify() error = %v", err)
	}

	if len(store.hashes) != 0 {
		raw, _ := json.Marshal(store.hashes)
		t.Fatalf("expected backup code list to be consumed, got %s", raw)
	}
}
