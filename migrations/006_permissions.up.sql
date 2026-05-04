-- 006_permissions.up.sql
-- Iteration 3: multi-user access control (groups + permissions)

CREATE TABLE groups (
    id TEXT PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    created_at INTEGER DEFAULT (strftime('%s', 'now'))
);

CREATE TABLE group_members (
    group_id TEXT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (group_id, user_id)
);

CREATE TABLE permissions (
    id TEXT PRIMARY KEY,
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    grantee_type TEXT NOT NULL,
    grantee_id TEXT NOT NULL,
    level INTEGER NOT NULL DEFAULT 0,
    created_by TEXT REFERENCES users(id),
    created_at INTEGER DEFAULT (strftime('%s', 'now')),
    updated_at INTEGER DEFAULT (strftime('%s', 'now'))
);

CREATE UNIQUE INDEX idx_permissions_unique ON permissions(target_type, target_id, grantee_type, grantee_id);
CREATE INDEX idx_permissions_target ON permissions(target_type, target_id);
CREATE INDEX idx_permissions_grantee ON permissions(grantee_type, grantee_id);

-- Soft delete support for users
ALTER TABLE users ADD COLUMN deleted_at INTEGER DEFAULT NULL;
