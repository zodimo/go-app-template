package database

import (
	"embed"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func migrateNow(store *Store) error {

	// 1. Initialize the iofs driver with the embedded filesystem
	d, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("migrateNow: iofs , %w", err)
	}

	// 2. Initialize the migrate instance
	// Replace the connection string with your actual DB URL
	m, err := migrate.NewWithSourceInstance("iofs", d, store.DbUrlForMigration())
	if err != nil {
		return fmt.Errorf("migrateNow: init , %w", err)
	}

	// 3. Apply migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("migrateNow: up , %w", err)
	}
	return nil
}


