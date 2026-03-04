//go:build integration

package seeder_test

import (
	"database/sql"
	"testing"

	"github.com/jackc/pgx/stdlib"
	"github.com/leomorpho/goship/config"
	"github.com/leomorpho/goship/framework/tests"
	"github.com/leomorpho/goship/tools/seeder"
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
