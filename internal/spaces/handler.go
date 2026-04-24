package spaces

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/mostdoc/mostdoc/internal/auth"
	"github.com/mostdoc/mostdoc/internal/httputil"
)

// Handler holds HTTP handlers for space endpoints.
type Handler struct {
	repo   *Repository
	logger *slog.Logger
}

// NewHandler creates a new spaces handler.
func NewHandler(repo *Repository, logger *slog.Logger) *Handler {
	return &Handler{repo: repo, logger: logger}
}

// RegisterRoutes registers space routes on the mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/spaces", auth.RequireAuth(h.List))
	mux.HandleFunc("POST /api/spaces", auth.RequireAuth(h.Create))
	mux.HandleFunc("GET /api/spaces/{id}", auth.RequireAuth(h.Get))
	mux.HandleFunc("PATCH /api/spaces/{id}", auth.RequireAuth(h.Update))
	mux.HandleFunc("DELETE /api/spaces/{id}", auth.RequireAuth(h.Delete))
}

// List returns all spaces.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	spaces, err := h.repo.List(r.Context())
	if err != nil {
		h.logger.Error("list spaces failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to list spaces")
		return
	}

	result := make([]Space, 0, len(spaces))
	for _, s := range spaces {
		result = append(result, ScanSpace(s))
	}

	httputil.JSON(w, http.StatusOK, ListResponse{Spaces: result})
}

// Create creates a new space.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Invalid request body")
		return
	}

	if req.Name == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Name is required")
		return
	}

	slug := slugify(req.Name)
	defaultPerm := req.DefaultPermission
	if defaultPerm == "" {
		defaultPerm = "none"
	}

	id := GenerateID()
	if err := h.repo.Create(r.Context(), id, req.Name, slug, defaultPerm, req.McpWriteEnabled, req.SnapshotRetentionDays); err != nil {
		h.logger.Error("create space failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to create space")
		return
	}

	space, err := h.repo.Get(r.Context(), id)
	if err != nil {
		h.logger.Error("fetch created space failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch created space")
		return
	}

	httputil.JSON(w, http.StatusCreated, ScanSpace(space))
}

// Get returns a single space.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Space ID is required")
		return
	}

	space, err := h.repo.Get(r.Context(), id)
	if err != nil {
		h.logger.Error("get space failed", "error", err)
		httputil.WriteError(w, http.StatusNotFound, "not_found", "Space not found")
		return
	}

	httputil.JSON(w, http.StatusOK, ScanSpace(space))
}

// Update modifies a space.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Space ID is required")
		return
	}

	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Invalid request body")
		return
	}

	// Fetch existing
	existing, err := h.repo.Get(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "Space not found")
		return
	}

	name := existing.Name
	if req.Name != "" {
		name = req.Name
	}

	defaultPerm := "none"
	if existing.DefaultPermission.Valid {
		defaultPerm = existing.DefaultPermission.String
	}
	if req.DefaultPermission != "" {
		defaultPerm = req.DefaultPermission
	}

	mcpEnabled := existing.McpWriteEnabled.Valid && existing.McpWriteEnabled.Int64 == 1
	if req.McpWriteEnabled != nil {
		mcpEnabled = *req.McpWriteEnabled
	}

	var retentionDays *int64
	if existing.SnapshotRetentionDays.Valid {
		v := existing.SnapshotRetentionDays.Int64
		retentionDays = &v
	}
	if req.SnapshotRetentionDays != nil {
		retentionDays = req.SnapshotRetentionDays
	}

	if err := h.repo.Update(r.Context(), id, name, defaultPerm, mcpEnabled, retentionDays); err != nil {
		h.logger.Error("update space failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to update space")
		return
	}

	space, err := h.repo.Get(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch updated space")
		return
	}

	httputil.JSON(w, http.StatusOK, ScanSpace(space))
}

// Delete removes a space.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Space ID is required")
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		h.logger.Error("delete space failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to delete space")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// slugify converts a title to a URL-friendly slug.
func slugify(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			if b.Len() > 0 && b.String()[b.Len()-1] != '-' {
				b.WriteRune('-')
			}
		}
	}
	result := strings.Trim(b.String(), "-")
	if result == "" {
		result = "untitled"
	}
	return result
}
