-- name: GetPermission :one
SELECT id, target_type, target_id, grantee_type, grantee_id, level, created_by, created_at, updated_at
FROM permissions
WHERE target_type = ? AND target_id = ? AND grantee_type = ? AND grantee_id = ?;

-- name: UpsertPermission :exec
INSERT INTO permissions (id, target_type, target_id, grantee_type, grantee_id, level, created_by, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(target_type, target_id, grantee_type, grantee_id)
DO UPDATE SET level = excluded.level, updated_at = excluded.updated_at;

-- name: DeletePermission :exec
DELETE FROM permissions
WHERE target_type = ? AND target_id = ? AND grantee_type = ? AND grantee_id = ?;

-- name: ListPermissionsByTarget :many
SELECT id, target_type, target_id, grantee_type, grantee_id, level, created_by, created_at, updated_at
FROM permissions
WHERE target_type = ? AND target_id = ?
ORDER BY grantee_type, grantee_id;

-- name: ListPermissionsByGrantee :many
SELECT id, target_type, target_id, grantee_type, grantee_id, level, created_by, created_at, updated_at
FROM permissions
WHERE grantee_type = ? AND grantee_id = ?
ORDER BY target_type, target_id;

-- name: ListPermissionsByTargetAndGrantee :many
SELECT id, target_type, target_id, grantee_type, grantee_id, level, created_by, created_at, updated_at
FROM permissions
WHERE target_type = ? AND target_id = ? AND grantee_type = ?
ORDER BY grantee_id;

-- name: DeletePermissionsForTarget :exec
DELETE FROM permissions WHERE target_type = ? AND target_id = ?;

-- name: DeletePermissionsByGrantee :exec
DELETE FROM permissions WHERE grantee_type = ? AND grantee_id = ?;
