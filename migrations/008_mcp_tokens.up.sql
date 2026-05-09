-- 008_mcp_tokens.up.sql
-- Iteration 6: MCP Integration — long-lived API tokens

CREATE TABLE IF NOT EXISTS mcp_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    token_prefix TEXT NOT NULL,        -- first few chars for display (e.g. "mcp_a1b2")
    token_hash TEXT NOT NULL,          -- bcrypt hash of the full secret
    scopes TEXT NOT NULL DEFAULT '["read"]',  -- JSON array of permitted operations
    space_id TEXT,                     -- optional space restriction (NULL = all spaces)
    last_used_at INTEGER,              -- unix timestamp
    created_at INTEGER NOT NULL,       -- unix timestamp
    expires_at INTEGER                 -- unix timestamp, NULL = never expires
);

CREATE INDEX IF NOT EXISTS idx_mcp_tokens_user_id ON mcp_tokens(user_id);
