package testutil

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"net/url"
)

func TestTestServerPostFormHandlesCSRF(t *testing.T) {
	s := NewTestServer(t)

	password := "password123!"
	_, email := seedAuthUser(t, s, password)

	s.PostForm("/user/login", url.Values{
		"email":    {email},
		"password": {password},
	}).
		AssertStatus(http.StatusFound).
		AssertRedirectsTo("/welcome/preferences")
}

func TestTestServerAsUser(t *testing.T) {
	s := NewTestServer(t)

	userID, _ := seedAuthUser(t, s, "irrelevant")
	s.Get("/auth/logout", s.AsUser(userID)).
		AssertStatus(http.StatusFound).
		AssertRedirectsTo("/")
}

func TestTestResponseAssertions(t *testing.T) {
	t.Run("contains and redirect", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusSeeOther,
			Header:     http.Header{"Location": []string{"/next"}},
			Body:       io.NopCloser(strings.NewReader("hello world")),
		}
		(&TestResponse{Response: resp, t: t}).
			AssertStatus(http.StatusSeeOther).
			AssertRedirectsTo("/next").
			AssertContains("hello")
	})

	t.Run("json", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"message":"ok"}`)),
		}
		var payload struct {
			Message string `json:"message"`
		}
		(&TestResponse{Response: resp, t: t}).AssertStatus(http.StatusOK).AssertJSON(&payload)
		if payload.Message != "ok" {
			t.Fatalf("decoded message = %q, want ok", payload.Message)
		}
	})
}

func seedAuthUser(t *testing.T, s *TestServer, password string) (int64, string) {
	t.Helper()

	hash, err := s.Container.Auth.HashPassword(password)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	email := fmt.Sprintf("testutil-%d@example.com", time.Now().UnixNano())
	res, err := s.Container.Database.Exec(`
		INSERT INTO users (created_at, updated_at, name, email, password, verified)
		VALUES (CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, ?, ?, ?, 1)
	`, "Test User", email, hash)
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("last insert id: %v", err)
	}

	return id, email
}
