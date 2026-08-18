package repository

import (
	"context"
	"database/sql"
)

// DBTX is satisfied by both *sqlx.DB and *sqlx.Tx, so a repository can be
// bound to either a transaction or the plain pool.
type DBTX interface {
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}
