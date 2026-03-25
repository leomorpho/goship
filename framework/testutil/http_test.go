package testutil

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"path/filepath"
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

	t.Run("sse event", func(t *testing.T) {
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader("event: message\ndata: hello\n\nevent: done\ndata: complete\n\n")),
		}
		(&TestResponse{Response: resp, t: t}).
			AssertStatus(http.StatusOK).
			AssertSSEEvent("message", "hello").
			AssertSSEEvent("done", "complete")
	})
}

func TestTestServerPostJSON(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("content-type = %q, want application/json", got)
		}
		var payload struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"name": payload.Name})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookie jar: %v", err)
	}
	s := &TestServer{
		Server: srv,
		t:      t,
		client: &http.Client{Jar: jar},
	}

	var body struct {
		Name string `json:"name"`
	}
	s.PostJSON("/json", map[string]string{"name": "api"}).
		AssertStatus(http.StatusOK).
		AssertJSON(&body)
	if body.Name != "api" {
		t.Fatalf("response name = %q, want api", body.Name)
	}
}

func TestTestServerPostMultipart(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data; boundary=") {
			t.Fatalf("unexpected content-type: %q", r.Header.Get("Content-Type"))
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("parse multipart form: %v", err)
		}
		if got := r.FormValue("title"); got != "avatar" {
			t.Fatalf("title = %q, want avatar", got)
		}
		file, header, err := r.FormFile("avatar")
		if err != nil {
			t.Fatalf("form file: %v", err)
		}
		defer file.Close()
		if header.Filename != "avatar.txt" {
			t.Fatalf("filename = %q, want avatar.txt", header.Filename)
		}
		content, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("read uploaded file: %v", err)
		}
		if string(content) != "hello-avatar" {
			t.Fatalf("file content = %q, want hello-avatar", string(content))
		}
		w.WriteHeader(http.StatusCreated)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatalf("cookie jar: %v", err)
	}
	s := &TestServer{
		Server: srv,
		t:      t,
		client: &http.Client{Jar: jar},
	}

	filePath := filepath.Join(t.TempDir(), "avatar.txt")
	if err := os.WriteFile(filePath, []byte("hello-avatar"), 0o644); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}

	s.PostMultipart("/upload", map[string]string{"title": "avatar"}, []MultipartFile{
		{FieldName: "avatar", FileName: "avatar.txt", Path: filePath},
	}).
		AssertStatus(http.StatusCreated)
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
