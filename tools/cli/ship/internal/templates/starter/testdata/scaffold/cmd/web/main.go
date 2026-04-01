package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/a-h/templ"
	goship "github.com/leomorpho/goship/starter/app"
	templates "github.com/leomorpho/goship/starter/app/views"
	pages "github.com/leomorpho/goship/starter/app/views/web/pages/gen"
)

const defaultDatabasePath = "tmp/starter.db"
const starterSessionCookie = "starter_session"

type authStore struct {
	mu       sync.Mutex
	users    map[string]string
	sessions map[string]string
}

var starterAuth = &authStore{
	users:    map[string]string{},
	sessions: map[string]string{},
}

func main() {
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/up", func(w http.ResponseWriter, _ *http.Request) {
		writeText(w, http.StatusOK, "alive")
	})
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		writeText(w, http.StatusOK, "alive")
	})
	mux.HandleFunc("/healthz", readinessHandler)
	mux.HandleFunc("/health/liveness", func(w http.ResponseWriter, _ *http.Request) {
		writeText(w, http.StatusOK, "alive")
	})
	mux.HandleFunc("/health/readiness", readinessHandler)
	mux.HandleFunc("/auth/logout", logoutHandler)

	for _, route := range goship.BuildRouter(nil) {
		route := route
		mux.HandleFunc(route.Path, func(w http.ResponseWriter, r *http.Request) {
			if err := handleRoute(w, r, route); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})
	}

	addr := ":" + envOrDefault("PORT", "3000")
	log.Printf("starter web listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func readinessHandler(w http.ResponseWriter, _ *http.Request) {
	if _, err := os.Stat(defaultDatabasePath); err != nil {
		http.Error(w, "not ready: run ship db:migrate", http.StatusServiceUnavailable)
		return
	}
	writeText(w, http.StatusOK, "ready")
}

func handleRoute(w http.ResponseWriter, r *http.Request, route goship.Route) error {
	switch route.Path {
	case "/auth/register":
		if r.Method == http.MethodPost {
			return registerHandler(w, r)
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return nil
		}
		return renderAuthPage(w, "Register", "register", "/auth/register", "", "Register")
	case "/auth/login":
		if r.Method == http.MethodPost {
			return loginHandler(w, r)
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return nil
		}
		return renderAuthPage(w, "Log in", "login", "/auth/login", r.URL.Query().Get("next"), "Log in")
	case "/auth/homeFeed", "/auth/profile":
		if _, ok := currentUser(r); !ok {
			http.Redirect(w, r, "/auth/login?next="+url.QueryEscape(route.Path), http.StatusSeeOther)
			return nil
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return nil
		}
		return renderRoute(w, route.Page)
	default:
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return nil
		}
		return renderRoute(w, route.Page)
	}
}

func renderRoute(w http.ResponseWriter, page templates.Page) error {
	component, title := componentForPage(page)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := fmt.Fprintf(w, "<!doctype html><html lang=\"en\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"><title>%s</title><link rel=\"stylesheet\" href=\"/static/styles_bundle.css\"></head><body><div class=\"starter-shell\"><header class=\"starter-header\"><div class=\"starter-brand\">GoShip Starter</div><nav class=\"starter-nav\"><a href=\"/\">Landing</a><a href=\"/auth/login\">Login</a><a href=\"/auth/register\">Register</a><a href=\"/auth/homeFeed\">Home Feed</a><a href=\"/auth/profile\">Profile</a></nav></header>", title); err != nil {
		return err
	}
	if err := component.Render(context.Background(), w); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "<footer class=\"starter-footer\">Database path: %s</footer></div></body></html>", filepath.ToSlash(defaultDatabasePath))
	return err
}

func componentForPage(page templates.Page) (templ.Component, string) {
	switch page {
	case templates.PageHomeFeed:
		return pages.HomeFeed(), "Home Feed"
	case templates.PageProfile:
		return pages.Profile(), "Profile"
	default:
		return pages.Landing(), "GoShip Starter"
	}
}

func envOrDefault(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func writeText(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func renderAuthPage(w http.ResponseWriter, title, component, action, next, submitLabel string) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err := fmt.Fprintf(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>%s</title><link rel="stylesheet" href="/static/styles_bundle.css"></head><body><div class="starter-shell"><header class="starter-header"><div class="starter-brand">GoShip Starter</div></header><section data-component="%s"><h1>%s</h1><form method="post" action="%s" data-component="%s"><label>Display Name<input name="display_name" type="text"></label><label>Email address<input name="email" type="email"></label><label>Password<input name="password" type="password"></label><label>Birthdate (you need to be 18)<input id="birthdate" name="birthdate" type="date"></label><input name="relationship_status" type="hidden" value="single"><input name="next" type="hidden" value="%s"><button id="login-button" type="submit">%s</button></form></section></div></body></html>`, title, component, title, action, component, next, submitLabel)
	return err
}

func registerHandler(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	email := r.FormValue("email")
	password := r.FormValue("password")
	if email == "" || password == "" {
		http.Error(w, "email and password are required", http.StatusBadRequest)
		return nil
	}
	starterAuth.mu.Lock()
	starterAuth.users[email] = password
	starterAuth.mu.Unlock()
	return startSessionAndRedirect(w, r, email, "/auth/profile")
}

func loginHandler(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	email := r.FormValue("email")
	password := r.FormValue("password")
	next := r.FormValue("next")
	if next == "" {
		next = "/auth/profile"
	}
	starterAuth.mu.Lock()
	storedPassword, ok := starterAuth.users[email]
	starterAuth.mu.Unlock()
	if !ok || storedPassword != password {
		http.Error(w, "invalid credentials", http.StatusUnauthorized)
		return nil
	}
	return startSessionAndRedirect(w, r, email, next)
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(starterSessionCookie); err == nil {
		starterAuth.mu.Lock()
		delete(starterAuth.sessions, cookie.Value)
		starterAuth.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: starterSessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
	http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
}

func startSessionAndRedirect(w http.ResponseWriter, r *http.Request, email, target string) error {
	token, err := sessionToken()
	if err != nil {
		return err
	}
	starterAuth.mu.Lock()
	starterAuth.sessions[token] = email
	starterAuth.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: starterSessionCookie, Value: token, Path: "/", HttpOnly: true})
	http.Redirect(w, r, target, http.StatusSeeOther)
	return nil
}

func currentUser(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(starterSessionCookie)
	if err != nil || cookie.Value == "" {
		return "", false
	}
	starterAuth.mu.Lock()
	defer starterAuth.mu.Unlock()
	email, ok := starterAuth.sessions[cookie.Value]
	return email, ok
}

func sessionToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
