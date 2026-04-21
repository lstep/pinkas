package db

import (
	"database/sql"
	"fmt"
)

func Migrate(conn *sql.DB) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS pages (
			id TEXT PRIMARY KEY,
			space_id TEXT DEFAULT 'default',
			title TEXT DEFAULT 'Untitled',
			slug TEXT DEFAULT 'untitled',
			position INTEGER DEFAULT 0,
			parent_id TEXT REFERENCES pages(id),
			created_by TEXT,
			created_at INTEGER DEFAULT (strftime('%s', 'now')),
			updated_at INTEGER DEFAULT (strftime('%s', 'now')),
			is_directory BOOLEAN DEFAULT 0,
			icon TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS page_snapshots (
			id TEXT PRIMARY KEY,
			page_id TEXT REFERENCES pages(id),
			yjs_snapshot BLOB,
			markdown TEXT,
			author_id TEXT,
			created_at INTEGER DEFAULT (strftime('%s', 'now')),
			label TEXT,
			is_compacted INTEGER DEFAULT 0
		)`,
	}

	tx, err := conn.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	for _, ddl := range migrations {
		if _, err := tx.Exec(ddl); err != nil {
			return fmt.Errorf("exec ddl: %w", err)
		}
	}

	return tx.Commit()
}

func Seed(conn *sql.DB) error {
	var count int
	err := conn.QueryRow("SELECT COUNT(*) FROM pages").Scan(&count)
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
