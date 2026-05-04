package groups

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/mostdoc/mostdoc/internal/auth"
	"github.com/mostdoc/mostdoc/internal/httputil"
)

// Handler holds HTTP handlers for group endpoints.
type Handler struct {
	repo   *Repository
	logger *slog.Logger
}

// NewHandler creates a new groups handler.
func NewHandler(repo *Repository, logger *slog.Logger) *Handler {
	return &Handler{repo: repo, logger: logger}
}

// RegisterRoutes registers group routes on the mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// All group management requires admin global role
	mux.HandleFunc("GET /api/groups", auth.RequireAuth(h.List))
	mux.HandleFunc("POST /api/groups", auth.RequireAuth(h.Create))
	mux.HandleFunc("GET /api/groups/{id}", auth.RequireAuth(h.Get))
	mux.HandleFunc("PATCH /api/groups/{id}", auth.RequireAuth(h.Update))
	mux.HandleFunc("DELETE /api/groups/{id}", auth.RequireAuth(h.Delete))
	mux.HandleFunc("GET /api/groups/{id}/members", auth.RequireAuth(h.ListMembers))
	mux.HandleFunc("POST /api/groups/{id}/members", auth.RequireAuth(h.AddMember))
	mux.HandleFunc("DELETE /api/groups/{id}/members/{userId}", auth.RequireAuth(h.RemoveMember))
}

// GroupResponse is the JSON response for a group.
type GroupResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// CreateGroupRequest is the body for POST /api/groups.
type CreateGroupRequest struct {
	Name string `json:"name"`
}

// UpdateGroupRequest is the body for PATCH /api/groups/{id}.
type UpdateGroupRequest struct {
	Name string `json:"name"`
}

// AddMemberRequest is the body for POST /api/groups/{id}/members.
type AddMemberRequest struct {
	UserID string `json:"userId"`
}

// List returns all groups (admin only).
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	if user.Role != "admin" {
		httputil.WriteError(w, http.StatusForbidden, "forbidden", "Admin access required")
		return
	}

	groups, err := h.repo.ListGroups(r.Context())
	if err != nil {
		h.logger.Error("list groups failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to list groups")
		return
	}

	result := make([]GroupResponse, 0, len(groups))
	for _, g := range groups {
		result = append(result, GroupResponse{ID: g.ID, Name: g.Name})
	}

	httputil.JSON(w, http.StatusOK, map[string]interface{}{"groups": result})
}

// Create creates a new group (admin only).
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	if user.Role != "admin" {
		httputil.WriteError(w, http.StatusForbidden, "forbidden", "Admin access required")
		return
	}

	var req CreateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Invalid request body")
		return
	}

	if req.Name == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Name is required")
		return
	}

	id := uuid.New().String()
	if err := h.repo.CreateGroup(r.Context(), id, req.Name); err != nil {
		h.logger.Error("create group failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to create group")
		return
	}

	httputil.JSON(w, http.StatusCreated, GroupResponse{ID: id, Name: req.Name})
}

// Get returns a single group.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	if user.Role != "admin" {
		httputil.WriteError(w, http.StatusForbidden, "forbidden", "Admin access required")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Group ID is required")
		return
	}

	group, err := h.repo.GetGroup(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "Group not found")
		return
	}

	httputil.JSON(w, http.StatusOK, GroupResponse{ID: group.ID, Name: group.Name})
}

// Update renames a group (admin only).
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	if user.Role != "admin" {
		httputil.WriteError(w, http.StatusForbidden, "forbidden", "Admin access required")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Group ID is required")
		return
	}

	var req UpdateGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Invalid request body")
		return
	}

	if req.Name == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Name is required")
		return
	}

	if err := h.repo.UpdateGroup(r.Context(), id, req.Name); err != nil {
		h.logger.Error("update group failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to update group")
		return
	}

	httputil.JSON(w, http.StatusOK, GroupResponse{ID: id, Name: req.Name})
}

// Delete removes a group (admin only).
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	if user.Role != "admin" {
		httputil.WriteError(w, http.StatusForbidden, "forbidden", "Admin access required")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Group ID is required")
		return
	}

	if err := h.repo.DeleteGroup(r.Context(), id); err != nil {
		h.logger.Error("delete group failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to delete group")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListMembers returns members of a group.
func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	if user.Role != "admin" {
		httputil.WriteError(w, http.StatusForbidden, "forbidden", "Admin access required")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Group ID is required")
		return
	}

	members, err := h.repo.ListMembers(r.Context(), id)
	if err != nil {
		h.logger.Error("list group members failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to list members")
		return
	}

	type MemberResponse struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name,omitempty"`
	}

	result := make([]MemberResponse, 0, len(members))
	for _, m := range members {
		name := ""
		if m.Name.Valid {
			name = m.Name.String
		}
		result = append(result, MemberResponse{
			ID:    m.ID,
			Email: m.Email,
			Name:  name,
		})
	}

	httputil.JSON(w, http.StatusOK, map[string]interface{}{"members": result})
}

// AddMember adds a user to a group (admin only).
func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	if user.Role != "admin" {
		httputil.WriteError(w, http.StatusForbidden, "forbidden", "Admin access required")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Group ID is required")
		return
	}

	var req AddMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Invalid request body")
		return
	}

	if req.UserID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "User ID is required")
		return
	}

	if err := h.repo.AddMember(r.Context(), id, req.UserID); err != nil {
		h.logger.Error("add group member failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to add member")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RemoveMember removes a user from a group (admin only).
func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	if user.Role != "admin" {
		httputil.WriteError(w, http.StatusForbidden, "forbidden", "Admin access required")
		return
	}

	groupID := r.PathValue("id")
	userID := r.PathValue("userId")
	if groupID == "" || userID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Group ID and User ID are required")
		return
	}

	if err := h.repo.RemoveMember(r.Context(), groupID, userID); err != nil {
		h.logger.Error("remove group member failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to remove member")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
