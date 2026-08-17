package repository

import (
	"context"
	"database/sql"
)

// DBTX is satisfied by both *sqlx.DB and *sqlx.Tx. Every repository method
// takes this instead of a concrete type so the service layer can pass a
// transaction through when an operation must be atomic, or the plain pool
// otherwise.
type DBTX interface {
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}
