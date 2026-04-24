package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/mostdoc/mostdoc/internal/httputil"
)

// Handler holds HTTP handlers for auth endpoints.
type Handler struct {
	service *Service
	repo    *Repository
	logger  *slog.Logger
}

// NewHandler creates a new auth handler.
func NewHandler(service *Service, repo *Repository, logger *slog.Logger) *Handler {
	return &Handler{service: service, repo: repo, logger: logger}
}

// RegisterRoutes registers auth routes on the mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/register", h.Register)
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("POST /api/auth/refresh", h.Refresh)
	mux.HandleFunc("POST /api/auth/logout", h.Logout)
	mux.HandleFunc("GET /api/auth/me", RequireAuth(h.Me))
}

// Register handles first admin registration.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Email and password are required")
		return
	}

	ctx := r.Context()

	// Only allow registration when zero users exist
	count, err := h.repo.CountUsers(ctx)
	if err != nil {
		h.logger.Error("count users failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Database error")
		return
	}
	if count > 0 {
		httputil.WriteError(w, http.StatusForbidden, "registration_closed", "Registration is closed. Contact an admin for an invite.")
		return
	}

	// Hash password
	hash, err := h.service.HashPassword(req.Password)
	if err != nil {
		h.logger.Error("hash password failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to hash password")
		return
	}

	// Create first admin
	userID := GenerateID()
	if err := h.repo.CreateUser(ctx, userID, req.Email, req.Name, hash, "admin"); err != nil {
		h.logger.Error("create user failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to create user")
		return
	}

	user := UserInfo{ID: userID, Email: req.Email, Name: req.Name, Role: "admin"}

	// Issue tokens
	tp, meta, err := h.service.IssueTokens(user)
	if err != nil {
		h.logger.Error("issue tokens failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to issue tokens")
		return
	}

	if err := h.repo.CreateRefreshToken(ctx, meta.ID, meta.UserID, meta.TokenHash, meta.ExpiresAt); err != nil {
		h.logger.Error("create refresh token failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to store refresh token")
		return
	}

	setRefreshCookie(w, meta.ID+"."+meta.Token)

	httputil.JSON(w, http.StatusCreated, LoginResponse{
		User:  user,
		Token: *tp,
	})
}

// Login handles user login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Email and password are required")
		return
	}

	ctx := r.Context()
	userRow, err := h.repo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
		return
	}

	if !h.service.CheckPassword(req.Password, userRow.PasswordHash) {
		httputil.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
		return
	}

	user := ScanUser(userRow)

	tp, meta, err := h.service.IssueTokens(user)
	if err != nil {
		h.logger.Error("issue tokens failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to issue tokens")
		return
	}

	if err := h.repo.CreateRefreshToken(ctx, meta.ID, meta.UserID, meta.TokenHash, meta.ExpiresAt); err != nil {
		h.logger.Error("create refresh token failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to store refresh token")
		return
	}

	setRefreshCookie(w, meta.ID+"."+meta.Token)

	httputil.JSON(w, http.StatusOK, LoginResponse{
		User:  user,
		Token: *tp,
	})
}

// Refresh rotates the refresh token and issues new tokens.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		httputil.WriteError(w, http.StatusUnauthorized, "missing_refresh_token", "No refresh token provided")
		return
	}

	ctx := r.Context()
	tp, meta, user, err := h.service.RotateRefreshToken(ctx, cookie.Value)
	if err != nil {
		h.logger.Debug("refresh failed", "error", err)
		httputil.WriteError(w, http.StatusUnauthorized, "invalid_refresh_token", "Invalid or expired refresh token")
		return
	}

	setRefreshCookie(w, meta.ID+"."+meta.Token)

	httputil.JSON(w, http.StatusOK, LoginResponse{
		User:  user,
		Token: *tp,
	})
}

// Logout invalidates the refresh token.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err == nil && cookie.Value != "" {
		parts := strings.SplitN(cookie.Value, ".", 2)
		if len(parts) == 2 {
			_ = h.repo.DeleteRefreshToken(r.Context(), parts[0])
		}
	}

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	w.WriteHeader(http.StatusNoContent)
}

// Me returns the current authenticated user.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return
	}

	httputil.JSON(w, http.StatusOK, MeResponse{User: user})
}

func setRefreshCookie(w http.ResponseWriter, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    value,
		Path:     "/",
		MaxAge:   int(7 * 24 * time.Hour / time.Second),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   false, // Set to true when behind HTTPS
	})
}
