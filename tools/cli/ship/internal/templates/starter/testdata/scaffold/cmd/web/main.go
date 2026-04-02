package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/a-h/templ"
	goship "github.com/leomorpho/goship/starter/app"
	starterfoundation "github.com/leomorpho/goship/starter/app/foundation"
	starterpolicies "github.com/leomorpho/goship/starter/app/policies"
	templates "github.com/leomorpho/goship/starter/app/views"
	pages "github.com/leomorpho/goship/starter/app/views/web/pages/gen"
	_ "modernc.org/sqlite"
)

const defaultDatabasePath = "tmp/starter.db"
const starterSessionCookie = "starter_session"

type starterUser struct {
	DisplayName string
	Email       string
	Password    string
	IsAdmin     bool
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

type starterCRUDRecord struct {
	ID     int
	Values map[string]string
}

var starterAuth = &authStore{
	users:    map[string]starterUser{},
	sessions: map[string]string{},
	resets:   map[string]string{},
}

func main() {
	container := starterfoundation.NewContainer()
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	mux.HandleFunc("/dev/mail", mailPreviewIndexHandler)
	mux.HandleFunc("/dev/mail/", mailPreviewShowHandler)
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

	for _, route := range goship.BuildRouter(container) {
		route := route
		mux.HandleFunc(route.Path, func(w http.ResponseWriter, r *http.Request) {
			if err := handleRoute(w, r, route, container); err != nil {
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

func handleRoute(w http.ResponseWriter, r *http.Request, route goship.Route, container *starterfoundation.Container) error {
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
	case "/auth/admin":
		user, ok := currentStarterUser(r)
		if !ok {
			http.Redirect(w, r, "/auth/login?next="+url.QueryEscape(route.Path), http.StatusSeeOther)
			return nil
		}
		if !starterpolicies.AdminDashboardAllows(starterpolicies.PolicyActor{
			Email:   user.Email,
			IsAdmin: user.IsAdmin,
		}) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return nil
		}
		if r.Method == http.MethodPost {
			return mutateAdminResource(w, r, route)
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return nil
		}
		return renderAdminPage(w, r, route)
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
		if route.Path == "/auth/profile" && container != nil && (container.SupportsModule("storage") || container.SupportsModule("emailsubscriptions")) {
			if r.Method == http.MethodPost {
				return profileModuleActionHandler(w, r, container)
			}
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return nil
			}
			return renderProfilePageWithModules(w, r, container)
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return nil
		}
		return renderRoute(w, route.Page)
	default:
		if route.Kind == goship.RouteKindResource {
			return handleStarterCRUDRoute(w, r, route)
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return nil
		}
		return renderRoute(w, route.Page)
	}
}

func renderProfilePageWithModules(w http.ResponseWriter, r *http.Request, container *starterfoundation.Container) error {
	files, err := os.ReadDir(filepath.Join("tmp", "storage"))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	email, _ := currentUser(r)
	subscribed, err := starterEmailSubscriptionStatus(email)
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err = fmt.Fprintf(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Profile</title><link rel="stylesheet" href="/static/styles_bundle.css"></head><body><div class="starter-shell"><header class="starter-header"><div class="starter-brand">GoShip Starter</div></header><section data-component="profile-storage"><h1>Profile</h1>`)
	if err != nil {
		return err
	}
	if container.SupportsModule("storage") {
		if _, err := fmt.Fprint(w, `<h2>Storage sandbox</h2><form method="post" action="/auth/profile" enctype="multipart/form-data"><label>Upload file<input name="storage_upload" type="file"></label><button type="submit">Upload</button></form><ul>`); err != nil {
			return err
		}
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			if _, err := fmt.Fprintf(w, `<li>%s</li>`, html.EscapeString(file.Name())); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(w, `</ul>`); err != nil {
			return err
		}
	}
	if container.SupportsModule("emailsubscriptions") {
		label := "Subscribe"
		action := "subscribe"
		status := "Not subscribed"
		if subscribed {
			label = "Unsubscribe"
			action = "unsubscribe"
			status = "Subscribed"
		}
		if _, err := fmt.Fprintf(w, `<section data-component="email-subscriptions"><h2>Email subscriptions</h2><p data-subscription-status>%s</p><form method="post" action="/auth/profile"><input type="hidden" name="email_subscription_action" value="%s"><button type="submit">%s</button></form></section>`, html.EscapeString(status), html.EscapeString(action), html.EscapeString(label)); err != nil {
			return err
		}
	}
	_, err = fmt.Fprint(w, `</section></div></body></html>`)
	return err
}

func profileModuleActionHandler(w http.ResponseWriter, r *http.Request, container *starterfoundation.Container) error {
	if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") && container.SupportsModule("storage") {
		return profileStorageUploadHandler(w, r)
	}
	if err := r.ParseForm(); err != nil {
		return err
	}
	if container.SupportsModule("emailsubscriptions") {
		email, ok := currentUser(r)
		if !ok {
			http.Redirect(w, r, "/auth/login?next="+url.QueryEscape("/auth/profile"), http.StatusSeeOther)
			return nil
		}
		action := strings.TrimSpace(r.FormValue("email_subscription_action"))
		if action == "subscribe" {
			if err := starterSetEmailSubscription(email, true); err != nil {
				return err
			}
		} else if action == "unsubscribe" {
			if err := starterSetEmailSubscription(email, false); err != nil {
				return err
			}
		}
		http.Redirect(w, r, "/auth/profile", http.StatusSeeOther)
		return nil
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	return nil
}

func profileStorageUploadHandler(w http.ResponseWriter, r *http.Request) error {
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		return err
	}
	file, header, err := r.FormFile("storage_upload")
	if err != nil {
		return err
	}
	defer file.Close()
	name := filepath.Base(strings.TrimSpace(header.Filename))
	if name == "" || name == "." || name == string(filepath.Separator) {
		return fmt.Errorf("invalid upload filename")
	}
	body, err := readMultipartFile(file)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join("tmp", "storage"), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join("tmp", "storage", name), body, 0o644); err != nil {
		return err
	}
	http.Redirect(w, r, "/auth/profile", http.StatusSeeOther)
	return nil
}

func starterEmailSubscriptionStatus(email string) (bool, error) {
	db, err := starterCRUDDB()
	if err != nil {
		return false, err
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS starter_email_subscriptions (
		email TEXT PRIMARY KEY
	)`); err != nil {
		return false, err
	}
	row := db.QueryRow(`SELECT email FROM starter_email_subscriptions WHERE email = ?`, email)
	var existing string
	if err := row.Scan(&existing); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func starterSetEmailSubscription(email string, subscribed bool) error {
	db, err := starterCRUDDB()
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS starter_email_subscriptions (
		email TEXT PRIMARY KEY
	)`); err != nil {
		return err
	}
	if subscribed {
		_, err = db.Exec(`INSERT OR REPLACE INTO starter_email_subscriptions (email) VALUES (?)`, email)
		return err
	}
	_, err = db.Exec(`DELETE FROM starter_email_subscriptions WHERE email = ?`, email)
	return err
}

func readMultipartFile(file multipart.File) ([]byte, error) {
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(file); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
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
	isAdmin := len(starterAuth.users) == 0
	starterAuth.users[email] = starterUser{
		DisplayName: r.FormValue("display_name"),
		Email:       email,
		Password:    password,
		IsAdmin:     isAdmin,
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

func currentStarterUser(r *http.Request) (starterUser, bool) {
	email, ok := currentUser(r)
	if !ok {
		return starterUser{}, false
	}
	starterAuth.mu.Lock()
	defer starterAuth.mu.Unlock()
	user, ok := starterAuth.users[email]
	return user, ok
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

func mailPreviewIndexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/dev/mail" {
		http.NotFound(w, r)
		return
	}
	entries, err := os.ReadDir(filepath.Join("app", "views", "emails"))
	if err != nil && !os.IsNotExist(err) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Mail previews</title><link rel="stylesheet" href="/static/styles_bundle.css"></head><body><div class="starter-shell"><section data-component="mail-preview-index"><h1>Mail previews</h1><ul>`)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".html") {
			continue
		}
		slug := strings.TrimSuffix(entry.Name(), ".html")
		_, _ = fmt.Fprintf(w, `<li><a href="/dev/mail/%s">/dev/mail/%s</a></li>`, html.EscapeString(slug), html.EscapeString(slug))
	}
	_, _ = fmt.Fprint(w, `</ul></section></div></body></html>`)
}

func mailPreviewShowHandler(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/dev/mail/")
	slug = strings.TrimSpace(slug)
	if slug == "" || strings.Contains(slug, "/") {
		http.NotFound(w, r)
		return
	}
	body, err := os.ReadFile(filepath.Join("app", "views", "emails", slug+".html"))
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	content := string(body)
	content = strings.ReplaceAll(content, "{{app_name}}", "GoShip Starter")
	content = strings.ReplaceAll(content, "{{support_email}}", "support@example.com")
	content = strings.ReplaceAll(content, "{{domain}}", "localhost")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprint(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><title>Mail preview</title><link rel="stylesheet" href="/static/styles_bundle.css"></head><body><div class="starter-shell"><section data-component="mail-preview">`)
	_, _ = fmt.Fprint(w, content)
	_, _ = fmt.Fprint(w, `</section></div></body></html>`)
}

func renderAdminPage(w http.ResponseWriter, r *http.Request, route goship.Route) error {
	routes := goship.BuildRouter(nil)
	selected := strings.TrimSpace(r.URL.Query().Get("resource"))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := fmt.Fprintf(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>Admin dashboard</title><link rel="stylesheet" href="/static/styles_bundle.css"></head><body><div class="starter-shell"><header class="starter-header"><div class="starter-brand">GoShip Starter</div></header><section data-component="admin-dashboard"><h1>Admin dashboard</h1><p>Starter backoffice overview for generated resources.</p><ul>`); err != nil {
		return err
	}
	var selectedRoute *goship.Route
	for _, candidate := range routes {
		if candidate.Kind != goship.RouteKindResource {
			continue
		}
		records, err := starterCRUDRecords(candidate)
		if err != nil {
			return err
		}
		if candidate.Name == selected {
			c := candidate
			selectedRoute = &c
		}
		if _, err := fmt.Fprintf(w, `<li><a href="%s?resource=%s">%s</a> <span data-admin-count="%s">%d</span></li>`, route.Path, url.QueryEscape(candidate.Name), html.EscapeString(candidate.Name), html.EscapeString(candidate.Name), len(records)); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprint(w, `</ul>`); err != nil {
		return err
	}
	if selectedRoute != nil {
		records, err := starterCRUDRecords(*selectedRoute)
		if err != nil {
			return err
		}
		fields := starterRouteFields(*selectedRoute)
		if _, err := fmt.Fprintf(w, `<form method="post" action="%s?resource=%s" data-admin-create="%s">`, route.Path, url.QueryEscape(selectedRoute.Name), html.EscapeString(selectedRoute.Name)); err != nil {
			return err
		}
		if err := renderStarterCRUDFields(w, fields, nil); err != nil {
			return err
		}
		if _, err := fmt.Fprint(w, `<button type="submit">Create</button></form>`); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, `<section data-admin-resource="%s"><h2>Admin resource: %s</h2><table><thead><tr>`, html.EscapeString(selectedRoute.Name), html.EscapeString(selectedRoute.Name)); err != nil {
			return err
		}
		for _, field := range fields {
			if _, err := fmt.Fprintf(w, `<th>%s</th>`, html.EscapeString(strings.Title(strings.ReplaceAll(field.Name, "_", " ")))); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(w, `</tr></thead><tbody>`); err != nil {
			return err
		}
		for _, record := range records {
			if _, err := fmt.Fprint(w, `<tr>`); err != nil {
				return err
			}
			for _, field := range fields {
				if _, err := fmt.Fprintf(w, `<td>%s</td>`, html.EscapeString(record.Values[field.Name])); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprintf(w, `<td><form method="post" action="%s?resource=%s&id=%d">`, route.Path, url.QueryEscape(selectedRoute.Name), record.ID); err != nil {
				return err
			}
			if _, err := fmt.Fprint(w, `<input type="hidden" name="_method" value="PUT">`); err != nil {
				return err
			}
			if err := renderStarterCRUDFields(w, fields, record.Values); err != nil {
				return err
			}
			if _, err := fmt.Fprint(w, `<button type="submit">Update</button></form>`); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(w, `<form method="post" action="%s?resource=%s&id=%d"><input type="hidden" name="_method" value="DELETE"><button type="submit">Delete</button></form></td>`, route.Path, url.QueryEscape(selectedRoute.Name), record.ID); err != nil {
				return err
			}
			if _, err := fmt.Fprint(w, `</tr>`); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(w, `</tbody></table></section>`); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintf(w, `<p data-admin-route>%s</p></section></div></body></html>`, html.EscapeString(route.Path))
	return err
}

func mutateAdminResource(w http.ResponseWriter, r *http.Request, adminRoute goship.Route) error {
	resourceName := strings.TrimSpace(r.URL.Query().Get("resource"))
	resource, ok := findStarterResourceRoute(resourceName)
	if !ok {
		http.Error(w, "resource not found", http.StatusNotFound)
		return nil
	}
	if err := r.ParseForm(); err != nil {
		return err
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	method := strings.ToUpper(strings.TrimSpace(r.FormValue("_method")))
	if method == "DELETE" {
		if err := deleteStarterCRUD(resource, id); err != nil {
			return err
		}
		http.Redirect(w, r, adminRoute.Path+"?resource="+url.QueryEscape(resource.Name), http.StatusSeeOther)
		return nil
	}
	values, errs := starterCRUDValuesFromRequest(r, starterRouteFields(resource))
	if len(errs) > 0 {
		writeValidationErrors(w, errs)
		return nil
	}
	if method == "PUT" {
		if err := updateStarterCRUD(resource, id, values); err != nil {
			return err
		}
	} else {
		if _, err := createStarterCRUD(resource, values); err != nil {
			return err
		}
	}
	http.Redirect(w, r, adminRoute.Path+"?resource="+url.QueryEscape(resource.Name), http.StatusSeeOther)
	return nil
}

func findStarterResourceRoute(name string) (goship.Route, bool) {
	for _, route := range goship.BuildRouter(nil) {
		if route.Kind == goship.RouteKindResource && route.Name == name {
			return route, true
		}
	}
	return goship.Route{}, false
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

func handleStarterCRUDRoute(w http.ResponseWriter, r *http.Request, route goship.Route) error {
	switch r.Method {
	case http.MethodGet:
		return renderStarterCRUDPage(w, r, route)
	case http.MethodPost:
		return mutateStarterCRUD(w, r, route)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return nil
	}
}

func renderStarterCRUDPage(w http.ResponseWriter, r *http.Request, route goship.Route) error {
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	if id != "" && !starterRouteSupportsAction(route, "show") && !starterRouteSupportsAction(route, "update") && !starterRouteSupportsAction(route, "destroy") {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return nil
	}
	if id == "" && !starterRouteSupportsAction(route, "index") && !starterRouteSupportsAction(route, "create") {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return nil
	}
	record, hasRecord, err := starterCRUDRecordByID(route, id)
	if err != nil {
		return err
	}
	all, err := starterCRUDRecords(route)
	if err != nil {
		return err
	}
	resourceLabel := strings.ReplaceAll(route.Name, "_", " ")
	fields := starterRouteFields(route)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err = fmt.Fprintf(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>%s CRUD</title><link rel="stylesheet" href="/static/styles_bundle.css"></head><body><div class="starter-shell"><header class="starter-header"><div class="starter-brand">GoShip Starter</div></header><section data-component="%s-crud"><h1>%s CRUD scaffold</h1>`, route.Name, route.Name, resourceLabel)
	if err != nil {
		return err
	}
	if starterRouteSupportsAction(route, "create") {
		if _, err := fmt.Fprintf(w, `<h2>Create %s</h2><form method="post" action="%s">`, resourceLabel, route.Path); err != nil {
			return err
		}
		if err := renderStarterCRUDFields(w, fields, nil); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, `<button type="submit">Create %s</button></form>`, resourceLabel); err != nil {
			return err
		}
	}
	if starterRouteSupportsAction(route, "index") {
		if _, err := fmt.Fprintf(w, `<h2>%s list</h2><ul>`, resourceLabel); err != nil {
			return err
		}
		for _, item := range all {
			if _, err := fmt.Fprintf(w, `<li><a href="%s?id=%d">%s</a></li>`, route.Path, item.ID, html.EscapeString(starterCRUDRecordSummary(item, fields))); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(w, `</ul>`); err != nil {
			return err
		}
	}
	if hasRecord {
		if _, err := fmt.Fprintf(w, `<section data-crud-show="%d">`, record.ID); err != nil {
			return err
		}
		if starterRouteSupportsAction(route, "show") {
			if _, err := fmt.Fprintf(w, `<h2>Show %s</h2>`, resourceLabel); err != nil {
				return err
			}
			if err := renderStarterCRUDRecord(w, fields, record); err != nil {
				return err
			}
		}
		if starterRouteSupportsAction(route, "update") {
			if _, err := fmt.Fprintf(w, `<h2>Edit %s</h2><form method="post" action="%s?id=%d"><input type="hidden" name="_method" value="PUT">`, resourceLabel, route.Path, record.ID); err != nil {
				return err
			}
			if err := renderStarterCRUDFields(w, fields, record.Values); err != nil {
				return err
			}
			if _, err := fmt.Fprintf(w, `<button type="submit">Update %s</button></form>`, resourceLabel); err != nil {
				return err
			}
		}
		if starterRouteSupportsAction(route, "destroy") {
			if _, err := fmt.Fprintf(w, `<form method="post" action="%s?id=%d"><input type="hidden" name="_method" value="DELETE"><button type="submit">Delete %s</button></form>`, route.Path, record.ID, resourceLabel); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(w, `</section>`); err != nil {
			return err
		}
	}
	_, err = fmt.Fprint(w, `</section></div></body></html>`)
	return err
}

func mutateStarterCRUD(w http.ResponseWriter, r *http.Request, route goship.Route) error {
	if err := r.ParseForm(); err != nil {
		return err
	}
	id := strings.TrimSpace(r.URL.Query().Get("id"))
	method := strings.ToUpper(strings.TrimSpace(r.FormValue("_method")))
	fields := starterRouteFields(route)
	if method == "DELETE" {
		if !starterRouteSupportsAction(route, "destroy") {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return nil
		}
		if err := deleteStarterCRUD(route, id); err != nil {
			return err
		}
		http.Redirect(w, r, route.Path, http.StatusSeeOther)
		return nil
	}
	values, errs := starterCRUDValuesFromRequest(r, fields)
	if len(errs) > 0 {
		resourceLabel := strings.ReplaceAll(route.Name, "_", " ")
		if wantsHTMLValidation(r) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusBadRequest)
			_, err := fmt.Fprintf(w, `<!doctype html><html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1"><title>%s CRUD</title><link rel="stylesheet" href="/static/styles_bundle.css"></head><body><div class="starter-shell"><section data-component="%s-crud"><h1>%s CRUD scaffold</h1><form method="post" action="%s">`, route.Name, route.Name, resourceLabel, route.Path)
			if err != nil {
				return err
			}
			if err := renderStarterCRUDFieldsWithErrors(w, fields, values, errs); err != nil {
				return err
			}
			_, err = fmt.Fprintf(w, `<button type="submit">Create %s</button></form></section></div></body></html>`, resourceLabel)
			return err
		}
		writeValidationErrors(w, errs)
		return nil
	}
	switch method {
	case "PUT":
		if !starterRouteSupportsAction(route, "update") {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return nil
		}
		if err := updateStarterCRUD(route, id, values); err != nil {
			return err
		}
		http.Redirect(w, r, route.Path+"?id="+url.QueryEscape(id), http.StatusSeeOther)
		return nil
	default:
		if !starterRouteSupportsAction(route, "create") {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return nil
		}
		newID, err := createStarterCRUD(route, values)
		if err != nil {
			return err
		}
		http.Redirect(w, r, route.Path+"?id="+newID, http.StatusSeeOther)
		return nil
	}
}

func starterRouteSupportsAction(route goship.Route, action string) bool {
	if len(route.Actions) == 0 {
		return true
	}
	for _, existing := range route.Actions {
		if strings.EqualFold(strings.TrimSpace(existing), action) {
			return true
		}
	}
	return false
}

func starterRouteFields(route goship.Route) []goship.RouteField {
	if len(route.Fields) != 0 {
		return append([]goship.RouteField(nil), route.Fields...)
	}
	return []goship.RouteField{{Name: "name", Type: "string"}}
}

func starterCRUDValuesFromRequest(r *http.Request, fields []goship.RouteField) (map[string]string, []validationError) {
	values := make(map[string]string, len(fields))
	var errs []validationError
	for _, field := range fields {
		value := strings.TrimSpace(r.FormValue(field.Name))
		values[field.Name] = value
		if value == "" {
			errs = append(errs, validationErr(field.Name, field.Name+" is required"))
		}
	}
	return values, errs
}

func renderStarterCRUDFields(w http.ResponseWriter, fields []goship.RouteField, values map[string]string) error {
	for _, field := range fields {
		value := ""
		if values != nil {
			value = values[field.Name]
		}
		if _, err := fmt.Fprintf(w, `<label>%s<input name="%s" type="%s" value="%s"></label>`, html.EscapeString(strings.Title(strings.ReplaceAll(field.Name, "_", " "))), html.EscapeString(field.Name), html.EscapeString(starterCRUDInputType(field.Type)), html.EscapeString(value)); err != nil {
			return err
		}
	}
	return nil
}

func renderStarterCRUDFieldsWithErrors(w http.ResponseWriter, fields []goship.RouteField, values map[string]string, errs []validationError) error {
	for _, field := range fields {
		value := values[field.Name]
		if _, err := fmt.Fprintf(w, `<label>%s<input name="%s" type="%s" value="%s"></label>`, html.EscapeString(strings.Title(strings.ReplaceAll(field.Name, "_", " "))), html.EscapeString(field.Name), html.EscapeString(starterCRUDInputType(field.Type)), html.EscapeString(value)); err != nil {
			return err
		}
		if msg := validationMessage(errs, field.Name); msg != "" {
			if _, err := fmt.Fprintf(w, `<p data-validation-for="%s">%s</p>`, html.EscapeString(field.Name), html.EscapeString(msg)); err != nil {
				return err
			}
		}
	}
	return nil
}

func renderStarterCRUDRecord(w http.ResponseWriter, fields []goship.RouteField, record starterCRUDRecord) error {
	if _, err := fmt.Fprint(w, `<dl>`); err != nil {
		return err
	}
	for _, field := range fields {
		value := record.Values[field.Name]
		if _, err := fmt.Fprintf(w, `<dt>%s</dt><dd>%s</dd>`, html.EscapeString(strings.Title(strings.ReplaceAll(field.Name, "_", " "))), html.EscapeString(value)); err != nil {
			return err
		}
	}
	_, err := fmt.Fprint(w, `</dl>`)
	return err
}

func starterCRUDRecordSummary(record starterCRUDRecord, fields []goship.RouteField) string {
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		value := strings.TrimSpace(record.Values[field.Name])
		if value != "" {
			parts = append(parts, value)
		}
	}
	if len(parts) == 0 {
		return fmt.Sprintf("Record %d", record.ID)
	}
	return strings.Join(parts, " · ")
}

func starterCRUDInputType(fieldType string) string {
	switch strings.TrimSpace(strings.ToLower(fieldType)) {
	case "email":
		return "email"
	case "url":
		return "url"
	case "time":
		return "datetime-local"
	default:
		return "text"
	}
}

func createStarterCRUD(route goship.Route, values map[string]string) (string, error) {
	db, err := starterCRUDDB()
	if err != nil {
		return "", err
	}
	defer db.Close()
	table, err := ensureStarterCRUDTable(db, route)
	if err != nil {
		return "", err
	}
	fields := starterRouteFields(route)
	columns := make([]string, 0, len(fields))
	placeholders := make([]string, 0, len(fields))
	args := make([]any, 0, len(fields))
	for _, field := range fields {
		columns = append(columns, field.Name)
		placeholders = append(placeholders, "?")
		args = append(args, values[field.Name])
	}
	res, err := db.Exec(fmt.Sprintf(`INSERT INTO %s (%s) VALUES (%s)`, table, strings.Join(columns, ", "), strings.Join(placeholders, ", ")), args...)
	if err != nil {
		return "", err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", id), nil
}

func updateStarterCRUD(route goship.Route, id string, values map[string]string) error {
	db, err := starterCRUDDB()
	if err != nil {
		return err
	}
	defer db.Close()
	table, err := ensureStarterCRUDTable(db, route)
	if err != nil {
		return err
	}
	fields := starterRouteFields(route)
	assignments := make([]string, 0, len(fields))
	args := make([]any, 0, len(fields)+1)
	for _, field := range fields {
		assignments = append(assignments, field.Name+" = ?")
		args = append(args, values[field.Name])
	}
	args = append(args, id)
	_, err = db.Exec(fmt.Sprintf(`UPDATE %s SET %s WHERE record_id = ?`, table, strings.Join(assignments, ", ")), args...)
	return err
}

func deleteStarterCRUD(route goship.Route, id string) error {
	db, err := starterCRUDDB()
	if err != nil {
		return err
	}
	defer db.Close()
	table, err := ensureStarterCRUDTable(db, route)
	if err != nil {
		return err
	}
	_, err = db.Exec(fmt.Sprintf(`DELETE FROM %s WHERE record_id = ?`, table), id)
	return err
}

func starterCRUDRecordByID(route goship.Route, id string) (starterCRUDRecord, bool, error) {
	if strings.TrimSpace(id) == "" {
		return starterCRUDRecord{}, false, nil
	}
	db, err := starterCRUDDB()
	if err != nil {
		return starterCRUDRecord{}, false, err
	}
	defer db.Close()
	table, err := ensureStarterCRUDTable(db, route)
	if err != nil {
		return starterCRUDRecord{}, false, err
	}
	fields := starterRouteFields(route)
	columns := []string{"record_id"}
	for _, field := range fields {
		columns = append(columns, field.Name)
	}
	row := db.QueryRow(fmt.Sprintf(`SELECT %s FROM %s WHERE record_id = ?`, strings.Join(columns, ", "), table), id)
	record, err := scanStarterCRUDRecord(row, fields)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return starterCRUDRecord{}, false, nil
		}
		return starterCRUDRecord{}, false, err
	}
	return record, true, nil
}

func starterCRUDRecords(route goship.Route) ([]starterCRUDRecord, error) {
	db, err := starterCRUDDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	table, err := ensureStarterCRUDTable(db, route)
	if err != nil {
		return nil, err
	}
	fields := starterRouteFields(route)
	columns := []string{"record_id"}
	for _, field := range fields {
		columns = append(columns, field.Name)
	}
	rows, err := db.Query(fmt.Sprintf(`SELECT %s FROM %s ORDER BY record_id ASC`, strings.Join(columns, ", "), table))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []starterCRUDRecord
	for rows.Next() {
		record, err := scanStarterCRUDRows(rows, fields)
		if err != nil {
			return nil, err
		}
		out = append(out, record)
	}
	return out, rows.Err()
}

func starterCRUDDB() (*sql.DB, error) {
	if _, err := os.Stat(defaultDatabasePath); err != nil {
		return nil, fmt.Errorf("starter CRUD requires %s; run ship db:migrate first", filepath.ToSlash(defaultDatabasePath))
	}
	db, err := sql.Open("sqlite", "file:"+filepath.ToSlash(defaultDatabasePath))
	if err != nil {
		return nil, err
	}
	return db, nil
}

func ensureStarterCRUDTable(db *sql.DB, route goship.Route) (string, error) {
	tableName := strings.TrimSpace(route.StorageTable)
	if tableName == "" {
		return "", fmt.Errorf("starter CRUD route %q is missing storage table metadata", route.Name)
	}
	for _, r := range tableName {
		if !(r == '_' || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')) {
			return "", fmt.Errorf("starter CRUD route %q has invalid storage table %q", route.Name, tableName)
		}
	}
	table := "starter_" + tableName
	columnDefs := make([]string, 0, len(starterRouteFields(route)))
	for _, field := range starterRouteFields(route) {
		columnDefs = append(columnDefs, fmt.Sprintf("%s %s NOT NULL", field.Name, starterCRUDSQLiteType(field.Type)))
	}
	if _, err := db.Exec(fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		record_id INTEGER PRIMARY KEY AUTOINCREMENT,
		%s
	)`, table, strings.Join(columnDefs, ",\n\t\t"))); err != nil {
		return "", err
	}
	return table, nil
}

func scanStarterCRUDRecord(row *sql.Row, fields []goship.RouteField) (starterCRUDRecord, error) {
	recordID := 0
	dest, values := starterCRUDScanDest(fields, &recordID)
	if err := row.Scan(dest...); err != nil {
		return starterCRUDRecord{}, err
	}
	return starterCRUDRecord{ID: recordID, Values: starterCRUDValueMap(fields, values)}, nil
}

func scanStarterCRUDRows(rows *sql.Rows, fields []goship.RouteField) (starterCRUDRecord, error) {
	recordID := 0
	dest, values := starterCRUDScanDest(fields, &recordID)
	if err := rows.Scan(dest...); err != nil {
		return starterCRUDRecord{}, err
	}
	return starterCRUDRecord{ID: recordID, Values: starterCRUDValueMap(fields, values)}, nil
}

func starterCRUDScanDest(fields []goship.RouteField, recordID *int) ([]any, map[string]*string) {
	dest := []any{recordID}
	values := map[string]*string{}
	for _, field := range fields {
		value := ""
		dest = append(dest, &value)
		values[field.Name] = &value
	}
	return dest, values
}

func starterCRUDValueMap(fields []goship.RouteField, values map[string]*string) map[string]string {
	out := make(map[string]string, len(fields))
	for _, field := range fields {
		if ptr := values[field.Name]; ptr != nil {
			out[field.Name] = *ptr
		}
	}
	return out
}

func starterCRUDSQLiteType(fieldType string) string {
	switch strings.TrimSpace(strings.ToLower(fieldType)) {
	case "int":
		return "INTEGER"
	case "bool":
		return "BOOLEAN"
	case "float":
		return "REAL"
	default:
		return "TEXT"
	}
}

func titleize(routeName string) string {
	name := strings.ReplaceAll(routeName, "_", " ")
	return strings.Title(name)
}
