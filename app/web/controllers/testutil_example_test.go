package controllers_test

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/leomorpho/goship/framework/testutil"
)

func TestLoginFlowWithHTTPTestutil(t *testing.T) {
	s := testutil.NewTestServer(t)

	password := "password123!"
	userID, email := seedAuthUserForHTTPTest(t, s, password)

	s.PostForm("/user/login", url.Values{
		"email":    {email},
		"password": {password},
	}).AssertRedirectsTo("/welcome/preferences")

	s.Get("/auth/logout", s.AsUser(userID)).AssertRedirectsTo("/")
}

func TestLoginPageUsesFrenchWhenAcceptLanguageIsFrench(t *testing.T) {
	s := testutil.NewTestServer(t)
	s.Get("/user/login", testutil.WithHeader("Accept-Language", "fr-FR,fr;q=0.9")).
		AssertStatus(200).
		AssertContains("Connectez-vous a votre compte")
}

func seedAuthUserForHTTPTest(t *testing.T, s *testutil.TestServer, password string) (int64, string) {
	t.Helper()

	hash, err := s.Container.Auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	email := fmt.Sprintf("app-testutil-%d@example.com", time.Now().UnixNano())
	res, err := s.Container.Database.Exec(`
		INSERT INTO users (created_at, updated_at, name, email, password, verified)
		VALUES (CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?, ?, ?, 1)
	`, "App Test User", email, hash)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}

	return id, email
}
