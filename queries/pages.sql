-- name: CreatePage :exec
INSERT INTO pages (id, space_id, directory_id, title, slug, position, created_by, icon)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetPage :one
SELECT id, space_id, directory_id, title, slug, position, created_by, created_at, updated_at, icon
FROM pages
WHERE id = ?;

-- name: GetPageBySlug :one
SELECT id, space_id, directory_id, title, slug, position, created_by, created_at, updated_at, icon
FROM pages
WHERE space_id = ? AND slug = ?;

-- name: UpdatePageTitle :exec
UPDATE pages
SET title = ?, slug = ?, updated_at = strftime('%s', 'now')
WHERE id = ?;

-- name: UpdatePagePosition :exec
UPDATE pages
SET position = ?, directory_id = ?, updated_at = strftime('%s', 'now')
WHERE id = ?;

-- name: UpdatePageIcon :exec
UPDATE pages
SET icon = ?, updated_at = strftime('%s', 'now')
WHERE id = ?;

-- name: DeletePage :exec
DELETE FROM pages WHERE id = ?;

-- name: ListPagesByDirectory :many
SELECT id, space_id, directory_id, title, slug, position, created_by, created_at, updated_at, icon
FROM pages
WHERE directory_id = ?
ORDER BY position;

-- name: ListRootPages :many
SELECT id, space_id, directory_id, title, slug, position, created_by, created_at, updated_at, icon
FROM pages
WHERE space_id = ? AND directory_id IS NULL
ORDER BY position;

-- name: GetMaxPagePosition :one
SELECT COALESCE(MAX(position), -1) FROM pages WHERE directory_id = ?;

-- name: GetPagesBySlugPrefix :many
SELECT id, space_id, directory_id, title, slug, position, created_by, created_at, updated_at, icon
FROM pages
WHERE space_id = ? AND slug LIKE ?
ORDER BY slug;

-- name: ListRecentPages :many
SELECT p.id, p.space_id, p.directory_id, p.title, p.slug, p.position, p.created_by, p.created_at, p.updated_at, p.icon
FROM pages p
JOIN spaces s ON p.space_id = s.id
ORDER BY p.updated_at DESC
LIMIT ?;

-- name: ListMyPages :many
SELECT p.id, p.space_id, p.directory_id, p.title, p.slug, p.position, p.created_by, p.created_at, p.updated_at, p.icon
FROM pages p
JOIN spaces s ON p.space_id = s.id
WHERE p.created_by = ?
ORDER BY p.updated_at DESC
LIMIT ?;


