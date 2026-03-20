package testutil

import (
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/gorilla/sessions"
)

const (
	authSessionName             = "ua"
	authSessionKeyUserID        = "user_id"
	authSessionKeyAuthenticated = "authenticated"
)

func (s *TestServer) AsUser(userID int64) RequestOpt {
	return func(cfg *requestConfig) error {
		if s == nil || s.Container == nil || s.Container.Config == nil {
			return errors.New("test server config is nil")
		}
		cookie, err := authSessionCookie(s.Container.Config.App.EncryptionKey, userID)
		if err != nil {
			return err
		}
		cfg.cookies = append(cfg.cookies, cookie)
		return nil
	}
}

func authSessionCookie(secret string, userID int64) (*http.Cookie, error) {
	if secret == "" {
		return nil, errors.New("empty app encryption key")
	}
	store := sessions.NewCookieStore([]byte(secret))

	req := httptest.NewRequest(http.MethodGet, "http://example.test/", nil)
	rec := httptest.NewRecorder()

	sess, err := store.Get(req, authSessionName)
	if err != nil {
		return nil, err
	}
	sess.Values[authSessionKeyUserID] = int(userID)
	sess.Values[authSessionKeyAuthenticated] = true
	if err := sess.Save(req, rec); err != nil {
		return nil, err
	}

	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == authSessionName {
			return cookie, nil
		}
	}
	return nil, errors.New("failed to create auth session cookie")
}
