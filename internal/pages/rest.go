package pages

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/mostdoc/mostdoc/internal/auth"
	"github.com/mostdoc/mostdoc/internal/httputil"
	"github.com/mostdoc/mostdoc/internal/permissions"
	sqlc "github.com/mostdoc/mostdoc/internal/db/query"
	"github.com/mostdoc/mostdoc/internal/sse"
)

// PageResponse is the JSON representation of a page.
type PageResponse struct {
	ID           string  `json:"id"`
	SpaceID      string  `json:"spaceId,omitempty"`
	DirectoryID  *string `json:"directoryId,omitempty"`
	DirectorySlug *string `json:"directorySlug,omitempty"`
	Title        string  `json:"title"`
	Slug         string  `json:"slug"`
	Position     int64   `json:"position"`
	Icon         *string `json:"icon,omitempty"`
	Permission   string  `json:"permission,omitempty"`
	CreatedBy    string  `json:"createdBy,omitempty"`
	CreatedAt    int64   `json:"createdAt,omitempty"`
	UpdatedAt    int64   `json:"updatedAt,omitempty"`
}

// CreatePageRequest is the body for POST /api/pages.
type CreatePageRequest struct {
	SpaceID     string  `json:"spaceId"`
	DirectoryID *string `json:"directoryId,omitempty"`
	Title       string  `json:"title"`
	Icon        string  `json:"icon,omitempty"`
}

// UpdatePageRequest is the body for PATCH /api/pages/{id}.
type UpdatePageRequest struct {
	Title    string `json:"title,omitempty"`
	Icon     string `json:"icon,omitempty"`
	Position *int64 `json:"position,omitempty"`
}

// MovePageRequest is the body for POST /api/pages/{id}/move.
type MovePageRequest struct {
	DirectoryID *string `json:"directoryId"`
	Position    *int64  `json:"position,omitempty"`
}

// RESTHandler holds HTTP handlers for page REST endpoints.
type RESTHandler struct {
	repo         *Repository
	logger       *slog.Logger
	sseHub       *sse.Hub
	permResolver *permissions.Resolver
}

// NewRESTHandler creates a new pages REST handler.
func NewRESTHandler(repo *Repository, logger *slog.Logger, sseHub *sse.Hub, permResolver *permissions.Resolver) *RESTHandler {
	return &RESTHandler{repo: repo, logger: logger, sseHub: sseHub, permResolver: permResolver}
}

// RegisterRESTRoutes registers page REST routes on the mux.
func (h *RESTHandler) RegisterRESTRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/spaces/{spaceId}/pages", auth.RequireAuth(h.ListRoot))
	mux.HandleFunc("POST /api/pages", auth.RequireAuth(h.Create))
	mux.HandleFunc("GET /api/pages/{id}", auth.RequireAuth(h.Get))
	mux.HandleFunc("GET /api/spaces/{spaceId}/pages/{slug}", auth.RequireAuth(h.GetBySlug))
	mux.HandleFunc("PATCH /api/pages/{id}", auth.RequireAuth(h.Update))
	mux.HandleFunc("DELETE /api/pages/{id}", auth.RequireAuth(h.Delete))
	mux.HandleFunc("POST /api/pages/{id}/move", auth.RequireAuth(h.Move))
	mux.HandleFunc("GET /api/directories/{id}/pages", auth.RequireAuth(h.ListByDirectory))
	mux.HandleFunc("GET /api/pages/{id}/breadcrumb", auth.RequireAuth(h.Breadcrumb))
}

// ListRoot returns root pages for a space (pages with no directory).
func (h *RESTHandler) ListRoot(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceId")
	if spaceID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Space ID is required")
		return
	}

	if !h.checkAccess(w, r, "space", spaceID, permissions.LevelViewer) {
		return
	}

	pages, err := h.repo.ListRootPages(r.Context(), spaceID)
	if err != nil {
		h.logger.Error("list root pages failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to list pages")
		return
	}

	result := make([]PageResponse, 0, len(pages))
	for _, p := range pages {
		result = append(result, toPageResponse(p))
	}

	// Filter by permission for non-admins
	user, _ := auth.UserFromContext(r.Context())
	if user.Role != "admin" {
		result = h.filterPages(r.Context(), user.ID, result)
	}

	httputil.JSON(w, http.StatusOK, map[string]interface{}{"pages": result})
}

// Create creates a new page.
func (h *RESTHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreatePageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Invalid request body")
		return
	}

	if req.Title == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Title is required")
		return
	}

	user, _ := auth.UserFromContext(r.Context())

	if !h.checkAccess(w, r, "space", req.SpaceID, permissions.LevelEditor) {
		return
	}

	slug := slugify(req.Title)
	// Handle slug collisions
	existing, err := h.repo.GetPagesBySlugPrefix(r.Context(), req.SpaceID, slug)
	if err == nil && len(existing) > 0 {
		slug = resolveSlugCollision(slug, existing)
	}

	maxPos, _ := h.repo.GetMaxPosition(r.Context(), req.DirectoryID)
	position := maxPos + 1

	id := GenerateID()
	if err := h.repo.CreatePage(r.Context(), id, req.SpaceID, req.Title, slug, position, req.DirectoryID, user.ID, req.Icon); err != nil {
		h.logger.Error("create page failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to create page")
		return
	}

	page, err := h.repo.GetPage(r.Context(), id)
	if err != nil {
		h.logger.Error("fetch created page failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch created page")
		return
	}

	httputil.JSON(w, http.StatusCreated, toPageResponse(page))

	// Broadcast SSE event
	if h.sseHub != nil {
		h.sseHub.Broadcast(sse.Event{
			Type: sse.EventPageCreated,
			Data: toPageResponse(page),
		})
	}
}

// Get returns a single page.
func (h *RESTHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Page ID is required")
		return
	}

	if !h.checkAccess(w, r, "page", id, permissions.LevelViewer) {
		return
	}

	page, err := h.repo.GetPage(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "Page not found")
		return
	}

	resp := toPageResponse(page)
	user, _ := auth.UserFromContext(r.Context())
	if user.Role == "admin" {
		resp.Permission = "admin"
	} else {
		resp.Permission = h.userPermission(r.Context(), user.ID, "page", id)
	}
	httputil.JSON(w, http.StatusOK, resp)
}

// GetBySlug returns a page by slug within a space.
func (h *RESTHandler) GetBySlug(w http.ResponseWriter, r *http.Request) {
	spaceID := r.PathValue("spaceId")
	slug := r.PathValue("slug")
	if spaceID == "" || slug == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Space ID and slug are required")
		return
	}

	if !h.checkAccess(w, r, "space", spaceID, permissions.LevelViewer) {
		return
	}

	page, err := h.repo.GetPageBySlug(r.Context(), spaceID, slug)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "Page not found")
		return
	}

	resp := toPageResponse(page)
	user, _ := auth.UserFromContext(r.Context())
	if user.Role == "admin" {
		resp.Permission = "admin"
	} else {
		resp.Permission = h.userPermission(r.Context(), user.ID, "page", page.ID)
	}
	httputil.JSON(w, http.StatusOK, resp)
}

// Update modifies a page.
func (h *RESTHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Page ID is required")
		return
	}

	if !h.checkAccess(w, r, "page", id, permissions.LevelEditor) {
		return
	}

	var req UpdatePageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Invalid request body")
		return
	}

	existing, err := h.repo.GetPage(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "Page not found")
		return
	}

	if req.Title != "" {
		slug := slugify(req.Title)
		existingSlug := ""
		if existing.Slug.Valid {
			existingSlug = existing.Slug.String
		}
		if slug != existingSlug {
			existingSlugs, _ := h.repo.GetPagesBySlugPrefix(r.Context(), existing.SpaceID.String, slug)
			if len(existingSlugs) > 0 {
				slug = resolveSlugCollision(slug, existingSlugs)
			}
		}
		if err := h.repo.UpdatePage(r.Context(), id, req.Title, slug); err != nil {
			h.logger.Error("update page title failed", "error", err)
			httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to update page")
			return
		}
	}

	if req.Icon != "" {
		if err := h.repo.UpdatePageIcon(r.Context(), id, req.Icon); err != nil {
			h.logger.Error("update page icon failed", "error", err)
		}
	}

	if req.Position != nil {
		var directoryID *string
		if existing.DirectoryID.Valid {
			directoryID = &existing.DirectoryID.String
		}
		if err := h.repo.UpdatePagePosition(r.Context(), id, *req.Position, directoryID); err != nil {
			h.logger.Error("update page position failed", "error", err)
		}
	}

	page, err := h.repo.GetPage(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to fetch updated page")
		return
	}

	httputil.JSON(w, http.StatusOK, toPageResponse(page))

	// Broadcast SSE event
	if h.sseHub != nil {
		h.sseHub.Broadcast(sse.Event{
			Type: sse.EventPageUpdated,
			Data: toPageResponse(page),
		})
	}
}

// Delete removes a page.
func (h *RESTHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Page ID is required")
		return
	}

	if !h.checkAccess(w, r, "page", id, permissions.LevelAdmin) {
		return
	}

	if err := h.repo.DeletePage(r.Context(), id); err != nil {
		h.logger.Error("delete page failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to delete page")
		return
	}

	// Broadcast SSE event
	if h.sseHub != nil {
		h.sseHub.Broadcast(sse.Event{
			Type: sse.EventPageDeleted,
			Data: map[string]string{"id": id},
		})
	}

	w.WriteHeader(http.StatusNoContent)
}

// Move moves a page to a new directory.
func (h *RESTHandler) Move(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Page ID is required")
		return
	}

	if !h.checkAccess(w, r, "page", id, permissions.LevelEditor) {
		return
	}

	var req MovePageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Invalid request body")
		return
	}

	var position int64
	if req.Position != nil {
		position = *req.Position
	} else {
		maxPos, _ := h.repo.GetMaxPosition(r.Context(), req.DirectoryID)
		position = maxPos + 1
	}

	if err := h.repo.UpdatePagePosition(r.Context(), id, position, req.DirectoryID); err != nil {
		h.logger.Error("move page failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to move page")
		return
	}

	// Broadcast SSE event
	if h.sseHub != nil {
		data := map[string]string{"id": id}
		if req.DirectoryID != nil {
			data["directoryId"] = *req.DirectoryID
		}
		h.sseHub.Broadcast(sse.Event{
			Type: sse.EventPageMoved,
			Data: data,
		})
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListByDirectory returns pages inside a directory.
func (h *RESTHandler) ListByDirectory(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Directory ID is required")
		return
	}

	if !h.checkAccess(w, r, "directory", id, permissions.LevelViewer) {
		return
	}

	pages, err := h.repo.ListPagesByDirectory(r.Context(), id)
	if err != nil {
		h.logger.Error("list pages by directory failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to list pages")
		return
	}

	result := make([]PageResponse, 0, len(pages))
	for _, p := range pages {
		result = append(result, toPageResponse(p))
	}

	// Filter by permission for non-admins
	user, _ := auth.UserFromContext(r.Context())
	if user.Role != "admin" {
		result = h.filterPages(r.Context(), user.ID, result)
	}

	httputil.JSON(w, http.StatusOK, map[string]interface{}{"pages": result})
}

// Breadcrumb returns the ancestor directories from root to the page's directory.
func (h *RESTHandler) Breadcrumb(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Page ID is required")
		return
	}

	if !h.checkAccess(w, r, "page", id, permissions.LevelViewer) {
		return
	}

	ancestors, err := h.repo.GetAncestors(r.Context(), id)
	if err != nil {
		h.logger.Error("breadcrumb failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to load breadcrumb")
		return
	}

	// Convert directories to a simplified response
	result := make([]map[string]interface{}, 0, len(ancestors))
	for _, d := range ancestors {
		item := map[string]interface{}{
			"id":   d.ID,
			"name": d.Name,
			"slug": d.Slug,
		}
		if d.Icon.Valid {
			item["icon"] = d.Icon.String
		}
		result = append(result, item)
	}

	httputil.JSON(w, http.StatusOK, map[string]interface{}{"ancestors": result})
}

// filterPages filters out pages the user doesn't have viewer+ access to.
func (h *RESTHandler) filterPages(ctx context.Context, userID string, pages []PageResponse) []PageResponse {
	filtered := make([]PageResponse, 0, len(pages))
	for _, p := range pages {
		level := h.permResolver.Resolve(ctx, userID, "page", p.ID)
		if level >= permissions.LevelViewer {
			p.Permission = levelToName(level)
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// userPermission returns the user's permission on a target as a string.
func (h *RESTHandler) userPermission(ctx context.Context, userID, targetType, targetID string) string {
	level := h.permResolver.Resolve(ctx, userID, targetType, targetID)
	return levelToName(level)
}

func levelToName(level int) string {
	switch {
	case level >= permissions.LevelAdmin:
		return "admin"
	case level >= permissions.LevelEditor:
		return "editor"
	case level >= permissions.LevelViewer:
		return "viewer"
	default:
		return "none"
	}
}

// checkAccess verifies the user has at least minLevel on the target. Returns false and writes an error response if denied.
func (h *RESTHandler) checkAccess(w http.ResponseWriter, r *http.Request, targetType, targetID string, minLevel int) bool {
	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
		return false
	}
	if user.Role == "admin" {
		return true
	}
	level := h.permResolver.Resolve(r.Context(), user.ID, targetType, targetID)
	if level < minLevel {
		httputil.WriteError(w, http.StatusForbidden, "forbidden", "Insufficient permissions")
		return false
	}
	return true
}

func toPageResponse(p sqlc.Page) PageResponse {
	var directoryID *string
	if p.DirectoryID.Valid {
		directoryID = &p.DirectoryID.String
	}
	var icon *string
	if p.Icon.Valid {
		icon = &p.Icon.String
	}
	var spaceID string
	if p.SpaceID.Valid {
		spaceID = p.SpaceID.String
	}
	var title string
	if p.Title.Valid {
		title = p.Title.String
	}
	var slug string
	if p.Slug.Valid {
		slug = p.Slug.String
	}
	var createdBy string
	if p.CreatedBy.Valid {
		createdBy = p.CreatedBy.String
	}
	var position int64
	if p.Position.Valid {
		position = p.Position.Int64
	}
	var createdAt, updatedAt int64
	if p.CreatedAt.Valid {
		createdAt = p.CreatedAt.Int64
	}
	if p.UpdatedAt.Valid {
		updatedAt = p.UpdatedAt.Int64
	}

	return PageResponse{
		ID:          p.ID,
		SpaceID:     spaceID,
		DirectoryID: directoryID,
		Title:       title,
		Slug:        slug,
		Position:    position,
		Icon:        icon,
		CreatedBy:   createdBy,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
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

func resolveSlugCollision(baseSlug string, existing []sqlc.Page) string {
	maxSuffix := 0
	for _, p := range existing {
		if !p.Slug.Valid {
			continue
		}
		s := p.Slug.String
		if s == baseSlug {
			if maxSuffix < 1 {
				maxSuffix = 1
			}
			continue
		}
		if strings.HasPrefix(s, baseSlug+"-") {
			suffixStr := strings.TrimPrefix(s, baseSlug+"-")
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

// GenerateID returns a new UUID.
func GenerateID() string {
	return uuid.NewString()
}
