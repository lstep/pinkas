package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/server"
	"github.com/mostdoc/mostdoc/internal/directories"
	"github.com/mostdoc/mostdoc/internal/mcptokens"
	"github.com/mostdoc/mostdoc/internal/pages"
	"github.com/mostdoc/mostdoc/internal/permissions"
	"github.com/mostdoc/mostdoc/internal/spaces"
)

// contextKey is used for storing auth info in request context.
type contextKey string

const (
	ctxKeyUserID    contextKey = "user_id"
	ctxKeyToken     contextKey = "mcp_token"
	ctxKeyIsAdmin   contextKey = "is_admin"
)

// Server wraps the MCP server with HTTP auth middleware.
type Server struct {
	mcpServer  *server.MCPServer
	sseServer  *server.SSEServer
	mcpTokenSvc *mcptokens.Service
	pagesRepo  *pages.Repository
	spacesRepo *spaces.Repository
	dirsRepo   *directories.Repository
	permRes    *permissions.Resolver
	logger     *slog.Logger
	writeLock  *SpaceWriteLock
}

// NewServer creates a new MCP server with all tool handlers wired.
func NewServer(
	mcpTokenSvc *mcptokens.Service,
	pagesRepo *pages.Repository,
	spacesRepo *spaces.Repository,
	dirsRepo *directories.Repository,
	permRes *permissions.Resolver,
	logger *slog.Logger,
) *Server {
	ms := server.NewMCPServer(
		"MostDoc Wiki",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithLogging(),
		server.WithRecovery(),
	)

	s := &Server{
		mcpServer:   ms,
		mcpTokenSvc: mcpTokenSvc,
		pagesRepo:   pagesRepo,
		spacesRepo:  spacesRepo,
		dirsRepo:    dirsRepo,
		permRes:     permRes,
		logger:      logger,
		writeLock:   NewSpaceWriteLock(30 * time.Second),
	}

	s.registerTools()
	s.sseServer = server.NewSSEServer(ms)

	return s
}

// Handler returns an http.Handler that wraps the SSE server with auth middleware.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /sse", s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		s.sseServer.SSEHandler().ServeHTTP(w, r)
	}))

	mux.HandleFunc("POST /message", s.authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		s.sseServer.MessageHandler().ServeHTTP(w, r)
	}))

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	return mux
}

// authMiddleware extracts and validates the MCP token from the Authorization header.
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tokenStr := extractToken(r)
		if tokenStr == "" {
			if r.Method == "POST" {
				http.Error(w, `{"error":"missing authorization"}`, http.StatusUnauthorized)
				return
			}
			// SSE connection without token — allow anon (limited read-only)
			ctx := context.WithValue(r.Context(), ctxKeyUserID, "")
			next(w, r.WithContext(ctx))
			return
		}

		token, err := s.mcpTokenSvc.ValidateToken(r.Context(), tokenStr)
		if err != nil {
			s.logger.Warn("mcp token validation failed", "error", err)
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}

		// Check if token is expired (ExpiresAt is a Unix timestamp; 0 means no expiry)
		if token.ExpiresAt > 0 && time.Now().Unix() > token.ExpiresAt {
			http.Error(w, `{"error":"token expired"}`, http.StatusUnauthorized)
			return
		}

		ctx := r.Context()
		ctx = context.WithValue(ctx, ctxKeyUserID, token.UserID)
		ctx = context.WithValue(ctx, ctxKeyToken, token)
		ctx = context.WithValue(ctx, ctxKeyIsAdmin, hasScope(token, mcptokens.ScopeAdmin))

		next(w, r.WithContext(ctx))
	}
}

// extractToken pulls the Bearer token from the Authorization header or query param.
func extractToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return r.URL.Query().Get("token")
}

// hasScope checks if a token has the given scope (Scopes is a JSON array string).
func hasScope(token *mcptokens.MCPToken, scope string) bool {
	var scopes []string
	if err := json.Unmarshal([]byte(token.Scopes), &scopes); err != nil {
		return false
	}
	for _, s := range scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// mcpUserFromContext extracts user info set by auth middleware.
func mcpUserFromContext(ctx context.Context) (userID string, token *mcptokens.MCPToken, isAdmin bool) {
	userID, _ = ctx.Value(ctxKeyUserID).(string)
	token, _ = ctx.Value(ctxKeyToken).(*mcptokens.MCPToken)
	isAdmin, _ = ctx.Value(ctxKeyIsAdmin).(bool)
	return
}

// requireWrite checks if the token has write or admin scope.
func requireWrite(token *mcptokens.MCPToken) error {
	if token == nil || (!hasScope(token, mcptokens.ScopeWrite) && !hasScope(token, mcptokens.ScopeAdmin)) {
		return fmt.Errorf("token requires 'write' or 'admin' scope")
	}
	return nil
}
