package directories

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	sqlc "github.com/mostdoc/mostdoc/internal/db/query"
)

// Repository wraps sqlc queries for directory operations.
type Repository struct {
	queries *sqlc.Queries
	conn    *sql.DB
}

// NewRepository creates a new directory repository.
func NewRepository(conn *sql.DB) *Repository {
	return &Repository{queries: sqlc.New(conn), conn: conn}
}

// CreateDirectory creates a new directory.
func (r *Repository) CreateDirectory(ctx context.Context, id, name, slug, spaceID string, parentID *string, position int64, icon, createdBy string) error {
	var space sql.NullString
	if spaceID != "" {
		space = sql.NullString{String: spaceID, Valid: true}
	}
	var parent sql.NullString
	if parentID != nil && *parentID != "" {
		parent = sql.NullString{String: *parentID, Valid: true}
	}
	var createdByNull sql.NullString
	if createdBy != "" {
		createdByNull = sql.NullString{String: createdBy, Valid: true}
	}
	var iconNull sql.NullString
	if icon != "" {
		iconNull = sql.NullString{String: icon, Valid: true}
	}
	return r.queries.CreateDirectory(ctx, sqlc.CreateDirectoryParams{
		ID:        id,
		SpaceID:   space,
		ParentID:  parent,
		Name:      name,
		Slug:      slug,
		Position:  sql.NullInt64{Int64: position, Valid: true},
		Icon:      iconNull,
		CreatedBy: createdByNull,
	})
}

// GetDirectory fetches a directory by ID.
func (r *Repository) GetDirectory(ctx context.Context, id string) (sqlc.Directory, error) {
	return r.queries.GetDirectory(ctx, id)
}

// GetDirectoryBySlug fetches a directory by space and slug.
func (r *Repository) GetDirectoryBySlug(ctx context.Context, spaceID, slug string) (sqlc.Directory, error) {
	return r.queries.GetDirectoryBySlug(ctx, sqlc.GetDirectoryBySlugParams{
		SpaceID: sql.NullString{String: spaceID, Valid: spaceID != ""},
		Slug:    slug,
	})
}

// UpdateDirectory updates a directory's name, slug, and icon.
func (r *Repository) UpdateDirectory(ctx context.Context, id, name, slug, icon string) error {
	if err := r.queries.UpdateDirectoryName(ctx, sqlc.UpdateDirectoryNameParams{
		Name: name,
		Slug: slug,
		ID:   id,
	}); err != nil {
		return err
	}
	if icon != "" {
		if err := r.queries.UpdateDirectoryIcon(ctx, sqlc.UpdateDirectoryIconParams{
			Icon: sql.NullString{String: icon, Valid: true},
			ID:   id,
		}); err != nil {
			return err
		}
	}
	return nil
}

// DeleteDirectory removes a directory by ID.
func (r *Repository) DeleteDirectory(ctx context.Context, id string) error {
	return r.queries.DeleteDirectory(ctx, id)
}

// ListRootDirectories lists root directories for a space.
func (r *Repository) ListRootDirectories(ctx context.Context, spaceID string) ([]sqlc.Directory, error) {
	return r.queries.ListRootDirectories(ctx, sql.NullString{String: spaceID, Valid: spaceID != ""})
}

// ListChildren lists direct children of a directory.
func (r *Repository) ListChildren(ctx context.Context, parentID string) ([]sqlc.Directory, error) {
	var parent sql.NullString
	if parentID != "" {
		parent = sql.NullString{String: parentID, Valid: true}
	}
	return r.queries.ListDirectorySubdirectories(ctx, parent)
}

// GetMaxPosition returns the max position for children of a parent directory.
func (r *Repository) GetMaxPosition(ctx context.Context, spaceID string, parentID *string) (int64, error) {
	var parent sql.NullString
	if parentID != nil && *parentID != "" {
		parent = sql.NullString{String: *parentID, Valid: true}
	}
	result, err := r.queries.GetMaxDirectoryPosition(ctx, parent)
	if err != nil {
		return -1, err
	}
	// sqlc returns interface{} for COALESCE with sqlite
	switch v := result.(type) {
	case int64:
		return v, nil
	case float64:
		return int64(v), nil
	case int:
		return int64(v), nil
	default:
		return -1, fmt.Errorf("unexpected type for max position: %T", result)
	}
}

// UpdateDirectoryPosition updates a directory's position and parent.
func (r *Repository) UpdateDirectoryPosition(ctx context.Context, id string, position int64, parentID *string) error {
	var parent sql.NullString
	if parentID != nil && *parentID != "" {
		parent = sql.NullString{String: *parentID, Valid: true}
	}
	return r.queries.UpdateDirectoryPosition(ctx, sqlc.UpdateDirectoryPositionParams{
		Position: sql.NullInt64{Int64: position, Valid: true},
		ParentID: parent,
		ID:       id,
	})
}

// GetAncestors walks up the parent_id chain and returns all ancestors
// from root to parent (not including the starting directory).
func (r *Repository) GetAncestors(ctx context.Context, directoryID string) ([]sqlc.Directory, error) {
	var ancestors []sqlc.Directory
	currentID := directoryID
	visited := make(map[string]bool)
	for {
		if visited[currentID] {
			break // circular reference guard
		}
		visited[currentID] = true
		dir, err := r.queries.GetDirectory(ctx, currentID)
		if err != nil {
			return nil, err
		}
		if !dir.ParentID.Valid || dir.ParentID.String == "" {
			break
		}
		parent, err := r.queries.GetDirectory(ctx, dir.ParentID.String)
		if err != nil {
			return nil, err
		}
		ancestors = append([]sqlc.Directory{parent}, ancestors...)
		currentID = parent.ID
	}
	return ancestors, nil
}

// GenerateID returns a new UUID.
func GenerateID() string {
	return uuid.NewString()
}
