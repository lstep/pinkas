-- 001_initial.up.sql
-- Iteration 1: pages and page_snapshots tables

CREATE TABLE IF NOT EXISTS pages (
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

CREATE TABLE IF NOT EXISTS page_snapshots (
    id TEXT PRIMARY KEY,
    page_id TEXT REFERENCES pages(id),
    yjs_snapshot BLOB,
    markdown TEXT,
    author_id TEXT,
    created_at INTEGER DEFAULT (strftime('%s', 'now')),
    label TEXT,
    is_compacted INTEGER DEFAULT 0
);
