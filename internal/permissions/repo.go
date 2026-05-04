package permissions

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	sqlc "github.com/mostdoc/mostdoc/internal/db/query"
)

// Repository wraps sqlc queries for permission operations.
type Repository struct {
	queries *sqlc.Queries
}

// NewRepository creates a new permissions repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{queries: sqlc.New(db)}
}

// GetPermission fetches a specific permission record.
func (r *Repository) GetPermission(ctx context.Context, targetType, targetID, granteeType, granteeID string) (sqlc.Permission, error) {
	return r.queries.GetPermission(ctx, sqlc.GetPermissionParams{
		TargetType:  targetType,
		TargetID:    targetID,
		GranteeType: granteeType,
		GranteeID:   granteeID,
	})
}

// UpsertPermission creates or updates a permission.
func (r *Repository) UpsertPermission(ctx context.Context, targetType, targetID, granteeType, granteeID, createdBy string, level int64) error {
	now := time.Now().Unix()
	return r.queries.UpsertPermission(ctx, sqlc.UpsertPermissionParams{
		ID:          uuid.New().String(),
		TargetType:  targetType,
		TargetID:    targetID,
		GranteeType: granteeType,
		GranteeID:   granteeID,
		Level:       level,
		CreatedBy:   sql.NullString{String: createdBy, Valid: createdBy != ""},
		CreatedAt:   sql.NullInt64{Int64: now, Valid: true},
		UpdatedAt:   sql.NullInt64{Int64: now, Valid: true},
	})
}

// DeletePermission removes a permission.
func (r *Repository) DeletePermission(ctx context.Context, targetType, targetID, granteeType, granteeID string) error {
	return r.queries.DeletePermission(ctx, sqlc.DeletePermissionParams{
		TargetType:  targetType,
		TargetID:    targetID,
		GranteeType: granteeType,
		GranteeID:   granteeID,
	})
}

// ListPermissionsByTarget returns all permissions for a target.
func (r *Repository) ListPermissionsByTarget(ctx context.Context, targetType, targetID string) ([]sqlc.Permission, error) {
	return r.queries.ListPermissionsByTarget(ctx, sqlc.ListPermissionsByTargetParams{
		TargetType: targetType,
		TargetID:   targetID,
	})
}

// ListPermissionsByGrantee returns all permissions for a grantee.
func (r *Repository) ListPermissionsByGrantee(ctx context.Context, granteeType, granteeID string) ([]sqlc.Permission, error) {
	return r.queries.ListPermissionsByGrantee(ctx, sqlc.ListPermissionsByGranteeParams{
		GranteeType: granteeType,
		GranteeID:   granteeID,
	})
}

// DeletePermissionsForTarget removes all permissions for a target.
func (r *Repository) DeletePermissionsForTarget(ctx context.Context, targetType, targetID string) error {
	return r.queries.DeletePermissionsForTarget(ctx, sqlc.DeletePermissionsForTargetParams{
		TargetType: targetType,
		TargetID:   targetID,
	})
}
