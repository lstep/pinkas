package pages

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mostdoc/mostdoc/internal/auth"
	"github.com/mostdoc/mostdoc/internal/db"
)

func setupTestHandler(t *testing.T) (*Handler, *Repository) {
	t.Helper()

	dataDir := t.TempDir()
	conn, err := db.Open(dataDir)
	if err != nil {
		t.Fatalf("open test db: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	if err := db.Migrate(conn, "file://../../migrations"); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := NewRepository(conn)
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	authService, _ := auth.NewService(auth.NewRepository(conn), "")
	return &Handler{repo: repo, logger: logger, dataDir: dataDir, authService: authService}, repo
}

// TestLoadWithNoSnapshot verifies the nil pointer regression fix.
// Before the fix, GetLatestSnapshot returned (nil, nil) for a new page,
// and the handler panicked trying to access snapshot.YjsSnapshot.
func TestLoadWithNoSnapshot(t *testing.T) {
	handler, repo := setupTestHandler(t)

	// Create a page but no snapshot
	ctx := t.Context()
	repo.CreatePage(ctx, "test-page-1", "default", "Test", "test", 0, nil, "user-1", "")

	req := httptest.NewRequest("GET", "/internal/load?docId=test-page-1", nil)
	rr := httptest.NewRecorder()

	handler.Load(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}

	if resp["yjsSnapshot"] != nil {
		t.Fatalf("expected nil snapshot, got %v", resp["yjsSnapshot"])
	}
}

func TestLoadWithSnapshot(t *testing.T) {
	handler, repo := setupTestHandler(t)

	ctx := t.Context()
	repo.CreatePage(ctx, "test-page-2", "default", "Test", "test", 0, nil, "user-1", "")
	repo.SaveSnapshot(ctx, "test-page-2", "# Hello", []byte("yjs-data"), "user-1")

	req := httptest.NewRequest("GET", "/internal/load?docId=test-page-2", nil)
	rr := httptest.NewRecorder()

	handler.Load(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}

	snapshot, ok := resp["yjsSnapshot"].(string)
	if !ok || snapshot == "" {
		t.Fatalf("expected non-empty snapshot, got %v", resp["yjsSnapshot"])
	}
}

func TestAuthWithMissingToken(t *testing.T) {
	handler, _ := setupTestHandler(t)

	req := httptest.NewRequest("GET", "/internal/auth", nil)
	rr := httptest.NewRecorder()

	handler.Auth(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestHealth(t *testing.T) {
	handler, _ := setupTestHandler(t)

	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	handler.Health(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Fatalf("expected status=ok, got %s", resp["status"])
	}
}
