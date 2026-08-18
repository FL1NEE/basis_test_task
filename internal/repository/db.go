package repository

import (
	"embed"
	"fmt"
	"time"

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
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(10 * time.Minute)
	return db, nil
}

// Migrate is safe to call on every startup: golang-migrate no-ops once
// the schema is already current.
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
