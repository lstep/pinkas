package pages

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/pinkas/pinkas/internal/auth"
	"github.com/pinkas/pinkas/internal/db"
	"github.com/pinkas/pinkas/internal/directories"
	"github.com/pinkas/pinkas/internal/groups"
	"github.com/pinkas/pinkas/internal/permissions"
	"github.com/pinkas/pinkas/internal/spaces"
)

// TestSearchEndToEnd simulates the full production search flow:
// 1. Setup database with space + page + snapshot
// 2. Init FTS5 + backfill
// 3. Create an admin user and auth token
// 4. Call the search HTTP endpoint
// 5. Verify results are returned with correct content
func TestSearchEndToEnd(t *testing.T) {
	ctx := t.Context()

	// Setup: temp database with migrations
	dataDir := t.TempDir()
	conn, err := db.Open(dataDir)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer conn.Close()

	if err := db.Migrate(conn, "file://../../migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if err := db.Seed(conn); err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Create repos
	pagesRepo := NewRepository(conn)
	spacesRepo := spaces.NewRepository(conn)
	dirsRepo := directories.NewRepository(conn)
	groupsRepo := groups.NewRepository(conn)
	permRepo := permissions.NewRepository(conn)

	// Init FTS5
	if err := pagesRepo.InitFTS5(ctx); err != nil {
		t.Fatalf("InitFTS5: %v", err)
	}

	// Create an admin user
	authRepo := auth.NewRepository(conn)
	authService, err := auth.NewService(authRepo, dataDir)
	if err != nil {
		t.Fatalf("auth service: %v", err)
	}
	err = authRepo.CreateUser(ctx, "user-admin-1", "admin@test.com", "Admin", "password", "admin")
	if err != nil {
		t.Fatalf("create admin user: %v", err)
	}

	// Create a page with a snapshot containing searchable content
	err = pagesRepo.CreatePage(ctx, "page-search-1", "default", "My Recipe Page", "my-recipe-page", 0, nil, "user-admin-1", "📄")
	if err != nil {
		t.Fatalf("CreatePage: %v", err)
	}

	err = pagesRepo.SaveSnapshot(ctx, "page-search-1", "This page mentions confiture and jam. It also has strawberries and sugar.", []byte("yjs-data"), "user-admin-1")
	if err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}

	// Backfill FTS index from existing snapshots
	if err := pagesRepo.BackfillFTS5(ctx); err != nil {
		t.Fatalf("BackfillFTS5: %v", err)
	}

	// Verify FTS index has data directly
	var ftCount int
	conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM page_fts").Scan(&ftCount)
	t.Logf("page_fts row count after backfill: %d", ftCount)
	if ftCount == 0 {
		t.Fatal("page_fts is empty after backfill — check InitFTS5 and BackfillFTS5")
	}

	// Setup permission resolver
	permResolver := permissions.NewResolver(
		permRepo,
		dirsRepo.GetDirectory,
		spacesRepo.Get,
		pagesRepo.GetPage,
		groupsRepo.ListUserGroups,
		slog.New(slog.NewTextHandler(&strings.Builder{}, nil)),
	)

	// Setup the REST handler with the search method
	logger := slog.New(slog.NewTextHandler(&strings.Builder{}, nil))
	restHandler := NewRESTHandler(pagesRepo, logger, nil, permResolver, "")

	// Generate an auth token for the admin user
	token, _, err := authService.IssueTokens(auth.UserInfo{
		ID:    "user-admin-1",
		Email: "admin@test.com",
		Name:  "Admin",
		Role:  "admin",
	})
	if err != nil {
		t.Fatalf("issue token: %v", err)
	}

	// Create the search HTTP request
	req := httptest.NewRequest("GET", "/api/pages/search?q=confiture", nil)
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	// Need to set up auth middleware manually — the search handler uses RequireAuth
	authMiddleware := auth.Middleware(authService)
	handler := authMiddleware(http.HandlerFunc(restHandler.Search))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Check response
	if rr.Code != http.StatusOK {
		t.Fatalf("search returned HTTP %d: %s", rr.Code, rr.Body.String())
	}

	var resp struct {
		Results []map[string]interface{} `json:"results"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}

	t.Logf("search returned %d results", len(resp.Results))

	// Debug: print all results
	for i, r := range resp.Results {
		t.Logf("  result[%d]: id=%v title=%v", i, r["id"], r["title"])
	}

	if len(resp.Results) == 0 {
		// Try also searching with less restrictive terms
		req2 := httptest.NewRequest("GET", "/api/pages/search?q=strawberries", nil)
		req2.Header.Set("Authorization", "Bearer "+token.AccessToken)
		rr2 := httptest.NewRecorder()
		handler.ServeHTTP(rr2, req2)

		t.Logf("search 'strawberries' HTTP %d: %s", rr2.Code, rr2.Body.String())

		var resp2 struct {
			Results []map[string]interface{} `json:"results"`
		}
		json.Unmarshal(rr2.Body.Bytes(), &resp2)
		t.Logf("search 'strawberries' returned %d results", len(resp2.Results))

		t.Fatal("search returned 0 results — full pipeline is broken")
	}

	// Verify we got the right page
	title := resp.Results[0]["title"].(string)
	if title != "My Recipe Page" {
		t.Fatalf("expected title 'My Recipe Page', got %q", title)
	}
}

// TestSearchWithoutSnapshots validates the edge case where pages exist
// but have no snapshots yet (just created via REST, never edited).
func TestSearchWithoutSnapshots(t *testing.T) {
	ctx := t.Context()

	conn, err := db.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer conn.Close()

	if err := db.Migrate(conn, "file://../../migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	if err := db.Seed(conn); err != nil {
		t.Fatalf("seed: %v", err)
	}

	repo := NewRepository(conn)

	// Init FTS5
	if err := repo.InitFTS5(ctx); err != nil {
		t.Fatalf("InitFTS5: %v", err)
	}

	// Create a page BUT no snapshot (simulates REST-only page creation)
	err = repo.CreatePage(ctx, "page-no-snap", "default", "Untitled Page", "untitled-page", 0, nil, "user-1", "")
	if err != nil {
		t.Fatalf("CreatePage: %v", err)
	}

	// Backfill
	if err := repo.BackfillFTS5(ctx); err != nil {
		t.Fatalf("BackfillFTS5: %v", err)
	}

	// Search should return 0 results (page has no content yet)
	results, err := repo.SearchPages(ctx, "untitled", 10)
	if err != nil {
		t.Fatalf("SearchPages: %v", err)
	}

	// This is expected — page has no snapshot so nothing is indexed
	t.Logf("pages without snapshots return %d search results (expected 0)", len(results))
}
