package auth

import (
	"time"
)

// UserInfo is stored in request context by the auth middleware.
type UserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

// contextKey is an unexported type to avoid collisions with third-party packages.
type contextKey struct{}

// UserContextKey is the key used to store UserInfo in context.
var UserContextKey = contextKey{}

// RegisterRequest is the body for POST /api/auth/register.
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

// LoginRequest is the body for POST /api/auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// TokenPair holds access and refresh tokens.
type TokenPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// LoginResponse is returned from POST /api/auth/login.
type LoginResponse struct {
	User  UserInfo  `json:"user"`
	Token TokenPair `json:"token"`
}

// RefreshRequest is the body for POST /api/auth/refresh.
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// MeResponse is returned from GET /api/auth/me.
type MeResponse struct {
	User UserInfo `json:"user"`
}

// jwtClaims defines the custom claims for our JWT tokens.
type jwtClaims struct {
	Email  string   `json:"email"`
	Name   string   `json:"name"`
	Role   string   `json:"role"`
	Scopes []string `json:"scopes,omitempty"`
}

// RefreshTokenMeta holds the plaintext token and its metadata.
type RefreshTokenMeta struct {
	ID        string
	Token     string
	TokenHash string
	UserID    string
	ExpiresAt time.Time
}

func (j jwtClaims) Valid() error {
	return nil
}
