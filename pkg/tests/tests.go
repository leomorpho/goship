package tests

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mikestefanello/pagoda/ent"
	"github.com/mikestefanello/pagoda/ent/enttest"
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

func CreateTestContainerPostgresEntClient(t *testing.T) (*ent.Client, context.Context) {
	connStr, ctx := CreateTestContainerPostgresConnStr(t)

	// Initialize a connection to the database using the connection string
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	// Ensure the pgvector extension is installed
	_, err = db.Exec("CREATE EXTENSION IF NOT EXISTS vector")
	if err != nil {
		t.Fatalf("failed to enable pgvector: %v", err)
	}

	// Initialize Ent client with a test schema.
	client := enttest.Open(t, "postgres", connStr)
	t.Cleanup(func() {
		client.Close()
	})

	err = client.Schema.Create(ctx)
	assert.NoError(t, err)
	return client, ctx
}

// CreateUser creates a random user entity
func CreateRandomUser(orm *ent.Client) (*ent.User, error) {
	seed := fmt.Sprintf("%d-%d", time.Now().UnixMilli(), rand.Intn(1000000))
	return orm.User.
		Create().
		SetEmail(fmt.Sprintf("testuser-%s@localhost.localhost", seed)).
		SetPassword("password").
		SetName(fmt.Sprintf("Test User %s", seed)).
		Save(context.Background())
}

// CreateUser creates a new user and returns its ID.
func CreateUser(ctx context.Context, client *ent.Client, name string, email string, password string, verified bool) *ent.User {
	// Create a new user with the provided arguments
	return client.User.
		Create().
		SetName(name).
		SetEmail(email).
		SetPassword(password).
		SetVerified(verified).
		SaveX(ctx)
}

// LinkProfilesAsMatches links two profiles as matches.
func LinkFriends(ctx context.Context, client *ent.Client, profileID int, matchIDs []int) {
	profile, err := client.Profile.Get(ctx, profileID)
	if err != nil {
		panic(fmt.Sprintf("Failed fetching profile for linking matches: %v", err))
	}

	// Link the profiles.
	for _, matchID := range matchIDs {
		err := client.Profile.
			UpdateOneID(profile.ID).
			AddFriendIDs(matchID).
			Exec(ctx)

		if err != nil {
			panic(fmt.Sprintf("Failed linking profile %d and %d: %v", profile.ID, matchID, err))
		}
	}
}
