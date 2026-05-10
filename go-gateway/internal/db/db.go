package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/XSAM/otelsql"
	"github.com/jmoiron/sqlx"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Querier abstracts the common sqlx query methods for testability.
type Querier interface {
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryRowxContext(ctx context.Context, query string, args ...interface{}) *sqlx.Row
	QueryxContext(ctx context.Context, query string, args ...interface{}) (*sqlx.Rows, error)
}

var _ Querier = (*sqlx.DB)(nil)

// New initializes a new instrumented sqlx.DB connection pool.
func New(databaseURL string) (*sqlx.DB, error) {
	// Register an instrumented pgx driver so queries emit OTel spans.
	driverName, err := otelsql.Register("pgx",
		otelsql.WithAttributes(),
	)
	if err != nil {
		return nil, fmt.Errorf("register otelsql driver: %w", err)
	}

	db, err := sqlx.Connect(driverName, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}
	return db, nil
}
