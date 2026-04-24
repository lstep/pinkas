package pages

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	sqlc "github.com/mostdoc/mostdoc/internal/db/query"
)

// Snapshot represents a page snapshot.
type Snapshot struct {
	ID          string
	PageID      string
	YjsSnapshot []byte
	Markdown    string
	AuthorID    string
	IsCompacted bool
}

// Repository wraps sqlc queries for page operations.
type Repository struct {
	queries *sqlc.Queries
	conn    *sql.DB
}

// NewRepository creates a new page repository.
func NewRepository(conn *sql.DB) *Repository {
	return &Repository{queries: sqlc.New(conn), conn: conn}
}

// GetPage fetches a page by ID.
func (r *Repository) GetPage(ctx context.Context, id string) (sqlc.Page, error) {
	return r.queries.GetPage(ctx, id)
}

// GetPageBySlug fetches a page by space and slug.
func (r *Repository) GetPageBySlug(ctx context.Context, spaceID, slug string) (sqlc.Page, error) {
	return r.queries.GetPageBySlug(ctx, sqlc.GetPageBySlugParams{
		SpaceID: sql.NullString{String: spaceID, Valid: spaceID != ""},
		Slug:    sql.NullString{String: slug, Valid: slug != ""},
	})
}

// CreatePage inserts a new page.
func (r *Repository) CreatePage(ctx context.Context, id, spaceID, title, slug string, position int64, directoryID *string, createdBy, icon string) error {
	var space sql.NullString
	if spaceID != "" {
		space = sql.NullString{String: spaceID, Valid: true}
	}
	var dir sql.NullString
	if directoryID != nil && *directoryID != "" {
		dir = sql.NullString{String: *directoryID, Valid: true}
	}
	return r.queries.CreatePage(ctx, sqlc.CreatePageParams{
		ID:          id,
		SpaceID:     space,
		DirectoryID: dir,
		Title:       sql.NullString{String: title, Valid: true},
		Slug:        sql.NullString{String: slug, Valid: true},
		Position:    sql.NullInt64{Int64: position, Valid: true},
		CreatedBy:   sql.NullString{String: createdBy, Valid: createdBy != ""},
		Icon:        sql.NullString{String: icon, Valid: icon != ""},
	})
}

// UpdatePage updates a page's title and slug.
func (r *Repository) UpdatePage(ctx context.Context, id, title, slug string) error {
	return r.queries.UpdatePageTitle(ctx, sqlc.UpdatePageTitleParams{
		Title: sql.NullString{String: title, Valid: true},
		Slug:  sql.NullString{String: slug, Valid: true},
		ID:    id,
	})
}

// UpdatePagePosition updates a page's position and directory.
func (r *Repository) UpdatePagePosition(ctx context.Context, id string, position int64, directoryID *string) error {
	var dir sql.NullString
	if directoryID != nil && *directoryID != "" {
		dir = sql.NullString{String: *directoryID, Valid: true}
	}
	return r.queries.UpdatePagePosition(ctx, sqlc.UpdatePagePositionParams{
		Position:    sql.NullInt64{Int64: position, Valid: true},
		DirectoryID: dir,
		ID:          id,
	})
}

// UpdatePageIcon updates a page's icon.
func (r *Repository) UpdatePageIcon(ctx context.Context, id, icon string) error {
	return r.queries.UpdatePageIcon(ctx, sqlc.UpdatePageIconParams{
		Icon: sql.NullString{String: icon, Valid: true},
		ID:   id,
	})
}

// DeletePage removes a page by ID.
func (r *Repository) DeletePage(ctx context.Context, id string) error {
	return r.queries.DeletePage(ctx, id)
}

// ListRootPages lists root pages for a space (pages with no directory).
func (r *Repository) ListRootPages(ctx context.Context, spaceID string) ([]sqlc.Page, error) {
	return r.queries.ListRootPages(ctx, sql.NullString{String: spaceID, Valid: spaceID != ""})
}

// ListPagesByDirectory lists pages inside a directory.
func (r *Repository) ListPagesByDirectory(ctx context.Context, directoryID string) ([]sqlc.Page, error) {
	var dir sql.NullString
	if directoryID != "" {
		dir = sql.NullString{String: directoryID, Valid: true}
	}
	return r.queries.ListPagesByDirectory(ctx, dir)
}

// GetMaxPosition returns the max position for pages in a directory.
func (r *Repository) GetMaxPosition(ctx context.Context, directoryID *string) (int64, error) {
	var dir sql.NullString
	if directoryID != nil && *directoryID != "" {
		dir = sql.NullString{String: *directoryID, Valid: true}
	}
	result, err := r.queries.GetMaxPagePosition(ctx, dir)
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

// GetPagesBySlugPrefix returns pages with slugs matching a prefix.
func (r *Repository) GetPagesBySlugPrefix(ctx context.Context, spaceID, slugPrefix string) ([]sqlc.Page, error) {
	return r.queries.GetPagesBySlugPrefix(ctx, sqlc.GetPagesBySlugPrefixParams{
		SpaceID: sql.NullString{String: spaceID, Valid: spaceID != ""},
		Slug:    sql.NullString{String: slugPrefix + "%", Valid: true},
	})
}

// SaveSnapshot inserts a page snapshot.
func (r *Repository) SaveSnapshot(ctx context.Context, pageID, markdown string, yjsSnapshot []byte, authorID string) error {
	id := uuid.New().String()
	_, err := r.conn.ExecContext(ctx,
		"INSERT INTO page_snapshots (id, page_id, yjs_snapshot, markdown, author_id) VALUES (?, ?, ?, ?, ?)",
		id, pageID, yjsSnapshot, markdown, authorID,
	)
	if err != nil {
		return fmt.Errorf("insert snapshot: %w", err)
	}
	return nil
}

// GetLatestSnapshot fetches the most recent snapshot for a page.
func (r *Repository) GetLatestSnapshot(ctx context.Context, pageID string) (*Snapshot, error) {
	var s Snapshot
	var compacted int
	err := r.conn.QueryRowContext(ctx,
		"SELECT id, page_id, yjs_snapshot, markdown, author_id, is_compacted FROM page_snapshots WHERE page_id = ? ORDER BY created_at DESC LIMIT 1",
		pageID,
	).Scan(&s.ID, &s.PageID, &s.YjsSnapshot, &s.Markdown, &s.AuthorID, &compacted)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest snapshot: %w", err)
	}
	s.IsCompacted = compacted != 0
	return &s, nil
}

// GetAncestors walks up the directory_id chain and returns all ancestor directories
// from root to the page's directory (not including the page itself).
func (r *Repository) GetAncestors(ctx context.Context, pageID string) ([]sqlc.Directory, error) {
	// First get the page to find its directory
	page, err := r.queries.GetPage(ctx, pageID)
	if err != nil {
		return nil, err
	}

	// If page has no directory, return empty
	if !page.DirectoryID.Valid || page.DirectoryID.String == "" {
		return nil, nil
	}

	// Walk up the directory parent chain
	var ancestors []sqlc.Directory
	currentDirID := page.DirectoryID.String
	visited := make(map[string]bool)

	for {
		if visited[currentDirID] {
			break // circular reference guard
		}
		visited[currentDirID] = true

		dir, err := r.queries.GetDirectory(ctx, currentDirID)
		if err != nil {
			return nil, err
		}

		// Prepend to maintain root-to-leaf order
		ancestors = append([]sqlc.Directory{dir}, ancestors...)

		if !dir.ParentID.Valid || dir.ParentID.String == "" {
			break
		}
		currentDirID = dir.ParentID.String
	}

	return ancestors, nil
}
