-- name: CreateDirectory :exec
INSERT INTO directories (id, space_id, parent_id, name, slug, position, icon, created_by)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetDirectory :one
SELECT id, space_id, parent_id, name, slug, position, icon, created_by, created_at, updated_at
FROM directories
WHERE id = ?;

-- name: GetDirectoryBySlug :one
SELECT id, space_id, parent_id, name, slug, position, icon, created_by, created_at, updated_at
FROM directories
WHERE space_id = ? AND slug = ?;

-- name: UpdateDirectoryName :exec
UPDATE directories
SET name = ?, slug = ?, updated_at = strftime('%s', 'now')
WHERE id = ?;

-- name: UpdateDirectoryPosition :exec
UPDATE directories
SET position = ?, parent_id = ?, updated_at = strftime('%s', 'now')
WHERE id = ?;

-- name: UpdateDirectoryIcon :exec
UPDATE directories
SET icon = ?, updated_at = strftime('%s', 'now')
WHERE id = ?;

-- name: DeleteDirectory :exec
DELETE FROM directories WHERE id = ?;

-- name: ListRootDirectories :many
SELECT id, space_id, parent_id, name, slug, position, icon, created_by, created_at, updated_at
FROM directories
WHERE space_id = ? AND parent_id IS NULL
ORDER BY position;

-- name: ListDirectorySubdirectories :many
SELECT id, space_id, parent_id, name, slug, position, icon, created_by, created_at, updated_at
FROM directories
WHERE parent_id = ?
ORDER BY position;

-- name: GetMaxDirectoryPosition :one
SELECT COALESCE(MAX(position), -1) FROM directories WHERE parent_id = ?;
