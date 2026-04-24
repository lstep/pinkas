-- 005_split_directories_pages.up.sql
-- Iteration 2 refactor: split directories and pages into separate tables

PRAGMA foreign_keys=off;

-- 1. Create directories table (self-referential tree)
CREATE TABLE directories (
    id TEXT PRIMARY KEY,
    space_id TEXT REFERENCES spaces(id),
    parent_id TEXT REFERENCES directories(id),
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    position INTEGER DEFAULT 0,
    icon TEXT,
    created_by TEXT REFERENCES users(id),
    created_at INTEGER DEFAULT (strftime('%s', 'now')),
    updated_at INTEGER DEFAULT (strftime('%s', 'now'))
);

CREATE UNIQUE INDEX idx_directories_space_slug ON directories(space_id, slug);
CREATE INDEX idx_directories_parent_id ON directories(parent_id);
CREATE INDEX idx_directories_space_id ON directories(space_id);

-- 2. Migrate existing directories from pages table
INSERT INTO directories (id, space_id, parent_id, name, slug, position, icon, created_by, created_at, updated_at)
SELECT id, space_id, parent_id, title, slug, position, icon, created_by, created_at, updated_at
FROM pages WHERE is_directory = 1;

-- 3. Create new pages table with directory_id instead of parent_id/is_directory
CREATE TABLE pages_new (
    id TEXT PRIMARY KEY,
    space_id TEXT REFERENCES spaces(id),
    directory_id TEXT REFERENCES directories(id),
    title TEXT DEFAULT 'Untitled',
    slug TEXT DEFAULT 'untitled',
    position INTEGER DEFAULT 0,
    created_by TEXT REFERENCES users(id),
    created_at INTEGER DEFAULT (strftime('%s', 'now')),
    updated_at INTEGER DEFAULT (strftime('%s', 'now')),
    icon TEXT
);

-- 4. Copy page data with directory_id mapping:
--    - parent was a directory  -> directory_id = parent_id
--    - parent was a page       -> directory_id = NULL (broken state, move to root)
--    - no parent               -> directory_id = NULL
INSERT INTO pages_new (id, space_id, directory_id, title, slug, position, created_by, created_at, updated_at, icon)
SELECT 
    id,
    space_id,
    CASE 
        WHEN parent_id IS NULL THEN NULL
        WHEN parent_id IN (SELECT id FROM directories) THEN parent_id
        ELSE NULL
    END AS directory_id,
    title,
    slug,
    position,
    created_by,
    created_at,
    updated_at,
    icon
FROM pages WHERE is_directory = 0;

-- 5. Replace old pages table
DROP TABLE pages;
ALTER TABLE pages_new RENAME TO pages;

-- 6. Create indexes on new pages table
CREATE INDEX idx_pages_space_id ON pages(space_id);
CREATE INDEX idx_pages_directory_id ON pages(directory_id);
CREATE INDEX idx_pages_slug ON pages(slug);
CREATE UNIQUE INDEX idx_pages_space_slug ON pages(space_id, slug);

PRAGMA foreign_keys=on;
