-- 004_pages_enrich.up.sql
-- Iteration 2: add FK to spaces, indexes, and ensure columns exist

-- Add FK constraint to spaces (recreate table since SQLite doesn't support ALTER TABLE ADD CONSTRAINT)
PRAGMA foreign_keys=off;

ALTER TABLE pages RENAME TO pages_old;

CREATE TABLE pages (
    id TEXT PRIMARY KEY,
    space_id TEXT REFERENCES spaces(id),
    title TEXT DEFAULT 'Untitled',
    slug TEXT DEFAULT 'untitled',
    position INTEGER DEFAULT 0,
    parent_id TEXT REFERENCES pages(id),
    created_by TEXT REFERENCES users(id),
    created_at INTEGER DEFAULT (strftime('%s', 'now')),
    updated_at INTEGER DEFAULT (strftime('%s', 'now')),
    is_directory BOOLEAN DEFAULT 0,
    icon TEXT
);

INSERT INTO pages (id, space_id, title, slug, position, parent_id, created_by, created_at, updated_at, is_directory, icon)
SELECT id, space_id, title, slug, position, parent_id, created_by, created_at, updated_at, is_directory, icon
FROM pages_old;

DROP TABLE pages_old;

-- Indexes for tree and lookup queries
CREATE INDEX idx_pages_space_id ON pages(space_id);
CREATE INDEX idx_pages_parent_id ON pages(parent_id);
CREATE INDEX idx_pages_slug ON pages(slug);
CREATE UNIQUE INDEX idx_pages_space_slug ON pages(space_id, slug);

PRAGMA foreign_keys=on;
