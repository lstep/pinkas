package mcptokens

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	tokenPrefix   = "mcp_"
	idLength      = 12
	secretLength  = 32 // bytes before base64
	bcryptCost    = 12
)

// Service handles MCP token business logic.
type Service struct {
	repo *Repository
}

// NewService creates a new MCP token service.
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// CreateToken generates a new MCP API token.
// Returns the full token string (shown once) and the stored record.
func (s *Service) CreateToken(ctx context.Context, userID string, req CreateTokenRequest) (*CreateTokenResponse, error) {
	// Generate token ID
	id, err := generateTokenID()
	if err != nil {
		return nil, fmt.Errorf("generate token id: %w", err)
	}

	// Generate secret
	secretBytes := make([]byte, secretLength)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, fmt.Errorf("generate secret: %w", err)
	}
	secret := base64URLEncode(secretBytes)

	// Full token string: mcp_{id}_{secret}
	fullToken := tokenPrefix + id + "_" + secret

	// Hash the secret for storage
	hash, err := bcrypt.GenerateFromPassword([]byte(secret), bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash secret: %w", err)
	}

	// Calculate expiresAt
	var expiresAt int64
	if req.ExpiresInDays > 0 {
		expiresAt = time.Now().Add(time.Duration(req.ExpiresInDays) * 24 * time.Hour).Unix()
	}

	// Default scopes
	if len(req.Scopes) == 0 {
		req.Scopes = []string{ScopeRead}
	}

	// Marshal scopes to JSON
	scopesJSON, err := json.Marshal(req.Scopes)
	if err != nil {
		return nil, fmt.Errorf("marshal scopes: %w", err)
	}

	now := time.Now().Unix()

	token := MCPToken{
		ID:          id,
		UserID:      userID,
		Name:        req.Name,
		TokenPrefix: tokenPrefix + id[:6],
		TokenHash:   string(hash),
		Scopes:      string(scopesJSON),
		SpaceID:     req.SpaceID,
		CreatedAt:   now,
		ExpiresAt:   expiresAt,
	}

	if err := s.repo.CreateToken(ctx, token); err != nil {
		return nil, fmt.Errorf("store token: %w", err)
	}

	return &CreateTokenResponse{
		Token:  token,
		Secret: fullToken,
	}, nil
}

// ValidateToken checks if an MCP token string is valid and returns the token record.
func (s *Service) ValidateToken(ctx context.Context, tokenStr string) (*MCPToken, error) {
	// Parse the token format: mcp_{id}_{secret}
	if len(tokenStr) < len(tokenPrefix)+1 {
		return nil, fmt.Errorf("invalid token format")
	}

	// Extract the ID portion (first idLength chars after prefix)
	fullToken := tokenStr[len(tokenPrefix):]
	if len(fullToken) < idLength+1 || fullToken[idLength] != '_' {
		return nil, fmt.Errorf("invalid token format")
	}
	id := fullToken[:idLength]
	secret := fullToken[idLength+1:]

	if id == "" || secret == "" {
		return nil, fmt.Errorf("invalid token format")
	}

	// Look up by ID
	token, err := s.repo.GetTokenByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("token not found: %w", err)
	}

	// Check expiry
	if token.ExpiresAt > 0 && time.Now().Unix() > token.ExpiresAt {
		return nil, fmt.Errorf("token expired")
	}

	// Verify hash
	if err := bcrypt.CompareHashAndPassword([]byte(token.TokenHash), []byte(secret)); err != nil {
		return nil, fmt.Errorf("invalid token")
	}

	// Update last_used_at asynchronously (fire and forget)
	go func() {
		_ = s.repo.UpdateLastUsed(context.Background(), token.ID, time.Now().Unix())
	}()

	return &token, nil
}

// ListTokens returns all tokens for a user.
func (s *Service) ListTokens(ctx context.Context, userID string) ([]MCPToken, error) {
	return s.repo.ListTokens(ctx, userID)
}

// DeleteToken revokes a token.
func (s *Service) DeleteToken(ctx context.Context, id, userID string) error {
	return s.repo.DeleteToken(ctx, id, userID)
}

// HasScope checks if a token has the given scope.
func HasScope(token *MCPToken, scope string) bool {
	var scopes []string
	if err := json.Unmarshal([]byte(token.Scopes), &scopes); err != nil {
		return false
	}
	for _, s := range scopes {
		if s == scope || s == ScopeAdmin {
			return true
		}
	}
	return false
}

// HasAccessToSpace checks if a token is allowed to access a space.
func HasAccessToSpace(token *MCPToken, spaceID string) bool {
	if token.SpaceID == "" {
		return true // no restriction
	}
	return token.SpaceID == spaceID
}

// generateTokenID generates a random alphanumeric ID.
func generateTokenID() (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, idLength)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		b[i] = charset[n.Int64()]
	}
	return string(b), nil
}

func base64URLEncode(data []byte) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	srcLen := len(data)
	b := make([]byte, (srcLen+2)/3*4)
	for i, j := 0, 0; i < srcLen; i += 3 {
		val := uint(data[i]) << 16
		if i+1 < srcLen {
			val |= uint(data[i+1]) << 8
		}
		if i+2 < srcLen {
			val |= uint(data[i+2])
		}
		b[j] = alphabet[(val>>18)&0x3F]
		b[j+1] = alphabet[(val>>12)&0x3F]
		b[j+2] = alphabet[(val>>6)&0x3F]
		b[j+3] = alphabet[val&0x3F]
		j += 4
	}
	return string(b)
}
