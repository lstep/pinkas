package pages

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/mostdoc/mostdoc/internal/auth"
	"github.com/mostdoc/mostdoc/internal/sse"
)

type saveRequest struct {
	DocID       string `json:"docId"`
	Markdown    string `json:"markdown"`
	YjsSnapshot []byte `json:"yjsSnapshot"`
	AuthorID    string `json:"authorId"`
}

type authResponse struct {
	UserID     string `json:"userId"`
	Permission string `json:"permission"`
}

func RegisterRoutes(mux *http.ServeMux, repo *Repository, logger *slog.Logger, dataDir string, authService *auth.Service, sseHub *sse.Hub) {
	handler := &Handler{repo: repo, logger: logger, dataDir: dataDir, authService: authService, sseHub: sseHub}
	restHandler := NewRESTHandler(repo, logger, sseHub)

	mux.HandleFunc("GET /internal/auth", handler.Auth)
	mux.HandleFunc("POST /internal/save", handler.Save)
	mux.HandleFunc("POST /internal/restore", handler.Restore)
	mux.HandleFunc("POST /internal/cleanup", handler.Cleanup)
	mux.HandleFunc("GET /internal/load", handler.Load)
	mux.HandleFunc("GET /health", handler.Health)

	restHandler.RegisterRESTRoutes(mux)
}

type Handler struct {
	repo        *Repository
	logger      *slog.Logger
	dataDir     string
	authService *auth.Service
	sseHub      *sse.Hub
}

func (h *Handler) Save(w http.ResponseWriter, r *http.Request) {
	var req saveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if req.DocID == "" {
		http.Error(w, "docId required", http.StatusBadRequest)
		return
	}

	// Look up page to get space slug for file path
	page, err := h.repo.GetPage(r.Context(), req.DocID)
	spaceSlug := "default"
	if err == nil && page.SpaceID.Valid {
		// TODO: look up space slug from space ID
		// For now, default space
		_ = page
	}

	// Write Markdown to disk
	mdPath := filepath.Join(h.dataDir, "docs", spaceSlug, req.DocID+".md")
	if err := os.MkdirAll(filepath.Dir(mdPath), 0755); err != nil {
		h.logger.Error("failed to create docs directory", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	if err := os.WriteFile(mdPath, []byte(req.Markdown), 0644); err != nil {
		h.logger.Error("failed to write markdown file", "error", err)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	// Save snapshot to database
	if req.YjsSnapshot != nil {
		if err := h.repo.SaveSnapshot(r.Context(), req.DocID, req.Markdown, req.YjsSnapshot, req.AuthorID); err != nil {
			h.logger.Error("failed to save snapshot", "error", err)
			// Don't fail the response if markdown was written to disk
		}
	}

	h.logger.Debug("saved document", "docId", req.DocID, "path", mdPath)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Auth(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, `{"error":"missing token"}`, http.StatusUnauthorized)
		return
	}

	user, err := h.authService.ParseAccessToken(token)
	if err != nil {
		http.Error(w, `{"error":"invalid token"}`, http.StatusForbidden)
		return
	}

	// Iteration 2: all authenticated users have admin permission
	// Full permission resolution deferred to Iteration 3
	resp := authResponse{
		UserID:     user.ID,
		Permission: "admin",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Restore(w http.ResponseWriter, r *http.Request) {
	// Iteration 1: stub — sidecar will broadcast snapshot
	h.logger.Info("restore requested (stub)")
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Cleanup(w http.ResponseWriter, r *http.Request) {
	// Iteration 1: stub
	h.logger.Info("cleanup requested (stub)")
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Load(w http.ResponseWriter, r *http.Request) {
	docId := r.URL.Query().Get("docId")
	if docId == "" {
		http.Error(w, "docId required", http.StatusBadRequest)
		return
	}
	snapshot, err := h.repo.GetLatestSnapshot(r.Context(), docId)
	if err != nil || snapshot == nil {
		h.logger.Debug("no snapshot found, returning empty", "docId", docId)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"yjsSnapshot": nil})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"yjsSnapshot": snapshot.YjsSnapshot,
	})
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
