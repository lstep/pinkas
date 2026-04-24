package spaces

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	sqlc "github.com/mostdoc/mostdoc/internal/db/query"
)

// Repository wraps sqlc queries for space operations.
type Repository struct {
	queries *sqlc.Queries
}

// NewRepository creates a new space repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{queries: sqlc.New(db)}
}

// Create inserts a new space.
func (r *Repository) Create(ctx context.Context, id, name, slug, defaultPermission string, mcpWriteEnabled bool, snapshotRetentionDays *int64) error {
	var mcp int64 = 1
	if !mcpWriteEnabled {
		mcp = 0
	}
	var retention sql.NullInt64
	if snapshotRetentionDays != nil {
		retention = sql.NullInt64{Int64: *snapshotRetentionDays, Valid: true}
	}
	return r.queries.CreateSpace(ctx, sqlc.CreateSpaceParams{
		ID:                    id,
		Name:                  name,
		Slug:                  slug,
		DefaultPermission:     sql.NullString{String: defaultPermission, Valid: defaultPermission != ""},
		McpWriteEnabled:       sql.NullInt64{Int64: mcp, Valid: true},
		SnapshotRetentionDays: retention,
	})
}

// Get fetches a space by ID.
func (r *Repository) Get(ctx context.Context, id string) (sqlc.Space, error) {
	return r.queries.GetSpace(ctx, id)
}

// GetBySlug fetches a space by slug.
func (r *Repository) GetBySlug(ctx context.Context, slug string) (sqlc.Space, error) {
	return r.queries.GetSpaceBySlug(ctx, slug)
}

// List returns all spaces ordered by name.
func (r *Repository) List(ctx context.Context) ([]sqlc.Space, error) {
	return r.queries.ListSpaces(ctx)
}

// Update modifies a space.
func (r *Repository) Update(ctx context.Context, id, name, defaultPermission string, mcpWriteEnabled bool, snapshotRetentionDays *int64) error {
	var mcp sql.NullInt64
	if mcpWriteEnabled {
		mcp = sql.NullInt64{Int64: 1, Valid: true}
	} else {
		mcp = sql.NullInt64{Int64: 0, Valid: true}
	}
	var retention sql.NullInt64
	if snapshotRetentionDays != nil {
		retention = sql.NullInt64{Int64: *snapshotRetentionDays, Valid: true}
	}
	return r.queries.UpdateSpace(ctx, sqlc.UpdateSpaceParams{
		Name:                  name,
		DefaultPermission:     sql.NullString{String: defaultPermission, Valid: defaultPermission != ""},
		McpWriteEnabled:       mcp,
		SnapshotRetentionDays: retention,
		ID:                    id,
	})
}

// Delete removes a space by ID.
func (r *Repository) Delete(ctx context.Context, id string) error {
	return r.queries.DeleteSpace(ctx, id)
}

// GenerateID returns a new UUID.
func GenerateID() string {
	return uuid.NewString()
}

// Now returns current UTC time.
func Now() time.Time {
	return time.Now().UTC()
}

// ScanSpace converts a sqlc Space to a domain Space.
func ScanSpace(s sqlc.Space) Space {
	var defaultPerm string
	if s.DefaultPermission.Valid {
		defaultPerm = s.DefaultPermission.String
	}
	var retentionDays *int64
	if s.SnapshotRetentionDays.Valid {
		v := s.SnapshotRetentionDays.Int64
		retentionDays = &v
	}
	mcpEnabled := s.McpWriteEnabled.Valid && s.McpWriteEnabled.Int64 == 1
	return Space{
		ID:                    s.ID,
		Name:                  s.Name,
		Slug:                  s.Slug,
		DefaultPermission:     defaultPerm,
		McpWriteEnabled:       mcpEnabled,
		SnapshotRetentionDays: retentionDays,
		CreatedAt:             s.CreatedAt.Int64,
	}
}
