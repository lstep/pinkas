package groups

import (
	"context"
	"database/sql"

	sqlc "github.com/mostdoc/mostdoc/internal/db/query"
)

// Repository wraps sqlc queries for group operations.
type Repository struct {
	queries *sqlc.Queries
}

// NewRepository creates a new groups repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{queries: sqlc.New(db)}
}

// CreateGroup creates a new group.
func (r *Repository) CreateGroup(ctx context.Context, id, name string) error {
	return r.queries.CreateGroup(ctx, sqlc.CreateGroupParams{
		ID:   id,
		Name: name,
	})
}

// GetGroup fetches a group by ID.
func (r *Repository) GetGroup(ctx context.Context, id string) (sqlc.Group, error) {
	return r.queries.GetGroup(ctx, id)
}

// ListGroups returns all groups ordered by name.
func (r *Repository) ListGroups(ctx context.Context) ([]sqlc.Group, error) {
	return r.queries.ListGroups(ctx)
}

// UpdateGroup renames a group.
func (r *Repository) UpdateGroup(ctx context.Context, id, name string) error {
	return r.queries.UpdateGroup(ctx, sqlc.UpdateGroupParams{
		Name: name,
		ID:   id,
	})
}

// DeleteGroup removes a group by ID.
func (r *Repository) DeleteGroup(ctx context.Context, id string) error {
	return r.queries.DeleteGroup(ctx, id)
}

// AddMember adds a user to a group.
func (r *Repository) AddMember(ctx context.Context, groupID, userID string) error {
	return r.queries.AddGroupMember(ctx, sqlc.AddGroupMemberParams{
		GroupID: groupID,
		UserID:  userID,
	})
}

// RemoveMember removes a user from a group.
func (r *Repository) RemoveMember(ctx context.Context, groupID, userID string) error {
	return r.queries.RemoveGroupMember(ctx, sqlc.RemoveGroupMemberParams{
		GroupID: groupID,
		UserID:  userID,
	})
}

// ListMembers returns all members of a group.
func (r *Repository) ListMembers(ctx context.Context, groupID string) ([]sqlc.ListGroupMembersRow, error) {
	return r.queries.ListGroupMembers(ctx, groupID)
}

// ListUserGroups returns all groups a user belongs to.
func (r *Repository) ListUserGroups(ctx context.Context, userID string) ([]sqlc.Group, error) {
	return r.queries.ListUserGroups(ctx, userID)
}
