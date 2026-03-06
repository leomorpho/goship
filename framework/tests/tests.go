package tests

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"golang.org/x/exp/rand"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

// NewContext creates a new Echo context for tests using an HTTP test request and response recorder
func NewContext(e *echo.Echo, url string) (echo.Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(http.MethodGet, url, strings.NewReader(""))
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

// InitSession initializes a session for a given Echo context
func InitSession(ctx echo.Context) {
	mw := session.Middleware(sessions.NewCookieStore([]byte("secret")))
	_ = ExecuteMiddleware(ctx, mw)
}

// ExecuteMiddleware executes a middleware function on a given Echo context
func ExecuteMiddleware(ctx echo.Context, mw echo.MiddlewareFunc) error {
	handler := mw(func(c echo.Context) error {
		return nil
	})
	return handler(ctx)
}

// AssertHTTPErrorCode asserts an HTTP status code on a given Echo HTTP error
func AssertHTTPErrorCode(t *testing.T, err error, code int) {
	httpError, ok := err.(*echo.HTTPError)
	require.True(t, ok)
	assert.Equal(t, code, httpError.Code)
}

type UserRecord struct {
	ID       int
	Name     string
	Email    string
	Password string
	Verified bool
}

// https://testcontainers.com/guides/getting-started-with-testcontainers-for-go/
// https://golang.testcontainers.org/modules/postgres/
func CreateTestContainerPostgresConnStr(t *testing.T) (string, context.Context) {
	ctx := context.Background()

	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("ankane/pgvector:v0.5.1"),
		postgres.WithDatabase("test-db"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(15*time.Second)),
	)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate pgContainer: %s", err)
		}
	})
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	assert.NoError(t, err)
	return connStr, ctx
}

// CreateTestContainerPostgresDB returns a migration-ready DB handle for integration tests.
func CreateTestContainerPostgresDB(t *testing.T) (*sql.DB, string, context.Context) {
	connStr, ctx := CreateTestContainerPostgresConnStr(t)

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	_, err = db.Exec("CREATE EXTENSION IF NOT EXISTS vector")
	if err != nil {
		t.Fatalf("failed to enable pgvector: %v", err)
	}

	if err := applyCoreMigrations(t, db); err != nil {
		t.Fatalf("failed to apply migrations: %v", err)
	}
	return db, "postgres", ctx
}

// CreateRandomUserDB creates a random user through SQL for DB-first tests.
func CreateRandomUserDB(db *sql.DB) (*UserRecord, error) {
	seed := fmt.Sprintf("%d-%d", time.Now().UnixMilli(), rand.Intn(1000000))
	return CreateUserDB(
		context.Background(),
		db,
		fmt.Sprintf("Test User %s", seed),
		fmt.Sprintf("testuser-%s@localhost.localhost", seed),
		"password",
		true,
	)
}

// CreateUserDB creates a user through SQL and returns a lightweight record.
func CreateUserDB(ctx context.Context, db *sql.DB, name, email, password string, verified bool) (*UserRecord, error) {
	if db == nil {
		return nil, fmt.Errorf("db is nil")
	}
	now := time.Now().UTC()
	var id int
	if err := db.QueryRowContext(
		ctx,
		`INSERT INTO users (created_at, updated_at, name, email, password, verified)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		now,
		now,
		name,
		email,
		password,
		verified,
	).Scan(&id); err != nil {
		return nil, err
	}
	return &UserRecord{
		ID:       id,
		Name:     name,
		Email:    email,
		Password: password,
		Verified: verified,
	}, nil
}

// LinkFriendsDB links a profile to friend profile IDs through SQL.
func LinkFriendsDB(ctx context.Context, db *sql.DB, profileID int, matchIDs []int) error {
	if db == nil {
		return fmt.Errorf("db is nil")
	}
	for _, matchID := range matchIDs {
		if _, err := db.ExecContext(
			ctx,
			`INSERT INTO profile_friends (profile_id, friend_id) VALUES ($1, $2)`,
			profileID,
			matchID,
		); err != nil {
			return err
		}
	}
	return nil
}

func applyCoreMigrations(t *testing.T, db *sql.DB) error {
	t.Helper()
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return fmt.Errorf("resolve current file path for migrations")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	migrationsDir := filepath.Join(repoRoot, "db", "migrate", "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir %q: %w", migrationsDir, err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS goship_schema_migrations (
	version VARCHAR(255) PRIMARY KEY,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`); err != nil {
		return fmt.Errorf("ensure migration tracking table: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		version := strings.SplitN(entry.Name(), ".", 2)[0]
		if version == "" {
			continue
		}

		var applied int
		err := db.QueryRow(`SELECT 1 FROM goship_schema_migrations WHERE version = $1`, version).Scan(&applied)
		if err == nil {
			continue
		}
		if err != sql.ErrNoRows {
			return fmt.Errorf("check migration version %q: %w", version, err)
		}

		content, err := os.ReadFile(filepath.Join(migrationsDir, entry.Name()))
		if err != nil {
			return fmt.Errorf("read migration %q: %w", entry.Name(), err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin tx for %q: %w", entry.Name(), err)
		}
		if _, err := tx.Exec(string(content)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("execute migration %q: %w", entry.Name(), err)
		}
		if _, err := tx.Exec(`INSERT INTO goship_schema_migrations (version) VALUES ($1)`, version); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %q: %w", entry.Name(), err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %q: %w", entry.Name(), err)
		}
	}
	return nil
}
