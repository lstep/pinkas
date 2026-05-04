package attachments

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/mostdoc/mostdoc/internal/auth"
	"github.com/mostdoc/mostdoc/internal/httputil"
	"github.com/mostdoc/mostdoc/internal/permissions"
)

type Handler struct {
	logger       *slog.Logger
	dataDir      string
	permResolver *permissions.Resolver
	authService  *auth.Service
}

func NewHandler(logger *slog.Logger, dataDir string, permResolver *permissions.Resolver, authService *auth.Service) *Handler {
	return &Handler{logger: logger, dataDir: dataDir, permResolver: permResolver, authService: authService}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/attachments", auth.RequireAuth(h.Upload))
	mux.HandleFunc("GET /api/files/{pageId}/{filename}", h.Serve)
	mux.HandleFunc("GET /api/files/{pageId}/{filename}/", h.Serve)
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Failed to parse multipart form")
		return
	}

	pageId := r.FormValue("pageId")
	if pageId == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "pageId is required")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Failed to get file from form")
		return
	}
	defer file.Close()

	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "User not found in context")
		return
	}

	if user.Role != "admin" {
		level := h.permResolver.Resolve(r.Context(), user.ID, "page", pageId)
		if level < permissions.LevelEditor {
			httputil.WriteError(w, http.StatusForbidden, "forbidden", "Editor access required")
			return
		}
	}

	contentType := header.Header.Get("Content-Type")
	allowedTypes := map[string]bool{
		"image/jpeg":  true,
		"image/png":   true,
		"image/gif":   true,
		"image/webp":  true,
		"image/svg+xml": true,
		"application/pdf": true,
		"application/msword": true,
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document": true,
		"text/plain":  true,
		"text/markdown": true,
	}
	if !allowedTypes[contentType] {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "File type not allowed")
		return
	}

	filename := uuid.New().String() + "-" + header.Filename
	dirPath := filepath.Join(h.dataDir, "attachments", pageId)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		h.logger.Error("failed to create attachments directory", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to create directory")
		return
	}

	filePath := filepath.Join(dirPath, filename)
	dst, err := os.Create(filePath)
	if err != nil {
		h.logger.Error("failed to create file", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to save file")
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		h.logger.Error("failed to copy file content", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to save file")
		return
	}

	httputil.JSON(w, http.StatusCreated, map[string]string{
		"url":      fmt.Sprintf("/api/files/%s/%s", pageId, filename),
		"filename": filename,
	})
}

func (h *Handler) Serve(w http.ResponseWriter, r *http.Request) {
	pageId := r.PathValue("pageId")
	filename := r.PathValue("filename")

	if pageId == "" || filename == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "pageId and filename are required")
		return
	}

	// Authenticate: check Authorization header first, then ?token= query param
	// (query param is needed for <img> tags which can't set custom headers)
	var userInfo auth.UserInfo
	var ok bool
	var authErr error
	userInfo, ok = auth.UserFromContext(r.Context())
	if !ok {
		token := r.URL.Query().Get("token")
		if token == "" {
			httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
			return
		}
		userInfo, authErr = h.authService.ParseAccessToken(token)
		if authErr != nil {
			httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Invalid token")
			return
		}
	}

	if userInfo.Role != "admin" {
		level := h.permResolver.Resolve(r.Context(), userInfo.ID, "page", pageId)
		if level < permissions.LevelViewer {
			httputil.WriteError(w, http.StatusForbidden, "forbidden", "Viewer access required")
			return
		}
	}

	safeFilename := filepath.Base(filename)
	filePath := filepath.Join(h.dataDir, "attachments", pageId, safeFilename)

	stat, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			httputil.WriteError(w, http.StatusNotFound, "not_found", "File not found")
			return
		}
		h.logger.Error("failed to stat file", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to access file")
		return
	}

	if stat.IsDir() {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "File not found")
		return
	}

	contentType := mimeTypeByExtension(safeFilename)
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", "inline")

	f, err := os.Open(filePath)
	if err != nil {
		h.logger.Error("failed to open file", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to read file")
		return
	}
	defer f.Close()

	http.ServeContent(w, r, safeFilename, stat.ModTime(), f)
}

func mimeTypeByExtension(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".pdf":
		return "application/pdf"
	case ".doc":
		return "application/msword"
	case ".docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case ".txt":
		return "text/plain"
	case ".md":
		return "text/markdown"
	default:
		return "application/octet-stream"
	}
}
