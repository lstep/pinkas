package mcptokens

import (
	"context"
	"database/sql"
	"fmt"
)

// Repository handles database operations for MCP tokens.
type Repository struct {
	conn *sql.DB
}

// NewRepository creates a new MCP token repository.
func NewRepository(conn *sql.DB) *Repository {
	return &Repository{conn: conn}
}

// CreateToken stores a new MCP token.
func (r *Repository) CreateToken(ctx context.Context, t MCPToken) error {
	var expiresAt, lastUsedAt interface{}
	if t.ExpiresAt > 0 {
		expiresAt = t.ExpiresAt
	}
	if t.LastUsedAt > 0 {
		lastUsedAt = t.LastUsedAt
	}

	_, err := r.conn.ExecContext(ctx, `
		INSERT INTO mcp_tokens (id, user_id, name, token_prefix, token_hash, scopes, space_id, last_used_at, created_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		t.ID, t.UserID, t.Name, t.TokenPrefix, t.TokenHash, t.Scopes,
		nullIfEmpty(t.SpaceID), lastUsedAt, t.CreatedAt, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("insert mcp token: %w", err)
	}
	return nil
}

// GetTokenByID fetches a token by its ID.
func (r *Repository) GetTokenByID(ctx context.Context, id string) (MCPToken, error) {
	var t MCPToken
	var spaceID, lastUsedAt, expiresAt sql.NullString

	err := r.conn.QueryRowContext(ctx, `
		SELECT id, user_id, name, token_prefix, token_hash, scopes, space_id, last_used_at, created_at, expires_at
		FROM mcp_tokens WHERE id = ?
	`, id).Scan(
		&t.ID, &t.UserID, &t.Name, &t.TokenPrefix, &t.TokenHash, &t.Scopes,
		&spaceID, &lastUsedAt, &t.CreatedAt, &expiresAt,
	)
	if err == sql.ErrNoRows {
		return MCPToken{}, fmt.Errorf("token not found")
	}
	if err != nil {
		return MCPToken{}, fmt.Errorf("get mcp token: %w", err)
	}

	t.SpaceID = spaceID.String
	if v := lastUsedAt.String; v != "" {
		t.LastUsedAt = parseInt64(v)
	}
	if v := expiresAt.String; v != "" {
		t.ExpiresAt = parseInt64(v)
	}

	return t, nil
}

// ListTokens returns all tokens owned by a user.
func (r *Repository) ListTokens(ctx context.Context, userID string) ([]MCPToken, error) {
	rows, err := r.conn.QueryContext(ctx, `
		SELECT id, user_id, name, token_prefix, token_hash, scopes, space_id, last_used_at, created_at, expires_at
		FROM mcp_tokens WHERE user_id = ?
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list mcp tokens: %w", err)
	}
	defer rows.Close()

	var tokens []MCPToken
	for rows.Next() {
		var t MCPToken
		var spaceID, lastUsedAt, expiresAt sql.NullString

		if err := rows.Scan(
			&t.ID, &t.UserID, &t.Name, &t.TokenPrefix, &t.TokenHash, &t.Scopes,
			&spaceID, &lastUsedAt, &t.CreatedAt, &expiresAt,
		); err != nil {
			return nil, fmt.Errorf("scan mcp token: %w", err)
		}

		t.SpaceID = spaceID.String
		if v := lastUsedAt.String; v != "" {
			t.LastUsedAt = parseInt64(v)
		}
		if v := expiresAt.String; v != "" {
			t.ExpiresAt = parseInt64(v)
		}

		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

// DeleteToken removes a token (scoped to user).
func (r *Repository) DeleteToken(ctx context.Context, id, userID string) error {
	res, err := r.conn.ExecContext(ctx, "DELETE FROM mcp_tokens WHERE id = ? AND user_id = ?", id, userID)
	if err != nil {
		return fmt.Errorf("delete mcp token: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("token not found")
	}
	return nil
}

// UpdateLastUsed updates the last_used_at timestamp for a token.
func (r *Repository) UpdateLastUsed(ctx context.Context, id string, ts int64) error {
	_, err := r.conn.ExecContext(ctx, "UPDATE mcp_tokens SET last_used_at = ? WHERE id = ?", ts, id)
	return err
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func parseInt64(s string) int64 {
	var v int64
	fmt.Sscanf(s, "%d", &v)
	return v
}
