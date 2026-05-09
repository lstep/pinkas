package mcptokens

// MCPToken represents a long-lived API token for MCP access.
type MCPToken struct {
	ID         string `json:"id"`
	UserID     string `json:"userId"`
	Name       string `json:"name"`
	TokenPrefix string `json:"tokenPrefix"`
	TokenHash  string `json:"-"` // never exposed
	Scopes     string `json:"scopes"` // JSON array
	SpaceID    string `json:"spaceId,omitempty"`
	LastUsedAt int64  `json:"lastUsedAt,omitempty"`
	CreatedAt  int64  `json:"createdAt"`
	ExpiresAt  int64  `json:"expiresAt,omitempty"`
}

// CreateTokenRequest is the body for POST /api/mcp-tokens.
type CreateTokenRequest struct {
	Name     string   `json:"name"`
	Scopes   []string `json:"scopes"`
	SpaceID  string   `json:"spaceId,omitempty"`
	ExpiresInDays int `json:"expiresInDays,omitempty"` // 0 = never expires
}

// CreateTokenResponse includes the full token secret (shown only once).
type CreateTokenResponse struct {
	Token   MCPToken `json:"token"`
	Secret  string   `json:"secret"` // the full mcp_xxx secret, shown once
}

// ListTokensResponse is returned from GET /api/mcp-tokens.
type ListTokensResponse struct {
	Tokens []MCPToken `json:"tokens"`
}

// Default scopes
const (
	ScopeRead  = "read"
	ScopeWrite = "write"
	ScopeAdmin = "admin"
)
