package auth

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"log/slog"
	"math/big"
	"net/http"
	"strings"
	"time"

	sqlc "github.com/mostdoc/mostdoc/internal/db/query"
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
	mux.HandleFunc("GET /api/users", RequireAuth(h.ListUsers))
	mux.HandleFunc("GET /api/users/{id}", RequireAuth(h.GetUser))
	mux.HandleFunc("PATCH /api/users/{id}", RequireAuth(h.UpdateUser))
	mux.HandleFunc("DELETE /api/users/{id}", RequireAuth(h.DeleteUser))
	mux.HandleFunc("POST /api/users/invite", RequireAuth(h.Invite))
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

// Me returns the current authenticated user, querying the database for fresh data.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return
	}

	// Query database for fresh user data (global_role may have changed since JWT was issued)
	dbUser, err := h.repo.GetUserByID(r.Context(), user.ID)
	if err != nil {
		// Fall back to JWT claims if DB query fails
		httputil.JSON(w, http.StatusOK, MeResponse{User: user})
		return
	}

	freshUser := ScanUser(dbUser)
	httputil.JSON(w, http.StatusOK, MeResponse{User: freshUser})
}

// UserResponse is the public user representation.
type UserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
	Role  string `json:"role,omitempty"`
}

func toUserResponse(u sqlc.User) UserResponse {
	name := ""
	if u.Name.Valid {
		name = u.Name.String
	}
	role := ""
	if u.GlobalRole.Valid {
		role = u.GlobalRole.String
	}
	return UserResponse{
		ID:    u.ID,
		Email: u.Email,
		Name:  name,
		Role:  role,
	}
}

// ListUsers returns all users (admin only).
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := UserFromContext(r.Context())
	if !ok || currentUser.Role != "admin" {
		httputil.WriteError(w, http.StatusForbidden, "forbidden", "Admin access required")
		return
	}

	users, err := h.repo.ListUsers(r.Context())
	if err != nil {
		h.logger.Error("list users failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to list users")
		return
	}

	result := make([]UserResponse, 0, len(users))
	for _, u := range users {
		result = append(result, toUserResponse(u))
	}

	httputil.JSON(w, http.StatusOK, map[string]interface{}{"users": result})
}

// GetUser returns a single user.
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := UserFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "User ID is required")
		return
	}

	// Only admin or the user themselves can view their details
	if currentUser.Role != "admin" && currentUser.ID != id {
		httputil.WriteError(w, http.StatusForbidden, "forbidden", "Access denied")
		return
	}

	user, err := h.repo.GetUserByID(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "User not found")
		return
	}

	httputil.JSON(w, http.StatusOK, toUserResponse(user))
}

// UpdateUserRequest is the body for PATCH /api/users/{id}.
type UpdateUserRequest struct {
	Name *string `json:"name,omitempty"`
	Role *string `json:"role,omitempty"`
}

// UpdateUser updates a user's name or role.
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := UserFromContext(r.Context())
	if !ok {
		httputil.WriteError(w, http.StatusUnauthorized, "unauthorized", "Not authenticated")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "User ID is required")
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Invalid request body")
		return
	}

	// Name: user can change their own name
	if req.Name != nil {
		if currentUser.ID != id {
			httputil.WriteError(w, http.StatusForbidden, "forbidden", "Only the user can change their name")
			return
		}
		name := sql.NullString{String: *req.Name, Valid: *req.Name != ""}
		if err := h.repo.UpdateUserName(r.Context(), id, name); err != nil {
			h.logger.Error("update user name failed", "error", err)
			httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to update user")
			return
		}
	}

	// Role: only admin can change roles
	if req.Role != nil {
		if currentUser.Role != "admin" {
			httputil.WriteError(w, http.StatusForbidden, "forbidden", "Admin access required to change roles")
			return
		}
		if err := h.repo.UpdateUserRole(r.Context(), id, sql.NullString{String: *req.Role, Valid: *req.Role != ""}); err != nil {
			h.logger.Error("update user role failed", "error", err)
			httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to update user")
			return
		}
	}

	user, err := h.repo.GetUserByID(r.Context(), id)
	if err != nil {
		httputil.WriteError(w, http.StatusNotFound, "not_found", "User not found after update")
		return
	}

	httputil.JSON(w, http.StatusOK, toUserResponse(user))
}

// DeleteUser soft-deletes a user (admin only).
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := UserFromContext(r.Context())
	if !ok || currentUser.Role != "admin" {
		httputil.WriteError(w, http.StatusForbidden, "forbidden", "Admin access required")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "User ID is required")
		return
	}

	// Prevent deleting yourself
	if currentUser.ID == id {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Cannot delete yourself")
		return
	}

	if err := h.repo.DeleteUser(r.Context(), id); err != nil {
		h.logger.Error("delete user failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to delete user")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Invite creates a new user with a temporary password (admin only).
func (h *Handler) Invite(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := UserFromContext(r.Context())
	if !ok || currentUser.Role != "admin" {
		httputil.WriteError(w, http.StatusForbidden, "forbidden", "Admin access required")
		return
	}

	var req InviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Invalid request body")
		return
	}

	if req.Email == "" {
		httputil.WriteError(w, http.StatusBadRequest, "bad_request", "Email is required")
		return
	}

	// Default role to "viewer" if not provided
	role := req.Role
	if role == "" {
		role = "viewer"
	}

	// Generate random 16-character temporary password
	tempPassword, err := generateTempPassword(16)
	if err != nil {
		h.logger.Error("generate temp password failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to generate temporary password")
		return
	}

	// Hash the temporary password
	hash, err := h.service.HashPassword(tempPassword)
	if err != nil {
		h.logger.Error("hash password failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to hash password")
		return
	}

	// Create the user
	userID := GenerateID()
	ctx := r.Context()
	if err := h.repo.CreateUser(ctx, userID, req.Email, req.Name, hash, role); err != nil {
		h.logger.Error("create user failed", "error", err)
		httputil.WriteError(w, http.StatusInternalServerError, "internal_error", "Failed to create user")
		return
	}

	httputil.JSON(w, http.StatusCreated, InviteResponse{
		ID:           userID,
		Email:        req.Email,
		Name:         req.Name,
		Role:         role,
		TempPassword: tempPassword,
	})
}

// generateTempPassword generates a random password of the given length
// using alphanumeric characters and special characters.
func generateTempPassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	result := make([]byte, length)
	for i := range result {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		result[i] = charset[n.Int64()]
	}
	return string(result), nil
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
