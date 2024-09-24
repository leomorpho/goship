package seeder_test

import (
	"database/sql"
	"testing"

	"github.com/jackc/pgx/stdlib"
	"github.com/mikestefanello/pagoda/config"
	"github.com/mikestefanello/pagoda/pkg/tests"
	"github.com/mikestefanello/pagoda/seeder"
)

func init() {
	// Register "pgx" as "postgres" explicitly for database/sql
	sql.Register("postgres", stdlib.GetDefaultDriver())
}

// TestSeeder tests the seeder code, confirming that it runs.
func TestSeeder(t *testing.T) {
	client, _ := tests.CreateTestContainerPostgresEntClient(t)
	defer client.Close()

	config := config.Config{}
	seeder.RunIdempotentSeeder(&config, client)

	// Assert stuff

	// Seeder should be idempotent and not throw exceptions if re-run
	seeder.RunIdempotentSeeder(&config, client)
}
