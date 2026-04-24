package directories

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/mostdoc/mostdoc/internal/auth"
	"github.com/mostdoc/mostdoc/internal/httputil"
	sqlc "github.com/mostdoc/mostdoc/internal/db/query"
	"github.com/mostdoc/mostdoc/internal/sse"
)

// DirectoryResponse is the JSON representation of a directory.
type DirectoryResponse struct {
	ID        string  `json:"id"`
	SpaceID   string  `json:"spaceId,omitempty"`
	ParentID  *string `json:"parentId,omitempty"`
	Name      string  `json:"name"`
	Slug      string  `json:"slug"`
	Position  int64   `json:"position"`
	Icon      *string `json:"icon,omitempty"`
	CreatedBy string  `json:"createdBy,omitempty"`
	CreatedAt int64   `json:"createdAt,omitempty"`
	UpdatedAt int64   `json:"updatedAt,omitempty"`
}

// CreateDirectoryRequest is the body for POST /api/directories.
type CreateDirectoryRequest struct {
	SpaceID  string  `json:"spaceId"`
	ParentID *string `json:"parentId,omitempty"`
	Name     string  `json:"name"`
	Icon     string  `json:"icon,omitempty"`
}

// UpdateDirectoryRequest is the body for PATCH /api/directories/{id}.
type UpdateDirectoryRequest struct {
	Name     string  `json:"name,omitempty"`
	Icon     string  `json:"icon,omitempty"`
	Position *int64  `json:"position,omitempty"`
}

// MoveDirectoryRequest is the body for POST /api/directories/{id}/move.
type MoveDirectoryRequest struct {
	ParentID *string `json:"parentId"`
	Position *int64  `json:"position,omitempty"`
}

// RESTHandler holds HTTP handlers for directory REST endpoints.
type RESTHandler struct {
	repo   *Repository
	logger *slog.Logger
	sseHub *sse.Hub
}

// NewRESTHandler creates a new directories REST handler.
func NewRESTHandler(repo *Repository, logger *slog.Logger, sseHub *sse.Hub) *RESTHandler {
	return &RESTHandler{repo: repo, logger: logger, sseHub: sseHub}
}

// RegisterRESTRoutes registers directory REST routes on the mux.
func (h *RESTHandler) RegisterRESTRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/spaces/{spaceId}/directories", auth.RequireAuth(h.ListRoot))
	mux.HandleFunc("POST /api/directories", auth.RequireAuth(h.Create))
	mux.HandleFunc("GET /api/directories/{id}", auth.RequireAuth(h.Get))
	mux.HandleFunc("GET /api/spaces/{spaceId}/directories/{slug}", auth.RequireAuth(h.GetBySlug))
	mux.HandleFunc("PATCH /api/directories/{id}", auth.RequireAuth(h.Update))
	mux.HandleFunc("DELETE /api/directories/{id}", auth.RequireAuth(h.Delete))
	mux.HandleFunc("POST /api/directories/{id}/move", auth.RequireAuth(h.Move))
	mux.HandleFunc("GET /api/directories/{id}/children", auth.RequireAuth(h.Children))
	mux.HandleFunc("GET /api/directories/{id}/breadcrumb", auth.RequireAuth(h.Breadcrumb))
}

// ListRoot returns root directories for a space.
func (h *RESTHandler) ListRoot(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceId")
	if spaceID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Space ID is required")
		return
	}

	directories, err := h.repo.ListRootDirectories(r.Context(), spaceID)
	if err != nil {
		h.logger.Error("list root directories failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to list directories")
		return
	}

	result := make([]DirectoryResponse, 0, len(directories))
	for _, d := range directories {
		result = append(result, toDirectoryResponse(d))
	}

	httputil.JSON(w, http.StatusOK, map[string]interface{}{"directories": result})
}

// Create creates a new directory.
func (h *RESTHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateDirectoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Invalid request body")
		return
	}

	if req.Name == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Name is required")
		return
	}

	user, _ := auth.UserFromContext(r.Context())

	slug := slugify(req.Name)
	// Handle slug collisions
	existing, err := h.repo.ListRootDirectories(r.Context(), req.SpaceID)
	if err == nil && len(existing) > 0 {
		slug = resolveSlugCollision(slug, existing)
	}

	maxPos, _ := h.repo.GetMaxPosition(r.Context(), req.SpaceID, req.ParentID)
	position := maxPos + 1

	id := GenerateID()
	if err := h.repo.CreateDirectory(r.Context(), id, req.Name, slug, req.SpaceID, req.ParentID, position, req.Icon, user.ID); err != nil {
		h.logger.Error("create directory failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to create directory")
		return
	}

	dir, err := h.repo.GetDirectory(r.Context(), id)
	if err != nil {
		h.logger.Error("fetch created directory failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch created directory")
		return
	}

	httputil.JSON(w, http.StatusCreated, toDirectoryResponse(dir))

	// Broadcast SSE event
	if h.sseHub != nil {
		h.sseHub.Broadcast(sse.Event{
			Type: sse.EventDirectoryCreated,
			Data: toDirectoryResponse(dir),
		})
	}
}

// Get returns a single directory.
func (h *RESTHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Directory ID is required")
		return
	}

	dir, err := h.repo.GetDirectory(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "Directory not found")
		return
	}

	httputil.JSON(w, http.StatusOK, toDirectoryResponse(dir))
}

// GetBySlug returns a directory by slug within a space.
func (h *RESTHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceId")
	slug := r.PathValue("slug")
	if spaceID == "" || slug == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Space ID and slug are required")
		return
	}

	dir, err := h.repo.GetDirectoryBySlug(r.Context(), spaceID, slug)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "Directory not found")
		return
	}

	httputil.JSON(w, http.StatusOK, toDirectoryResponse(dir))
}

// Update modifies a directory.
func (h *RESTHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Directory ID is required")
		return
	}

	var req UpdateDirectoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Invalid request body")
		return
	}

	existing, err := h.repo.GetDirectory(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "Directory not found")
		return
	}

	name := existing.Name
	if req.Name != "" {
		name = req.Name
	}

	slug := existing.Slug
	if req.Name != "" {
		slug = slugify(req.Name)
		existingSlug := existing.Slug
		if slug != existingSlug {
			// Check for slug collisions in the same space
			var spaceID string
			if existing.SpaceID.Valid {
				spaceID = existing.SpaceID.String
			}
			rootDirs, _ := h.repo.ListRootDirectories(r.Context(), spaceID)
			var siblings []sqlc.Directory
			for _, d := range rootDirs {
				if d.ParentID.Valid == existing.ParentID.Valid {
					if (d.ParentID.Valid && existing.ParentID.Valid && d.ParentID.String == existing.ParentID.String) ||
						(!d.ParentID.Valid && !existing.ParentID.Valid) {
						siblings = append(siblings, d)
					}
				}
			}
			if len(siblings) > 0 {
				slug = resolveSlugCollision(slug, siblings)
			}
		}
	}

	icon := ""
	if existing.Icon.Valid {
		icon = existing.Icon.String
	}
	if req.Icon != "" {
		icon = req.Icon
	}

	if err := h.repo.UpdateDirectory(r.Context(), id, name, slug, icon); err != nil {
		h.logger.Error("update directory failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to update directory")
		return
	}

	if req.Position != nil {
		var parentID *string
		if existing.ParentID.Valid {
			parentID = &existing.ParentID.String
		}
		if err := h.repo.UpdateDirectoryPosition(r.Context(), id, *req.Position, parentID); err != nil {
			h.logger.Error("update directory position failed", "error", err)
		}
	}

	dir, err := h.repo.GetDirectory(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch updated directory")
		return
	}

	httputil.JSON(w, http.StatusOK, toDirectoryResponse(dir))

	// Broadcast SSE event
	if h.sseHub != nil {
		h.sseHub.Broadcast(sse.Event{
			Type: sse.EventDirectoryUpdated,
			Data: toDirectoryResponse(dir),
		})
	}
}

// Delete removes a directory.
func (h *RESTHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Directory ID is required")
		return
	}

	if err := h.repo.DeleteDirectory(r.Context(), id); err != nil {
		h.logger.Error("delete directory failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to delete directory")
		return
	}

	// Broadcast SSE event
	if h.sseHub != nil {
		h.sseHub.Broadcast(sse.Event{
			Type: sse.EventDirectoryDeleted,
			Data: map[string]string{"id": id},
		})
	}

	w.WriteHeader(http.StatusNoContent)
}

// Move moves a directory to a new parent.
func (h *RESTHandler) Move(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Directory ID is required")
		return
	}

	var req MoveDirectoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Invalid request body")
		return
	}

	// Circular move check: prevent moving a directory into itself or its descendants
	if req.ParentID != nil && *req.ParentID == id {
		httputil.WriteError(w, http.StatusBadRequest, "circular_move", "Cannot move a directory into itself")
		return
	}

	// Check if moving into a descendant
	if req.ParentID != nil && *req.ParentID != "" {
		ancestors, err := h.repo.GetAncestors(r.Context(), id)
		if err != nil {
			h.logger.Error("get ancestors failed", "error", err)
			httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to validate move")
			return
		}
		for _, ancestor := range ancestors {
			if ancestor.ID == *req.ParentID {
				httputil.WriteError(w, http.StatusBadRequest, "circular_move", "Cannot move a directory into its descendant")
				return
			}
		}
	}

	var position int64
	if req.Position != nil {
		position = *req.Position
	} else {
		var spaceID string
		existing, err := h.repo.GetDirectory(r.Context(), id)
		if err != nil {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "Directory not found")
			return
		}
		if existing.SpaceID.Valid {
			spaceID = existing.SpaceID.String
		}
		maxPos, _ := h.repo.GetMaxPosition(r.Context(), spaceID, req.ParentID)
		position = maxPos + 1
	}

	if err := h.repo.UpdateDirectoryPosition(r.Context(), id, position, req.ParentID); err != nil {
		h.logger.Error("move directory failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to move directory")
		return
	}

	// Broadcast SSE event
	if h.sseHub != nil {
		data := map[string]string{"id": id}
		if req.ParentID != nil {
			data["parentId"] = *req.ParentID
		}
		h.sseHub.Broadcast(sse.Event{
			Type: sse.EventDirectoryMoved,
			Data: data,
		})
	}

	w.WriteHeader(http.StatusNoContent)
}

// Children returns direct children of a directory.
func (h *RESTHandler) Children(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Directory ID is required")
		return
	}

	children, err := h.repo.ListChildren(r.Context(), id)
	if err != nil {
		h.logger.Error("list children failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to list children")
		return
	}

	result := make([]DirectoryResponse, 0, len(children))
	for _, d := range children {
		result = append(result, toDirectoryResponse(d))
	}

	httputil.JSON(w, http.StatusOK, map[string]interface{}{"directories": result})
}

// Breadcrumb returns the ancestor chain from root to parent.
func (h *RESTHandler) Breadcrumb(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Directory ID is required")
		return
	}

	ancestors, err := h.repo.GetAncestors(r.Context(), id)
	if err != nil {
		h.logger.Error("breadcrumb failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to load breadcrumb")
		return
	}

	result := make([]DirectoryResponse, 0, len(ancestors))
	for _, d := range ancestors {
		result = append(result, toDirectoryResponse(d))
	}

	httputil.JSON(w, http.StatusOK, map[string]interface{}{"ancestors": result})
}

func toDirectoryResponse(d sqlc.Directory) DirectoryResponse {
	var parentID *string
	if d.ParentID.Valid {
		parentID = &d.ParentID.String
	}
	var icon *string
	if d.Icon.Valid {
		icon = &d.Icon.String
	}
	var spaceID string
	if d.SpaceID.Valid {
		spaceID = d.SpaceID.String
	}
	var createdBy string
	if d.CreatedBy.Valid {
		createdBy = d.CreatedBy.String
	}
	var position int64
	if d.Position.Valid {
		position = d.Position.Int64
	}
	var createdAt, updatedAt int64
	if d.CreatedAt.Valid {
		createdAt = d.CreatedAt.Int64
	}
	if d.UpdatedAt.Valid {
		updatedAt = d.UpdatedAt.Int64
	}

	return DirectoryResponse{
		ID:        d.ID,
		SpaceID:   spaceID,
		ParentID:  parentID,
		Name:      d.Name,
		Slug:      d.Slug,
		Position:  position,
		Icon:      icon,
		CreatedBy: createdBy,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
}

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

func resolveSlugCollision(baseSlug string, existing []sqlc.Directory) string {
	maxSuffix := 0
	for _, d := range existing {
		if d.Slug == baseSlug {
			if maxSuffix < 1 {
				maxSuffix = 1
			}
			continue
		}
		if strings.HasPrefix(d.Slug, baseSlug+"-") {
			suffixStr := strings.TrimPrefix(d.Slug, baseSlug+"-")
			if suffix, err := strconv.Atoi(suffixStr); err == nil && suffix > maxSuffix {
				maxSuffix = suffix
			}
		}
	}
	if maxSuffix == 0 {
		return baseSlug
	}
	return baseSlug + "-" + strconv.Itoa(maxSuffix+1)
}


