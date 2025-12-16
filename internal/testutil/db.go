package testutil

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"addon-radar/internal/database"
)

// TestDB holds the database connection for tests
type TestDB struct {
	Queries *database.Queries
	Pool    *pgxpool.Pool
}

// SetupTestDB creates a PostgreSQL container and returns a TestDB instance.
// The container and connection pool are automatically cleaned up when the test finishes.
func SetupTestDB(t *testing.T) *TestDB {
	ctx := context.Background()

	postgresContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		if err := postgresContainer.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	})

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	t.Cleanup(func() { pool.Close() })

	// Apply schema - try multiple relative paths to find the schema file
	schemaPaths := []string{
		"../../sql/schema.sql",
		"../../../sql/schema.sql",
	}

	var schema []byte
	for _, path := range schemaPaths {
		// #nosec G304 -- This is test code reading from known, hardcoded paths only
		schema, err = os.ReadFile(path)
		if err == nil {
			break
		}
	}
	require.NoError(t, err, "failed to read schema.sql from any known path")

	_, err = pool.Exec(ctx, string(schema))
	require.NoError(t, err)

	return &TestDB{
		Queries: database.New(pool),
		Pool:    pool,
	}
}
