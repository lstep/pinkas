package db

import (
	"database/sql"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func Migrate(conn *sql.DB, migrationsPath string) error {
	driver, err := sqlite3.WithInstance(conn, &sqlite3.Config{})
	if err != nil {
		return fmt.Errorf("create migrate driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		migrationsPath,
		"sqlite3",
		driver,
	)
	if err != nil {
		return fmt.Errorf("new migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}

func Seed(conn *sql.DB) error {
	// Ensure default space exists
	var spaceCount int
	err := conn.QueryRow("SELECT COUNT(*) FROM spaces").Scan(&spaceCount)
	if err != nil {
		return fmt.Errorf("count spaces: %w", err)
	}

	if spaceCount == 0 {
		_, err := conn.Exec(`
			INSERT INTO spaces (id, name, slug, default_permission)
			VALUES ('default', 'Default Space', 'default', 'editor')
		`)
		if err != nil {
			return fmt.Errorf("seed space: %w", err)
		}
	}

	// Ensure seed page exists
	var count int
	err = conn.QueryRow("SELECT COUNT(*) FROM pages").Scan(&count)
	if err != nil {
		return fmt.Errorf("count pages: %w", err)
	}

	if count == 0 {
		_, err := conn.Exec(`
			INSERT INTO pages (id, space_id, title, slug, position)
			VALUES ('seed-page-001', 'default', 'Welcome', 'welcome', 0)
		`)
		if err != nil {
			return fmt.Errorf("seed page: %w", err)
		}
	}

	return nil
}
