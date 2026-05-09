package mcptokens

import (
	"testing"
	"time"

	"github.com/mostdoc/mostdoc/internal/db"
)

func TestCreateAndValidateToken(t *testing.T) {
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
	svc := NewService(repo)

	// Create a token
	resp, err := svc.CreateToken(ctx, "user-1", CreateTokenRequest{
		Name:   "Test Token",
		Scopes: []string{ScopeRead, ScopeWrite},
	})
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	if resp.Token.Name != "Test Token" {
		t.Errorf("expected name Test Token, got %s", resp.Token.Name)
	}
	if resp.Token.UserID != "user-1" {
		t.Errorf("expected user user-1, got %s", resp.Token.UserID)
	}
	if resp.Token.TokenHash == "" {
		t.Error("expected non-empty token hash")
	}
	if resp.Secret == "" {
		t.Error("expected non-empty secret")
	}

	// Validate the token
	validated, err := svc.ValidateToken(ctx, resp.Secret)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if validated.ID != resp.Token.ID {
		t.Errorf("expected id %s, got %s", resp.Token.ID, validated.ID)
	}
}

func TestListAndDeleteTokens(t *testing.T) {
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
	svc := NewService(repo)

	// Create 2 tokens for user-1
	_, err = svc.CreateToken(ctx, "user-1", CreateTokenRequest{Name: "Token A", Scopes: []string{ScopeRead}})
	if err != nil {
		t.Fatalf("CreateToken A: %v", err)
	}
	_, err = svc.CreateToken(ctx, "user-1", CreateTokenRequest{Name: "Token B", Scopes: []string{ScopeWrite}})
	if err != nil {
		t.Fatalf("CreateToken B: %v", err)
	}

	// Create 1 token for user-2
	_, err = svc.CreateToken(ctx, "user-2", CreateTokenRequest{Name: "Token C", Scopes: []string{ScopeRead}})
	if err != nil {
		t.Fatalf("CreateToken C: %v", err)
	}

	// List tokens for user-1
	tokens, err := svc.ListTokens(ctx, "user-1")
	if err != nil {
		t.Fatalf("ListTokens: %v", err)
	}
	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens for user-1, got %d", len(tokens))
	}

	// List tokens for user-2
	tokens, err = svc.ListTokens(ctx, "user-2")
	if err != nil {
		t.Fatalf("ListTokens: %v", err)
	}
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token for user-2, got %d", len(tokens))
	}

	// Delete token B
	err = svc.DeleteToken(ctx, tokens[0].ID, "user-2")
	if err != nil {
		t.Fatalf("DeleteToken: %v", err)
	}

	// Verify deletion
	tokens, err = svc.ListTokens(ctx, "user-2")
	if err != nil {
		t.Fatalf("ListTokens after delete: %v", err)
	}
	if len(tokens) != 0 {
		t.Fatalf("expected 0 tokens after delete, got %d", len(tokens))
	}
}

func TestExpiredToken(t *testing.T) {
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
	svc := NewService(repo)

	// Create a token without expiry
	resp, err := svc.CreateToken(ctx, "user-1", CreateTokenRequest{
		Name:   "Expired Token Test",
		Scopes: []string{ScopeRead},
		// no ExpiresInDays set — no expiry by default
	})
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	// Manually set the token as expired in the database
	_, err = conn.ExecContext(ctx, "UPDATE mcp_tokens SET expires_at = 1 WHERE id = ?", resp.Token.ID)
	if err != nil {
		t.Fatalf("update expiration: %v", err)
	}

	// Validate should fail — token is expired
	_, err = svc.ValidateToken(ctx, resp.Secret)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestHasScope(t *testing.T) {
	token := &MCPToken{Scopes: `["read","write"]`}

	if !HasScope(token, ScopeRead) {
		t.Error("expected HasScope(read) = true")
	}
	if !HasScope(token, ScopeWrite) {
		t.Error("expected HasScope(write) = true")
	}
	if HasScope(token, ScopeAdmin) {
		t.Error("expected HasScope(admin) = false")
	}

	// Admin scope should match everything
	adminToken := &MCPToken{Scopes: `["admin"]`}
	if !HasScope(adminToken, ScopeRead) {
		t.Error("expected admin token HasScope(read) = true")
	}
	if !HasScope(adminToken, ScopeWrite) {
		t.Error("expected admin token HasScope(write) = true")
	}
	if !HasScope(adminToken, ScopeAdmin) {
		t.Error("expected admin token HasScope(admin) = true")
	}
}

func TestHasAccessToSpace(t *testing.T) {
	unrestricted := &MCPToken{SpaceID: ""}
	if !HasAccessToSpace(unrestricted, "space-1") {
		t.Error("expected unrestricted token HasAccessToSpace = true")
	}

	restricted := &MCPToken{SpaceID: "space-1"}
	if !HasAccessToSpace(restricted, "space-1") {
		t.Error("expected restricted token HasAccessToSpace(space-1) = true")
	}
	if HasAccessToSpace(restricted, "space-2") {
		t.Error("expected restricted token HasAccessToSpace(space-2) = false")
	}
}

func TestInvalidTokenFormat(t *testing.T) {
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
	svc := NewService(repo)

	_, err = svc.ValidateToken(ctx, "invalid-token-format")
	if err == nil {
		t.Fatal("expected error for invalid token format, got nil")
	}

	_, err = svc.ValidateToken(ctx, "")
	if err == nil {
		t.Fatal("expected error for empty token, got nil")
	}

	// Valid format but non-existent ID
	_, err = svc.ValidateToken(ctx, "mcp_nonexistent123_secret")
	if err == nil {
		t.Fatal("expected error for non-existent token, got nil")
	}
}

func TestCreateTokenDefaultScopes(t *testing.T) {
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
	svc := NewService(repo)

	// Create token without specifying scopes — should default to ["read"]
	resp, err := svc.CreateToken(ctx, "user-1", CreateTokenRequest{
		Name: "Default Scopes Token",
	})
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	if !HasScope(&resp.Token, ScopeRead) {
		t.Error("expected default scope to be read")
	}
	if HasScope(&resp.Token, ScopeWrite) {
		t.Error("expected no write scope by default")
	}
}

func TestUpdateLastUsed(t *testing.T) {
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
	svc := NewService(repo)

	resp, err := svc.CreateToken(ctx, "user-1", CreateTokenRequest{
		Name:   "Last Used Test",
		Scopes: []string{ScopeRead},
	})
	if err != nil {
		t.Fatalf("CreateToken: %v", err)
	}

	// Update last used
	now := time.Now().Unix()
	err = repo.UpdateLastUsed(ctx, resp.Token.ID, now)
	if err != nil {
		t.Fatalf("UpdateLastUsed: %v", err)
	}

	// Verify via validate
	token, err := svc.ValidateToken(ctx, resp.Secret)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if token.LastUsedAt != now {
		t.Errorf("expected last_used_at %d, got %d", now, token.LastUsedAt)
	}
}
