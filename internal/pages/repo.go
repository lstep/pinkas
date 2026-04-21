package pages

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type Snapshot struct {
	ID           string
	PageID       string
	YjsSnapshot  []byte
	Markdown     string
	AuthorID     string
	IsCompacted  bool
}

type SQLiteDB struct {
	conn *sql.DB
}

func NewSQLiteDB(conn *sql.DB) *SQLiteDB {
	return &SQLiteDB{conn: conn}
}

func (d *SQLiteDB) GetPage(id string) (Page, error) {
	var p Page
	err := d.conn.QueryRow("SELECT id, title, slug FROM pages WHERE id = ?", id).Scan(&p.ID, &p.Title, &p.Slug)
	if err != nil {
		return Page{}, fmt.Errorf("get page: %w", err)
	}
	return p, nil
}

func (d *SQLiteDB) SaveSnapshot(pageID, markdown string, yjsSnapshot []byte, authorID string) error {
	id := uuid.New().String()
	_, err := d.conn.Exec(
		"INSERT INTO page_snapshots (id, page_id, yjs_snapshot, markdown, author_id) VALUES (?, ?, ?, ?, ?)",
		id, pageID, yjsSnapshot, markdown, authorID,
	)
	if err != nil {
		return fmt.Errorf("insert snapshot: %w", err)
	}
	return nil
}

func (d *SQLiteDB) GetLatestSnapshot(pageID string) (*Snapshot, error) {
	var s Snapshot
	var compacted int
	err := d.conn.QueryRow(
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
