package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	"github.com/gorilla/sessions"
	"github.com/leomorpho/goship/v2/config"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

const (
	authSessionName             = "ua"
	authSessionKeyUserID        = "user_id"
	authSessionKeyAuthenticated = "authenticated"
)

type authCookie struct {
	Name     string `json:"name"`
	Value    string `json:"value"`
	Path     string `json:"path"`
	HTTPOnly bool   `json:"httpOnly"`
}

func main() {
	email := strings.TrimSpace(os.Getenv("E2E_ADMIN_EMAIL"))
	if email == "" {
		email = "admin@goship.test"
	}
	password := strings.TrimSpace(os.Getenv("E2E_ADMIN_PASSWORD"))
	if password == "" {
		password = "Adminpass12345!"
	}

	cfg, err := config.GetConfig()
	must(err)

	db, err := sql.Open(cfg.Database.EmbeddedDriver, cfg.Database.EmbeddedConnection)
	must(err)
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	userID, err := ensureAdminUser(ctx, db, email, password)
	must(err)

	cookie, err := buildAuthCookie(cfg.App.EncryptionKey, userID)
	must(err)

	must(json.NewEncoder(os.Stdout).Encode(cookie))
}

func ensureAdminUser(ctx context.Context, db *sql.DB, email, password string) (int, error) {
	var userID int
	err := db.QueryRowContext(ctx, "SELECT id FROM users WHERE lower(email) = lower(?)", email).Scan(&userID)
	switch {
	case err == nil:
		return userID, nil
	case !errors.Is(err, sql.ErrNoRows):
		return 0, err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}

	now := time.Now().UTC()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	if err := tx.QueryRowContext(
		ctx,
		`INSERT INTO users (created_at, updated_at, name, email, password, verified)
		 VALUES (?, ?, ?, ?, ?, ?)
		 RETURNING id`,
		now,
		now,
		"Admin User",
		strings.ToLower(email),
		string(passwordHash),
		false,
	).Scan(&userID); err != nil {
		return 0, err
	}

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO profiles (created_at, updated_at, user_profile, fully_onboarded)
		 VALUES (?, ?, ?, ?)`,
		now,
		now,
		userID,
		true,
	); err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return userID, nil
}

func buildAuthCookie(secret string, userID int) (authCookie, error) {
	store := sessions.NewCookieStore([]byte(secret))

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/", nil)
	rec := httptest.NewRecorder()

	sess, err := store.Get(req, authSessionName)
	if err != nil {
		return authCookie{}, err
	}
	sess.Values[authSessionKeyUserID] = userID
	sess.Values[authSessionKeyAuthenticated] = true
	sess.Options.Path = "/"
	sess.Options.HttpOnly = true
	if err := sess.Save(req, rec); err != nil {
		return authCookie{}, err
	}

	for _, cookie := range rec.Result().Cookies() {
		if cookie.Name == authSessionName {
			return authCookie{
				Name:     cookie.Name,
				Value:    cookie.Value,
				Path:     cookie.Path,
				HTTPOnly: cookie.HttpOnly,
			}, nil
		}
	}

	return authCookie{}, errors.New("failed to create auth session cookie")
}

func must(err error) {
	if err == nil {
		return
	}
	panic(err)
}
