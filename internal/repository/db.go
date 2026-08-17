package repository

import (
	"embed"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	mysqlmigrate "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
)

//go:embed migrationsfs/*.sql
var migrationsFS embed.FS

func Connect(dsn string) (*sqlx.DB, error) {
	db, err := sqlx.Connect("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("connect to mysql: %w", err)
	}
	return db, nil
}

// Migrate applies every pending migration embedded in the binary against
// an already-open connection. Safe to call on every startup: golang-migrate
// tracks the applied version in a schema_migrations table and is a no-op
// once the schema is current.
func Migrate(db *sqlx.DB) error {
	sourceDriver, err := iofs.New(migrationsFS, "migrationsfs")
	if err != nil {
		return fmt.Errorf("load embedded migrations: %w", err)
	}

	dbDriver, err := mysqlmigrate.WithInstance(db.DB, &mysqlmigrate.Config{})
	if err != nil {
		return fmt.Errorf("init mysql migration driver: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, "mysql", dbDriver)
	if err != nil {
		return fmt.Errorf("init migrator: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
