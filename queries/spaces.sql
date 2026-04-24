-- name: CreateSpace :exec
INSERT INTO spaces (id, name, slug, default_permission, mcp_write_enabled, snapshot_retention_days)
VALUES (?, ?, ?, ?, ?, ?);

-- name: GetSpace :one
SELECT id, workspace_id, name, slug, default_permission, mcp_write_enabled, snapshot_retention_days, created_at
FROM spaces
WHERE id = ?;

-- name: GetSpaceBySlug :one
SELECT id, workspace_id, name, slug, default_permission, mcp_write_enabled, snapshot_retention_days, created_at
FROM spaces
WHERE slug = ?;

-- name: ListSpaces :many
SELECT id, workspace_id, name, slug, default_permission, mcp_write_enabled, snapshot_retention_days, created_at
FROM spaces
ORDER BY name;

-- name: UpdateSpace :exec
UPDATE spaces
SET name = ?, default_permission = ?, mcp_write_enabled = ?, snapshot_retention_days = ?, updated_at = strftime('%s', 'now')
WHERE id = ?;

-- name: DeleteSpace :exec
DELETE FROM spaces WHERE id = ?;
