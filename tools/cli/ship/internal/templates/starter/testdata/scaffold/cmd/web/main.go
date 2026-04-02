package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/a-h/templ"
	goship "github.com/leomorpho/goship/starter/app"
	templates "github.com/leomorpho/goship/starter/app/views"
	pages "github.com/leomorpho/goship/starter/app/views/web/pages/gen"
)

const defaultDatabasePath = "tmp/starter.db"
const starterSessionCookie = "starter_session"

type starterUser struct {
	DisplayName string
	Email       string
	Password    string
}

type validationError struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
	Code    string `json:"code"`
}

type authStore struct {
	mu       sync.Mutex
	users    map[string]starterUser
	sessions map[string]string
	resets   map[string]string
}

var starterAuth = &authStore{
	users:    map[string]starterUser{},
	sessions: map[string]string{},
	resets:   map[string]string{},
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
	case "/auth/password/reset":
		if r.Method == http.MethodPost {
			return passwordResetRequestHandler(w, r)
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return nil
		}
		return renderSimpleFormPage(w, "Reset password", "password-reset", "/auth/password/reset", "", "Request reset link", []formField{
			{Name: "email", Label: "Email address", Type: "email", Value: ""},
		})
	case "/auth/password/reset/confirm":
		if r.Method == http.MethodPost {
			return passwordResetConfirmHandler(w, r)
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return nil
		}
		return renderPasswordResetConfirmPage(w, r)
	case "/auth/session":
		return sessionHandler(w, r)
	case "/auth/settings":
		if _, ok := currentUser(r); !ok {
			http.Redirect(w, r, "/auth/login?next="+url.QueryEscape(route.Path), http.StatusSeeOther)
			return nil
		}
		if r.Method == http.MethodPost {
			return settingsHandler(w, r)
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return nil
		}
		return renderSettingsPage(w, r)
	case "/auth/delete-account":
		if _, ok := currentUser(r); !ok {
			http.Redirect(w, r, "/auth/login?next="+url.QueryEscape(route.Path), http.StatusSeeOther)
			return nil
		}
		if r.Method == http.MethodPost {
			return deleteAccountHandler(w, r)
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return nil
		}
		return renderDeleteAccountPage(w, r)
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

type formField struct {
	Name  string
	Label string
	Type  string
	Value string
	Error string
}

func renderSimpleFormPage(w http.ResponseWriter, title, component, action, next, submitLabel string, fields []formField) error {
	return renderSimpleFormPageStatus(w, http.StatusOK, title, component, action, next, submitLabel, fields)
}

func renderSimpleFormPageStatus(w http.ResponseWriter, status int, title, component, action, next, submitLabel string, fields []formField) error {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, err := fmt.Fprintf(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>%s</title><link rel="stylesheet" href="/static/styles_bundle.css"></head><body><div class="starter-shell"><header class="starter-header"><div class="starter-brand">GoShip Starter</div></header><section data-component="%s"><h1>%s</h1><form method="post" action="%s" data-component="%s">`, title, component, title, action, component)
	if err != nil {
		return err
	}
	for _, field := range fields {
		if field.Type == "hidden" {
			if _, err := fmt.Fprintf(w, `<input name="%s" type="hidden" value="%s">`, html.EscapeString(field.Name), html.EscapeString(field.Value)); err != nil {
				return err
			}
			continue
		}
		if _, err := fmt.Fprintf(w, `<label>%s<input name="%s" type="%s" value="%s"></label>`, html.EscapeString(field.Label), html.EscapeString(field.Name), html.EscapeString(field.Type), html.EscapeString(field.Value)); err != nil {
			return err
		}
		if field.Error != "" {
			if _, err := fmt.Fprintf(w, `<p data-validation-for="%s">%s</p>`, html.EscapeString(field.Name), html.EscapeString(field.Error)); err != nil {
				return err
			}
		}
	}
	if _, err := fmt.Fprintf(w, `<input name="next" type="hidden" value="%s"><button type="submit">%s</button></form></section></div></body></html>`, next, submitLabel); err != nil {
		return err
	}
	return nil
}

func registerHandler(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	if errs := requireFormFields(r, map[string]string{
		"display_name": "display name is required",
		"email":        "email is required",
		"password":     "password is required",
	}); len(errs) > 0 {
		if wantsHTMLValidation(r) {
			return renderSimpleFormPageStatus(w, http.StatusBadRequest, "Register", "register", "/auth/register", r.FormValue("next"), "Register", []formField{
				{Name: "display_name", Label: "Display Name", Type: "text", Value: r.FormValue("display_name"), Error: validationMessage(errs, "display_name")},
				{Name: "email", Label: "Email address", Type: "email", Value: r.FormValue("email"), Error: validationMessage(errs, "email")},
				{Name: "password", Label: "Password", Type: "password", Value: "", Error: validationMessage(errs, "password")},
				{Name: "birthdate", Label: "Birthdate (you need to be 18)", Type: "date", Value: r.FormValue("birthdate")},
				{Name: "relationship_status", Type: "hidden", Value: r.FormValue("relationship_status")},
			})
		}
		writeValidationErrors(w, errs)
		return nil
	}
	email := r.FormValue("email")
	password := r.FormValue("password")
	starterAuth.mu.Lock()
	starterAuth.users[email] = starterUser{
		DisplayName: r.FormValue("display_name"),
		Email:       email,
		Password:    password,
	}
	starterAuth.mu.Unlock()
	return startSessionAndRedirect(w, r, email, "/auth/profile")
}

func loginHandler(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	if errs := requireFormFields(r, map[string]string{
		"email":    "email is required",
		"password": "password is required",
	}); len(errs) > 0 {
		if wantsHTMLValidation(r) {
			return renderSimpleFormPageStatus(w, http.StatusBadRequest, "Log in", "login", "/auth/login", r.FormValue("next"), "Log in", []formField{
				{Name: "email", Label: "Email address", Type: "email", Value: r.FormValue("email"), Error: validationMessage(errs, "email")},
				{Name: "password", Label: "Password", Type: "password", Value: "", Error: validationMessage(errs, "password")},
			})
		}
		writeValidationErrors(w, errs)
		return nil
	}
	email := r.FormValue("email")
	password := r.FormValue("password")
	next := r.FormValue("next")
	if next == "" {
		next = "/auth/profile"
	}
	starterAuth.mu.Lock()
	user, ok := starterAuth.users[email]
	starterAuth.mu.Unlock()
	if !ok || user.Password != password {
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

func sessionHandler(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return nil
	}
	// Starter note: keep the session surface same-origin and simple. Anonymous
	// callers follow the same login redirect semantics as other protected pages
	// instead of receiving a separate unauthenticated JSON contract.
	email, ok := currentUser(r)
	if !ok {
		http.Redirect(w, r, "/auth/login?next="+url.QueryEscape("/auth/session"), http.StatusSeeOther)
		return nil
	}
	starterAuth.mu.Lock()
	user := starterAuth.users[email]
	starterAuth.mu.Unlock()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(map[string]string{
		"email":        user.Email,
		"display_name": user.DisplayName,
	})
}

func renderSettingsPage(w http.ResponseWriter, r *http.Request) error {
	email, ok := currentUser(r)
	if !ok {
		http.Redirect(w, r, "/auth/login?next="+url.QueryEscape("/auth/settings"), http.StatusSeeOther)
		return nil
	}
	starterAuth.mu.Lock()
	user := starterAuth.users[email]
	starterAuth.mu.Unlock()
	return renderSimpleFormPage(w, "Account settings", "account-settings", "/auth/settings", "", "Save settings", []formField{
		{Name: "display_name", Label: "Display Name", Type: "text", Value: user.DisplayName},
	})
}

func settingsHandler(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	email, ok := currentUser(r)
	if !ok {
		http.Redirect(w, r, "/auth/login?next="+url.QueryEscape("/auth/settings"), http.StatusSeeOther)
		return nil
	}
	if errs := requireFormFields(r, map[string]string{
		"display_name": "display name is required",
	}); len(errs) > 0 {
		if wantsHTMLValidation(r) {
			return renderSimpleFormPageStatus(w, http.StatusBadRequest, "Account settings", "account-settings", "/auth/settings", "", "Save settings", []formField{
				{Name: "display_name", Label: "Display Name", Type: "text", Value: r.FormValue("display_name"), Error: validationMessage(errs, "display_name")},
			})
		}
		writeValidationErrors(w, errs)
		return nil
	}
	displayName := r.FormValue("display_name")
	starterAuth.mu.Lock()
	user := starterAuth.users[email]
	user.DisplayName = displayName
	starterAuth.users[email] = user
	starterAuth.mu.Unlock()
	http.Redirect(w, r, "/auth/settings", http.StatusSeeOther)
	return nil
}

func passwordResetRequestHandler(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	if errs := requireFormFields(r, map[string]string{
		"email": "email is required",
	}); len(errs) > 0 {
		if wantsHTMLValidation(r) {
			return renderSimpleFormPageStatus(w, http.StatusBadRequest, "Reset password", "password-reset", "/auth/password/reset", "", "Request reset link", []formField{
				{Name: "email", Label: "Email address", Type: "email", Value: r.FormValue("email"), Error: validationMessage(errs, "email")},
			})
		}
		writeValidationErrors(w, errs)
		return nil
	}
	email := r.FormValue("email")
	starterAuth.mu.Lock()
	if _, ok := starterAuth.users[email]; ok {
		starterAuth.resets[email] = resetTokenForEmail(email)
	}
	starterAuth.mu.Unlock()
	http.Redirect(w, r, "/auth/password/reset/confirm?email="+url.QueryEscape(email), http.StatusSeeOther)
	return nil
}

func renderPasswordResetConfirmPage(w http.ResponseWriter, r *http.Request) error {
	email := r.URL.Query().Get("email")
	starterAuth.mu.Lock()
	token := starterAuth.resets[email]
	starterAuth.mu.Unlock()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err := fmt.Fprintf(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Reset password</title><link rel="stylesheet" href="/static/styles_bundle.css"></head><body><div class="starter-shell"><header class="starter-header"><div class="starter-brand">GoShip Starter</div></header><section data-component="password-reset-confirm"><h1>Reset password</h1><p data-reset-token>%s</p><form method="post" action="/auth/password/reset/confirm" data-component="password-reset-confirm"><label>Email address<input name="email" type="email" value="%s"></label><label>Reset token<input name="token" type="text" value="%s"></label><label>New password<input name="password" type="password"></label><button type="submit">Reset password</button></form></section></div></body></html>`, token, email, token)
	return err
}

func passwordResetConfirmHandler(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	if errs := requireFormFields(r, map[string]string{
		"email":    "email is required",
		"token":    "token is required",
		"password": "password is required",
	}); len(errs) > 0 {
		if wantsHTMLValidation(r) {
			return renderSimpleFormPageStatus(w, http.StatusBadRequest, "Reset password", "password-reset-confirm", "/auth/password/reset/confirm", "", "Reset password", []formField{
				{Name: "email", Label: "Email address", Type: "email", Value: r.FormValue("email"), Error: validationMessage(errs, "email")},
				{Name: "token", Label: "Reset token", Type: "text", Value: r.FormValue("token"), Error: validationMessage(errs, "token")},
				{Name: "password", Label: "New password", Type: "password", Value: "", Error: validationMessage(errs, "password")},
			})
		}
		writeValidationErrors(w, errs)
		return nil
	}
	email := r.FormValue("email")
	token := r.FormValue("token")
	password := r.FormValue("password")
	starterAuth.mu.Lock()
	defer starterAuth.mu.Unlock()
	expected := starterAuth.resets[email]
	user, ok := starterAuth.users[email]
	if !ok || expected == "" || token != expected {
		http.Error(w, "invalid reset token", http.StatusUnauthorized)
		return nil
	}
	user.Password = password
	starterAuth.users[email] = user
	delete(starterAuth.resets, email)
	for sessionToken, sessionEmail := range starterAuth.sessions {
		if sessionEmail == email {
			delete(starterAuth.sessions, sessionToken)
		}
	}
	http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
	return nil
}

func renderDeleteAccountPage(w http.ResponseWriter, r *http.Request) error {
	email, ok := currentUser(r)
	if !ok {
		http.Redirect(w, r, "/auth/login?next="+url.QueryEscape("/auth/delete-account"), http.StatusSeeOther)
		return nil
	}
	return renderSimpleFormPage(w, "Delete account", "delete-account", "/auth/delete-account", "", "Delete account now", []formField{
		{Name: "email", Label: "Email address", Type: "email", Value: email},
	})
}

func deleteAccountHandler(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	email, ok := currentUser(r)
	if !ok {
		http.Redirect(w, r, "/auth/login?next="+url.QueryEscape("/auth/delete-account"), http.StatusSeeOther)
		return nil
	}
	if errs := requireFormFields(r, map[string]string{
		"email": "email confirmation is required",
	}); len(errs) > 0 {
		if wantsHTMLValidation(r) {
			return renderSimpleFormPageStatus(w, http.StatusBadRequest, "Delete account", "delete-account", "/auth/delete-account", "", "Delete account now", []formField{
				{Name: "email", Label: "Email address", Type: "email", Value: r.FormValue("email"), Error: validationMessage(errs, "email")},
			})
		}
		writeValidationErrors(w, errs)
		return nil
	}
	if r.FormValue("email") != email {
		errs := []validationError{validationErr("email", "email confirmation mismatch")}
		if wantsHTMLValidation(r) {
			return renderSimpleFormPageStatus(w, http.StatusBadRequest, "Delete account", "delete-account", "/auth/delete-account", "", "Delete account now", []formField{
				{Name: "email", Label: "Email address", Type: "email", Value: r.FormValue("email"), Error: validationMessage(errs, "email")},
			})
		}
		writeValidationErrors(w, errs)
		return nil
	}
	starterAuth.mu.Lock()
	delete(starterAuth.users, email)
	delete(starterAuth.resets, email)
	for sessionToken, sessionEmail := range starterAuth.sessions {
		if sessionEmail == email {
			delete(starterAuth.sessions, sessionToken)
		}
	}
	starterAuth.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: starterSessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
	http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
	return nil
}

func sessionToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func resetTokenForEmail(email string) string {
	// Starter note: this deterministic token keeps the password-reset proof
	// no-infra and testable. It is intentionally a starter-only simplification,
	// not a production-grade reset-token pattern.
	return "reset-" + hex.EncodeToString([]byte(email))
}

func requireFormFields(r *http.Request, fields map[string]string) []validationError {
	errs := make([]validationError, 0, len(fields))
	for field, message := range fields {
		if r.FormValue(field) == "" {
			errs = append(errs, validationErr(field, message))
		}
	}
	return errs
}

func validationErr(field, message string) validationError {
	return validationError{
		Field:   field,
		Message: message,
		Code:    "validation_error",
	}
}

func writeValidationErrors(w http.ResponseWriter, errs []validationError) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"errors": errs,
	})
}

func validationMessage(errs []validationError, field string) string {
	for _, err := range errs {
		if err.Field == field {
			return err.Message
		}
	}
	return ""
}

func wantsHTMLValidation(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	return accept != "" && !strings.Contains(accept, "application/json") && strings.Contains(accept, "text/html")
}
