package pages

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mostdoc/mostdoc/internal/auth"
	"github.com/mostdoc/mostdoc/internal/httputil"
	"github.com/mostdoc/mostdoc/internal/permissions"
)

// SnapshotResponse is the JSON representation of a snapshot.
type SnapshotResponse struct {
	ID        string `json:"id"`
	PageID    string `json:"pageId"`
	Label     string `json:"label,omitempty"`
	Markdown  string `json:"markdown,omitempty"`
	AuthorID  string `json:"authorId,omitempty"`
	CreatedAt int64  `json:"createdAt"`
}

// ListSnapshots returns all snapshots for a page.
func (h *RESTHandler) ListSnapshots(w http.ResponseWriter, r *http.Request) {
	pageID := r.PathValue("id")
	if pageID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Page ID is required")
		return
	}

	if !h.checkAccess(w, r, "page", pageID, permissions.LevelViewer) {
		return
	}

	snapshots, err := h.repo.ListSnapshots(r.Context(), pageID)
	if err != nil {
		h.logger.Error("list snapshots failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to list snapshots")
		return
	}

	result := make([]SnapshotResponse, 0, len(snapshots))
	for _, s := range snapshots {
		result = append(result, SnapshotResponse{
			ID:        s.ID,
			PageID:    s.PageID,
			Label:     s.Label,
			CreatedAt: s.CreatedAt * 1000,
		})
	}

	httputil.JSON(w, http.StatusOK, map[string]interface{}{"snapshots": result})
}

// GetSnapshot returns a single snapshot by ID.
func (h *RESTHandler) GetSnapshot(w http.ResponseWriter, r *http.Request) {
	pageID := r.PathValue("id")
	snapshotID := r.PathValue("snapshotId")
	if pageID == "" || snapshotID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Page ID and snapshot ID are required")
		return
	}

	if !h.checkAccess(w, r, "page", pageID, permissions.LevelViewer) {
		return
	}

	snapshot, err := h.repo.GetSnapshotByID(r.Context(), snapshotID)
	if err != nil {
		h.logger.Error("get snapshot failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to get snapshot")
		return
	}
	if snapshot == nil {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "Snapshot not found")
		return
	}

	resp := map[string]interface{}{
		"id":          snapshot.ID,
		"pageId":      snapshot.PageID,
		"label":       snapshot.Label,
		"markdown":    snapshot.Markdown,
		"authorId":    snapshot.AuthorID,
		"createdAt":   snapshot.CreatedAt * 1000,
		"yjsSnapshot": base64.StdEncoding.EncodeToString(snapshot.YjsSnapshot),
	}

	httputil.JSON(w, http.StatusOK, resp)
}

// RestoreSnapshot restores a page to a snapshot.
func (h *RESTHandler) RestoreSnapshot(w http.ResponseWriter, r *http.Request) {
	pageID := r.PathValue("id")
	if pageID == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Page ID is required")
		return
	}

	if !h.checkAccess(w, r, "page", pageID, permissions.LevelEditor) {
		return
	}

	var req struct {
		SnapshotID string `json:"snapshotId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Invalid request body")
		return
	}

	var snapshot *Snapshot
	var err error

	if req.SnapshotID != "" {
		snapshot, err = h.repo.GetSnapshotByID(r.Context(), req.SnapshotID)
		if err != nil {
			h.logger.Error("get snapshot failed", "error", err)
			httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to get snapshot")
			return
		}
		if snapshot == nil {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "Snapshot not found")
			return
		}
	} else {
		// Use latest snapshot
		snapshot, err = h.repo.GetLatestSnapshot(r.Context(), pageID)
		if err != nil {
			h.logger.Error("get latest snapshot failed", "error", err)
			httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to get latest snapshot")
			return
		}
		if snapshot == nil {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "No snapshots found for page")
			return
		}
	}

	// Save pre-restore snapshot
	user, _ := auth.UserFromContext(r.Context())
	_, err = h.repo.SaveSnapshotWithLabel(r.Context(), pageID, snapshot.Markdown, snapshot.YjsSnapshot, user.ID, "pre-restore")
	if err != nil {
		h.logger.Error("save pre-restore snapshot failed", "error", err)
		// Don't fail the request, just log the error
	}

	// Forward to sidecar
	if h.collabURL != "" {
		payload := map[string]interface{}{
			"docId":       pageID,
			"yjsSnapshot": base64.StdEncoding.EncodeToString(snapshot.YjsSnapshot),
		}
		payloadBytes, _ := json.Marshal(payload)
		restoreURL := h.collabURL + "/internal/restore"
		resp, err := http.Post(restoreURL, "application/json", bytes.NewReader(payloadBytes))
		if err != nil {
			h.logger.Error("forward restore to sidecar failed", "error", err)
			httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to forward restore to sidecar")
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			h.logger.Error("sidecar restore returned non-OK status", "status", resp.StatusCode)
			httputil.WriteError(w, http.StatusInternalServerError, "internal_error", fmt.Sprintf("Sidecar restore failed with status %d", resp.StatusCode))
			return
		}
	}

	httputil.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
