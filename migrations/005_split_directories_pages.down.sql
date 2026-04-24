-- 005_split_directories_pages.down.sql
-- Reverse: merge directories back into pages table

PRAGMA foreign_keys=off;

-- 1. Recreate pages with parent_id and is_directory
CREATE TABLE pages_old (
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

-- 2. Copy directories back as pages with is_directory=1
INSERT INTO pages_old (id, space_id, title, slug, position, parent_id, created_by, created_at, updated_at, is_directory, icon)
SELECT id, space_id, name, slug, position, parent_id, created_by, created_at, updated_at, 1, icon
FROM directories;

-- 3. Copy pages back with parent_id = directory_id
INSERT INTO pages_old (id, space_id, title, slug, position, parent_id, created_by, created_at, updated_at, is_directory, icon)
SELECT id, space_id, title, slug, position, directory_id, created_by, created_at, updated_at, 0, icon
FROM pages;

-- 4. Replace current pages table
DROP TABLE pages;
ALTER TABLE pages_old RENAME TO pages;

-- 5. Drop directories table and its indexes
DROP INDEX idx_directories_space_slug;
DROP INDEX idx_directories_parent_id;
DROP INDEX idx_directories_space_id;
DROP TABLE directories;

-- 6. Recreate original page indexes
CREATE INDEX idx_pages_space_id ON pages(space_id);
CREATE INDEX idx_pages_parent_id ON pages(parent_id);
CREATE INDEX idx_pages_slug ON pages(slug);
CREATE UNIQUE INDEX idx_pages_space_slug ON pages(space_id, slug);

PRAGMA foreign_keys=on;
