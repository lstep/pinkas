-- 004_pages_enrich.down.sql
PRAGMA foreign_keys=off;

DROP INDEX IF EXISTS idx_pages_space_slug;
DROP INDEX IF EXISTS idx_pages_space_id;
DROP INDEX IF EXISTS idx_pages_parent_id;
DROP INDEX IF EXISTS idx_pages_slug;

ALTER TABLE pages RENAME TO pages_new;

CREATE TABLE pages (
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
);

INSERT INTO pages (id, space_id, title, slug, position, parent_id, created_by, created_at, updated_at, is_directory, icon)
SELECT id, space_id, title, slug, position, parent_id, created_by, created_at, updated_at, is_directory, icon
FROM pages_new;

DROP TABLE pages_new;

PRAGMA foreign_keys=on;
