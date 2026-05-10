package testutil

import (
	"context"
	"testing"

	migratepkg "github.com/autotraka/go-gateway/internal/migrate"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func SetupTestDB(tb testing.TB) (*pgxpool.Pool, func()) {
	tb.Helper()

	ctx := context.Background()

	c, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("testuser"),
		tcpostgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		tb.Fatalf("failed to start postgres container: %v", err)
	}

	connStr, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		tb.Fatalf("failed to get connection string: %v", err)
	}

	if err := migratepkg.Run(connStr, "up"); err != nil {
		tb.Fatalf("failed to run migrations: %v", err)
	}

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		tb.Fatalf("failed to connect to database: %v", err)
	}

	cleanup := func() {
		pool.Close()
		c.Terminate(ctx)
	}

	return pool, cleanup
}