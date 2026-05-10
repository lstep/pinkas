package pages

import (
	"testing"

	"github.com/pinkas/pinkas/internal/db"
)

func TestFTS5Search(t *testing.T) {
	ctx := t.Context()

	// Setup: temp database with migrations
	conn, err := db.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer conn.Close()

	if err := db.Migrate(conn, "file://../../migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewRepository(conn)

	// Step 1: Check if FTS5 is available in the SQLite build
	var hasFTS5 int
	err = conn.QueryRowContext(ctx,
		"SELECT CASE WHEN EXISTS (SELECT 1 FROM pragma_compile_options WHERE compile_options LIKE 'ENABLE_FTS5%') THEN 1 ELSE 0 END",
	).Scan(&hasFTS5)
	if err != nil {
		t.Fatalf("check fts5 availability: %v", err)
	}
	t.Logf("FTS5 compiled into driver: %v", hasFTS5 == 1)
	if hasFTS5 == 0 {
		t.Fatal("FTS5 is NOT available — go-sqlite3 needs -tags fts5 build flag")
	}

	// Step 2: Initialize FTS5 table + triggers
	if err := repo.InitFTS5(ctx); err != nil {
		t.Fatalf("InitFTS5: %v", err)
	}

	// Step 3: Create a page with a snapshot containing searchable content
	err = repo.CreatePage(ctx, "page-1", "space-1", "My Page", "my-page", 0, nil, "user-1", "")
	if err != nil {
		t.Fatalf("CreatePage: %v", err)
	}

	err = repo.SaveSnapshot(ctx, "page-1", "This page mentions confiture and jam", []byte("yjs-data"), "user-1")
	if err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}

	// Step 4: Backfill FTS index
	if err := repo.BackfillFTS5(ctx); err != nil {
		t.Fatalf("BackfillFTS5: %v", err)
	}

	// Step 5: Search for the word
	results, err := repo.SearchPages(ctx, "confiture", 10)
	if err != nil {
		t.Fatalf("SearchPages: %v", err)
	}

	if len(results) == 0 {
		// Diagnose: check if page_fts has any rows
		var count int
		conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM page_fts").Scan(&count)
		t.Logf("page_fts row count: %d", count)

		// Check if the snapshot trigger populated it
		var triggerCount int
		conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM page_snapshots").Scan(&triggerCount)
		t.Logf("page_snapshots row count: %d", triggerCount)

		t.Fatal("SearchPages returned 0 results — FTS5 index is empty or query failed")
	}

	if results[0].Markdown != "This page mentions confiture and jam" {
		t.Fatalf("expected markdown match, got %q", results[0].Markdown)
	}
}

func TestFTS5BackfillExistingContent(t *testing.T) {
	ctx := t.Context()

	conn, err := db.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer conn.Close()

	if err := db.Migrate(conn, "file://../../migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewRepository(conn)

	// Check FTS5 availability
	var hasFTS5 int
	conn.QueryRowContext(ctx,
		"SELECT CASE WHEN EXISTS (SELECT 1 FROM pragma_compile_options WHERE compile_options LIKE 'ENABLE_FTS5%') THEN 1 ELSE 0 END",
	).Scan(&hasFTS5)
	if hasFTS5 == 0 {
		t.Skip("FTS5 not compiled into the SQLite driver")
	}

	// Create page + snapshot BEFORE InitFTS5 (simulating existing content)
	err = repo.CreatePage(ctx, "page-2", "space-1", "Recipes", "recipes", 0, nil, "user-1", "")
	if err != nil {
		t.Fatalf("CreatePage: %v", err)
	}
	err = repo.SaveSnapshot(ctx, "page-2", "Strawberry confiture recipe", []byte("yjs-data"), "user-1")
	if err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}

	// Now init FTS5 (as happens on app restart)
	if err := repo.InitFTS5(ctx); err != nil {
		t.Fatalf("InitFTS5: %v", err)
	}

	// Backfill should pick up the existing snapshot
	if err := repo.BackfillFTS5(ctx); err != nil {
		t.Fatalf("BackfillFTS5: %v", err)
	}

	// Search for content that existed before FTS5 was created
	results, err := repo.SearchPages(ctx, "confiture", 10)
	if err != nil {
		t.Fatalf("SearchPages: %v", err)
	}

	if len(results) == 0 {
		var ftsCount int
		conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM page_fts").Scan(&ftsCount)
		t.Logf("page_fts rows after backfill: %d", ftsCount)
		t.Fatal("BackfillFTS5 did not populate existing content into the FTS index")
	}
}

func TestFTS5TriggerOnNewSnapshot(t *testing.T) {
	ctx := t.Context()

	conn, err := db.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer conn.Close()

	if err := db.Migrate(conn, "file://../../migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewRepository(conn)

	var hasFTS5 int
	conn.QueryRowContext(ctx,
		"SELECT CASE WHEN EXISTS (SELECT 1 FROM pragma_compile_options WHERE compile_options LIKE 'ENABLE_FTS5%') THEN 1 ELSE 0 END",
	).Scan(&hasFTS5)
	if hasFTS5 == 0 {
		t.Skip("FTS5 not compiled into the SQLite driver")
	}

	// Init FTS5 first
	if err := repo.InitFTS5(ctx); err != nil {
		t.Fatalf("InitFTS5: %v", err)
	}

	// Create page then save snapshot (trigger should fire)
	err = repo.CreatePage(ctx, "page-3", "space-1", "Cooking", "cooking", 0, nil, "user-1", "")
	if err != nil {
		t.Fatalf("CreatePage: %v", err)
	}
	err = repo.SaveSnapshot(ctx, "page-3", "How to make confiture", []byte("yjs-data"), "user-1")
	if err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}

	// Search — should find it without needing backfill
	results, err := repo.SearchPages(ctx, "confiture", 10)
	if err != nil {
		t.Fatalf("SearchPages: %v", err)
	}

	if len(results) == 0 {
		// Check if trigger even fired
		var ftsCount int
		conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM page_fts").Scan(&ftsCount)
		t.Logf("page_fts rows after insert trigger: %d", ftsCount)
		t.Fatal("page_fts_ai trigger did not populate the FTS index")
	}
}

func TestFTS5SearchPermissionFiltering(t *testing.T) {
	t.Skip("Requires setting up auth + permission resolver — covered by integration test")
}
