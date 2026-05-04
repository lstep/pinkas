package permissions

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/mostdoc/mostdoc/internal/auth"
	"github.com/mostdoc/mostdoc/internal/httputil"
)

// Handler holds HTTP handlers for permission management endpoints.
type Handler struct {
	repo   *Repository
	logger *slog.Logger
}

// NewHandler creates a new permissions handler.
func NewHandler(repo *Repository, logger *slog.Logger) *Handler {
	return &Handler{repo: repo, logger: logger}
}

// RegisterRoutes registers permission management routes (admin only).
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/permissions", auth.RequireAuth(h.ListForTarget))
	mux.HandleFunc("POST /api/permissions", auth.RequireAuth(h.SetPermission))
	mux.HandleFunc("DELETE /api/permissions", auth.RequireAuth(h.RemovePermission))
}

// PermissionResponse is the JSON representation of a permission.
type PermissionResponse struct {
	TargetType  string `json:"targetType"`
	TargetID    string `json:"targetId"`
	GranteeType string `json:"granteeType"`
	GranteeID   string `json:"granteeId"`
	Level       int64  `json:"level"`
}

// SetPermissionRequest is the body for POST /api/permissions.
type SetPermissionRequest struct {
	TargetType  string `json:"targetType"`
	TargetID    string `json:"targetId"`
	GranteeType string `json:"granteeType"`
	GranteeID   string `json:"granteeId"`
	Level       int64  `json:"level"`
}

// RemovePermissionRequest is the body for DELETE /api/permissions.
type RemovePermissionRequest struct {
	TargetType  string `json:"targetType"`
	TargetID    string `json:"targetId"`
	GranteeType string `json:"granteeType"`
	GranteeID   string `json:"granteeId"`
}

// ListForTarget returns all permissions for a target (admin only).
// Query params: targetType, targetId
func (h *Handler) ListForTarget(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	if user.Role != "admin" {
		httputil.WriteError(w, http.StatusForbidden, "forbidden", "Admin access required")
		return
	}

	targetType := r.URL.Query().Get("targetType")
	targetID := r.URL.Query().Get("targetId")
	if targetType == "" || targetID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "targetType and targetId query params required")
		return
	}

	perms, err := h.repo.ListPermissionsByTarget(r.Context(), targetType, targetID)
	if err != nil {
		h.logger.Error("list permissions failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to list permissions")
		return
	}

	result := make([]PermissionResponse, 0, len(perms))
	for _, p := range perms {
		result = append(result, PermissionResponse{
			TargetType:  p.TargetType,
			TargetID:    p.TargetID,
			GranteeType: p.GranteeType,
			GranteeID:   p.GranteeID,
			Level:       p.Level,
		})
	}

	httputil.JSON(w, http.StatusOK, map[string]interface{}{"permissions": result})
}

// SetPermission creates or updates a permission (admin only).
func (h *Handler) SetPermission(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	if user.Role != "admin" {
		httputil.WriteError(w, http.StatusForbidden, "forbidden", "Admin access required")
		return
	}

	var req SetPermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Invalid request body")
		return
	}

	if req.TargetType == "" || req.TargetID == "" || req.GranteeType == "" || req.GranteeID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "targetType, targetId, granteeType, granteeId are required")
		return
	}

	if err := h.repo.UpsertPermission(r.Context(), req.TargetType, req.TargetID, req.GranteeType, req.GranteeID, user.ID, req.Level); err != nil {
		h.logger.Error("set permission failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to set permission")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// RemovePermission deletes a permission (admin only).
func (h *Handler) RemovePermission(w http.ResponseWriter, r *http.Request) {
	user, _ := auth.UserFromContext(r.Context())
	if user.Role != "admin" {
		httputil.WriteError(w, http.StatusForbidden, "forbidden", "Admin access required")
		return
	}

	var req RemovePermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Invalid request body")
		return
	}

	if err := h.repo.DeletePermission(r.Context(), req.TargetType, req.TargetID, req.GranteeType, req.GranteeID); err != nil {
		h.logger.Error("remove permission failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to remove permission")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
