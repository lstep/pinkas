package pages

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

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
	Label       string
	CreatedAt   int64
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

// SaveSnapshotWithLabel inserts a page snapshot with a label.
func (r *Repository) SaveSnapshotWithLabel(ctx context.Context, pageID, markdown string, yjsSnapshot []byte, authorID, label string) (string, error) {
	id := uuid.New().String()
	_, err := r.conn.ExecContext(ctx,
		"INSERT INTO page_snapshots (id, page_id, yjs_snapshot, markdown, author_id, label) VALUES (?, ?, ?, ?, ?, ?)",
		id, pageID, yjsSnapshot, markdown, authorID, label,
	)
	if err != nil {
		return "", fmt.Errorf("insert snapshot: %w", err)
	}
	return id, nil
}

// GetLatestSnapshot fetches the most recent snapshot for a page.
func (r *Repository) GetLatestSnapshot(ctx context.Context, pageID string) (*Snapshot, error) {
	var s Snapshot
	var compacted int
	var label sql.NullString
	var createdAt sql.NullInt64
	err := r.conn.QueryRowContext(ctx,
		"SELECT id, page_id, yjs_snapshot, markdown, author_id, is_compacted, label, created_at FROM page_snapshots WHERE page_id = ? ORDER BY created_at DESC LIMIT 1",
		pageID,
	).Scan(&s.ID, &s.PageID, &s.YjsSnapshot, &s.Markdown, &s.AuthorID, &compacted, &label, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest snapshot: %w", err)
	}
	s.IsCompacted = compacted != 0
	s.Label = label.String
	s.CreatedAt = createdAt.Int64
	return &s, nil
}

// ListSnapshots returns all snapshots for a page, ordered by created_at DESC.
func (r *Repository) ListSnapshots(ctx context.Context, pageID string) ([]*Snapshot, error) {
	rows, err := r.conn.QueryContext(ctx,
		"SELECT id, page_id, yjs_snapshot, markdown, author_id, is_compacted, label, created_at FROM page_snapshots WHERE page_id = ? ORDER BY created_at DESC",
		pageID,
	)
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}
	defer rows.Close()

	var result []*Snapshot
	for rows.Next() {
		var s Snapshot
		var compacted int
		var label sql.NullString
		var createdAt sql.NullInt64
		if err := rows.Scan(&s.ID, &s.PageID, &s.YjsSnapshot, &s.Markdown, &s.AuthorID, &compacted, &label, &createdAt); err != nil {
			return nil, fmt.Errorf("scan snapshot: %w", err)
		}
		s.IsCompacted = compacted != 0
		s.Label = label.String
		s.CreatedAt = createdAt.Int64
		result = append(result, &s)
	}
	return result, rows.Err()
}

// GetSnapshotByID fetches a single snapshot by its ID.
func (r *Repository) GetSnapshotByID(ctx context.Context, snapshotID string) (*Snapshot, error) {
	var s Snapshot
	var compacted int
	var label sql.NullString
	var createdAt sql.NullInt64
	err := r.conn.QueryRowContext(ctx,
		"SELECT id, page_id, yjs_snapshot, markdown, author_id, is_compacted, label, created_at FROM page_snapshots WHERE id = ?",
		snapshotID,
	).Scan(&s.ID, &s.PageID, &s.YjsSnapshot, &s.Markdown, &s.AuthorID, &compacted, &label, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}
	s.IsCompacted = compacted != 0
	s.Label = label.String
	s.CreatedAt = createdAt.Int64
	return &s, nil
}

// DeleteSnapshotsForPage deletes all snapshots for a page.
func (r *Repository) DeleteSnapshotsForPage(ctx context.Context, pageID string) error {
	_, err := r.conn.ExecContext(ctx, "DELETE FROM page_snapshots WHERE page_id = ?", pageID)
	if err != nil {
		return fmt.Errorf("delete snapshots: %w", err)
	}
	return nil
}

// ListPagesWithSnapshots returns all distinct page IDs that have Yjs snapshots.
func (r *Repository) ListPagesWithSnapshots(ctx context.Context) ([]string, error) {
	rows, err := r.conn.QueryContext(ctx,
		"SELECT DISTINCT page_id FROM page_snapshots WHERE yjs_snapshot IS NOT NULL AND yjs_snapshot != ''",
	)
	if err != nil {
		return nil, fmt.Errorf("list pages with snapshots: %w", err)
	}
	defer rows.Close()

	var pageIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan page id: %w", err)
		}
		pageIDs = append(pageIDs, id)
	}
	return pageIDs, rows.Err()
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

// SearchResult represents a page search result with markdown content.
type SearchResult struct {
	Page     sqlc.Page
	Markdown string
}

// escapeFTS5 escapes a query string for safe use in FTS5 MATCH.
// It wraps the query in double quotes and escapes internal quotes.
func escapeFTS5(query string) string {
	if query == "" {
		return ""
	}
	// Escape double quotes by doubling them
	escaped := strings.ReplaceAll(query, `"`, `""`)
	// Wrap in double quotes for phrase matching
	return `"` + escaped + `"`
}

// SearchPages searches pages using FTS5 full-text search.
// Returns empty results if FTS5 is not available (page_fts table doesn't exist).
func (r *Repository) SearchPages(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	if query == "" {
		return []SearchResult{}, nil
	}

	escapedQuery := escapeFTS5(query)

	// Use raw SQL for FTS5 search since sqlc doesn't support virtual tables
	// Deduplicate by page_id using GROUP BY with MIN(rank) to keep the best match.
	rows, err := r.conn.QueryContext(ctx, `
		SELECT p.id, p.space_id, p.directory_id, p.title, p.slug, p.position, p.created_by, p.created_at, p.updated_at, p.icon,
		       (SELECT markdown FROM page_snapshots WHERE page_id = p.id ORDER BY created_at DESC LIMIT 1) AS markdown
		FROM (
			SELECT page_id, MIN(rank) AS rank
			FROM page_fts
			WHERE page_fts MATCH ?
			GROUP BY page_id
		) AS matched
		JOIN pages p ON p.id = matched.page_id
		ORDER BY matched.rank
		LIMIT ?
	`, escapedQuery, limit)
	if err != nil {
		// Gracefully handle case where page_fts doesn't exist (FTS5 not available)
		if isTableNotExistError(err) {
			return []SearchResult{}, nil
		}
		return nil, fmt.Errorf("search pages: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var p sqlc.Page
		var markdown sql.NullString
		err := rows.Scan(
			&p.ID, &p.SpaceID, &p.DirectoryID, &p.Title, &p.Slug, &p.Position,
			&p.CreatedBy, &p.CreatedAt, &p.UpdatedAt, &p.Icon, &markdown,
		)
		if err != nil {
			return nil, fmt.Errorf("scan search result: %w", err)
		}
		results = append(results, SearchResult{
			Page:     p,
			Markdown: markdown.String,
		})
	}

	return results, rows.Err()
}

// isTableNotExistError checks if an error is due to a table not existing.
func isTableNotExistError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// SQLite error messages for missing table
	return strings.Contains(errStr, "no such table") || strings.Contains(errStr, "page_fts")
}

// InitFTS5 initializes the FTS5 virtual table and triggers if FTS5 is available.
// This should be called at application startup after migrations have run.
func (r *Repository) InitFTS5(ctx context.Context) error {
	// Check if FTS5 is available
	var hasFTS5 int
	err := r.conn.QueryRowContext(ctx, "SELECT CASE WHEN EXISTS (SELECT 1 FROM pragma_compile_options WHERE compile_options LIKE 'ENABLE_FTS5%') THEN 1 ELSE 0 END").Scan(&hasFTS5)
	if err != nil {
		return fmt.Errorf("check fts5 availability: %w", err)
	}
	if hasFTS5 == 0 {
		// FTS5 not available, skip initialization
		return nil
	}

	// Create FTS5 virtual table
	_, err = r.conn.ExecContext(ctx, `
		CREATE VIRTUAL TABLE IF NOT EXISTS page_fts USING fts5(
			page_id UNINDEXED,
			title,
			content,
			tokenize='porter'
		)
	`)
	if err != nil {
		return fmt.Errorf("create page_fts table: %w", err)
	}

	// Create trigger for INSERT on page_snapshots
	_, err = r.conn.ExecContext(ctx, `
		CREATE TRIGGER IF NOT EXISTS page_fts_ai
		AFTER INSERT ON page_snapshots
		BEGIN
			INSERT OR REPLACE INTO page_fts (page_id, title, content)
			SELECT 
				new.page_id,
				p.title,
				new.markdown
			FROM pages p
			WHERE p.id = new.page_id;
		END
	`)
	if err != nil {
		return fmt.Errorf("create page_fts_ai trigger: %w", err)
	}

	// Create trigger for DELETE on page_snapshots
	_, err = r.conn.ExecContext(ctx, `
		CREATE TRIGGER IF NOT EXISTS page_fts_ad
		AFTER DELETE ON page_snapshots
		BEGIN
			DELETE FROM page_fts
			WHERE page_id = old.page_id
			AND NOT EXISTS (
				SELECT 1 FROM page_snapshots WHERE page_id = old.page_id LIMIT 1
			);
		END
	`)
	if err != nil {
		return fmt.Errorf("create page_fts_ad trigger: %w", err)
	}

	// Create trigger for UPDATE on pages
	_, err = r.conn.ExecContext(ctx, `
		CREATE TRIGGER IF NOT EXISTS page_fts_au_pages
		AFTER UPDATE OF title ON pages
		BEGIN
			UPDATE page_fts SET title = new.title WHERE page_id = new.id;
		END
	`)
	if err != nil {
		return fmt.Errorf("create page_fts_au_pages trigger: %w", err)
	}

	return nil
}

// BackfillFTS5 populates the FTS5 index from existing page snapshots.
// Should be called after InitFTS5 to index content that was saved before the
// FTS5 table existed.
func (r *Repository) BackfillFTS5(ctx context.Context) error {
	_, err := r.conn.ExecContext(ctx, `
		INSERT OR REPLACE INTO page_fts (page_id, title, content)
		SELECT p.id, p.title, ps.markdown
		FROM page_snapshots ps
		JOIN pages p ON p.id = ps.page_id
		WHERE ps.id IN (
			SELECT id FROM page_snapshots
			WHERE page_id = ps.page_id
			ORDER BY created_at DESC
			LIMIT 1
		)
	`)
	if err != nil {
		if isTableNotExistError(err) {
			return nil // FTS5 not available, skip
		}
		return fmt.Errorf("backfill fts5: %w", err)
	}
	return nil
}
