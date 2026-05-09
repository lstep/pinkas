package mcptokens

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/mostdoc/mostdoc/internal/auth"
	"github.com/mostdoc/mostdoc/internal/httputil"
)

// Handler holds HTTP handlers for MCP token management.
type Handler struct {
	service *Service
	logger  *slog.Logger
}

// NewHandler creates a new MCP token handler.
func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

// RegisterRoutes registers MCP token routes on the mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/mcp-tokens", auth.RequireAuth(h.CreateToken))
	mux.HandleFunc("GET /api/mcp-tokens", auth.RequireAuth(h.ListTokens))
	mux.HandleFunc("DELETE /api/mcp-tokens/{id}", auth.RequireAuth(h.DeleteToken))
}

// CreateToken generates a new MCP API token.
func (h *Handler) CreateToken(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return
	}

	// Admin only for now
	if user.Role != "admin" {
		httputil.WriteError(w, http.StatusForbidden, "forbidden", "Admin access required")
		return
	}

	var req CreateTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Invalid request body")
		return
	}

	if req.Name == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Token name is required")
		return
	}

	// Validate scopes
	if len(req.Scopes) > 0 {
		validScopes := map[string]bool{ScopeRead: true, ScopeWrite: true, ScopeAdmin: true}
		for _, s := range req.Scopes {
			if !validScopes[s] {
				httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Invalid scope: "+s)
				return
			}
		}
	}

	resp, err := h.service.CreateToken(r.Context(), user.ID, req)
	if err != nil {
		h.logger.Error("create mcp token failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to create token")
		return
	}

	httputil.JSON(w, http.StatusCreated, resp)
}

// ListTokens returns all MCP tokens for the authenticated user.
func (h *Handler) ListTokens(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return
	}

	tokens, err := h.service.ListTokens(r.Context(), user.ID)
	if err != nil {
		h.logger.Error("list mcp tokens failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to list tokens")
		return
	}

	// Strip token_hash from response
	type tokenResponse struct {
		ID          string `json:"id"`
		UserID      string `json:"userId"`
		Name        string `json:"name"`
		TokenPrefix string `json:"tokenPrefix"`
		Scopes      string `json:"scopes"`
		SpaceID     string `json:"spaceId,omitempty"`
		LastUsedAt  int64  `json:"lastUsedAt,omitempty"`
		CreatedAt   int64  `json:"createdAt"`
		ExpiresAt   int64  `json:"expiresAt,omitempty"`
	}

	httputil.JSON(w, http.StatusOK, ListTokensResponse{Tokens: tokens})
}

// DeleteToken revokes an MCP token.
func (h *Handler) DeleteToken(w http.ResponseWriter, r *http.Request) {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Token ID is required")
		return
	}

	if err := h.service.DeleteToken(r.Context(), id, user.ID); err != nil {
		h.logger.Error("delete mcp token failed", "error", err)
		httputil.WriteError(w, http.StatusNotFound, "not_found", "Token not found")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
