-- 003_spaces.up.sql
-- Iteration 2: spaces table

CREATE TABLE spaces (
    id TEXT PRIMARY KEY,
    workspace_id TEXT DEFAULT 'default',
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    default_permission TEXT DEFAULT 'none',
    mcp_write_enabled INTEGER DEFAULT 1,
    snapshot_retention_days INTEGER DEFAULT NULL,
    created_at INTEGER DEFAULT (strftime('%s', 'now'))
);

CREATE INDEX idx_spaces_slug ON spaces(slug);
