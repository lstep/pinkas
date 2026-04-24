package auth

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	sqlc "github.com/mostdoc/mostdoc/internal/db/query"
)

// Repository wraps sqlc queries for auth operations.
type Repository struct {
	queries *sqlc.Queries
}

// NewRepository creates a new auth repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{queries: sqlc.New(db)}
}

// WithTx returns a repository that executes within the given transaction.
func (r *Repository) WithTx(tx *sql.Tx) *Repository {
	return &Repository{queries: r.queries.WithTx(tx)}
}

// CreateUser inserts a new user.
func (r *Repository) CreateUser(ctx context.Context, id, email, name, passwordHash, role string) error {
	return r.queries.CreateUser(ctx, sqlc.CreateUserParams{
		ID:           id,
		Email:        email,
		Name:         sql.NullString{String: name, Valid: name != ""},
		PasswordHash: passwordHash,
		GlobalRole:   sql.NullString{String: role, Valid: role != ""},
	})
}

// GetUserByEmail fetches a user by email address.
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (sqlc.User, error) {
	return r.queries.GetUserByEmail(ctx, email)
}

// GetUserByID fetches a user by ID.
func (r *Repository) GetUserByID(ctx context.Context, id string) (sqlc.User, error) {
	return r.queries.GetUserByID(ctx, id)
}

// CountUsers returns the total number of users.
func (r *Repository) CountUsers(ctx context.Context) (int64, error) {
	return r.queries.CountUsers(ctx)
}

// CreateRefreshToken stores a hashed refresh token.
func (r *Repository) CreateRefreshToken(ctx context.Context, id, userID, tokenHash string, expiresAt time.Time) error {
	return r.queries.CreateRefreshToken(ctx, sqlc.CreateRefreshTokenParams{
		ID:        id,
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: expiresAt.Unix(),
	})
}

// GetRefreshToken fetches a refresh token by ID.
func (r *Repository) GetRefreshToken(ctx context.Context, id string) (sqlc.RefreshToken, error) {
	return r.queries.GetRefreshToken(ctx, id)
}

// DeleteRefreshToken removes a refresh token.
func (r *Repository) DeleteRefreshToken(ctx context.Context, id string) error {
	return r.queries.DeleteRefreshToken(ctx, id)
}

// DeleteExpiredRefreshTokens removes all expired tokens.
func (r *Repository) DeleteExpiredRefreshTokens(ctx context.Context) error {
	return r.queries.DeleteExpiredRefreshTokens(ctx)
}

// GetSetting fetches a setting value by key.
func (r *Repository) GetSetting(ctx context.Context, key string) (string, error) {
	s, err := r.queries.GetSetting(ctx, key)
	if err != nil {
		return "", err
	}
	return s.Value, nil
}

// UpsertSetting sets a key-value pair.
func (r *Repository) UpsertSetting(ctx context.Context, key, value string) error {
	return r.queries.UpsertSetting(ctx, sqlc.UpsertSettingParams{
		Key:   key,
		Value: value,
	})
}

// GenerateID returns a new UUID string.
func GenerateID() string {
	return uuid.NewString()
}

// Now returns the current UTC time.
func Now() time.Time {
	return time.Now().UTC()
}

// ScanUser converts a sqlc User row to a domain UserInfo.
func ScanUser(u sqlc.User) UserInfo {
	name := ""
	if u.Name.Valid {
		name = u.Name.String
	}
	role := ""
	if u.GlobalRole.Valid {
		role = u.GlobalRole.String
	}
	return UserInfo{
		ID:    u.ID,
		Email: u.Email,
		Name:  name,
		Role:  role,
	}
}

// IsNotFound returns true if the error is sql.ErrNoRows.
func IsNotFound(err error) bool {
	return err == sql.ErrNoRows
}
