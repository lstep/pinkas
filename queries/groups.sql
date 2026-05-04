-- name: CreateGroup :exec
INSERT INTO groups (id, name) VALUES (?, ?);

-- name: GetGroup :one
SELECT id, name, created_at FROM groups WHERE id = ?;

-- name: ListGroups :many
SELECT id, name, created_at FROM groups ORDER BY name;

-- name: UpdateGroup :exec
UPDATE groups SET name = ? WHERE id = ?;

-- name: DeleteGroup :exec
DELETE FROM groups WHERE id = ?;

-- name: AddGroupMember :exec
INSERT OR IGNORE INTO group_members (group_id, user_id) VALUES (?, ?);

-- name: RemoveGroupMember :exec
DELETE FROM group_members WHERE group_id = ? AND user_id = ?;

-- name: ListGroupMembers :many
SELECT u.id, u.email, u.name, u.global_role, u.created_at
FROM users u
JOIN group_members gm ON gm.user_id = u.id
WHERE gm.group_id = ?
ORDER BY u.email;

-- name: ListUserGroups :many
SELECT g.id, g.name, g.created_at
FROM groups g
JOIN group_members gm ON gm.group_id = g.id
WHERE gm.user_id = ?
ORDER BY g.name;
