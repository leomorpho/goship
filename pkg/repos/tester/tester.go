package tester

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"time"

	"github.com/mikestefanello/pagoda/ent"
	"github.com/mikestefanello/pagoda/ent/enttest"
	"github.com/stretchr/testify/assert"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

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
