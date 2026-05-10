package auth

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/pinkas/pinkas/internal/db"
)

func setupTestAuth(t *testing.T) (*Handler, func()) {
	dataDir := t.TempDir()
	conn, err := db.Open(dataDir)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.Migrate(conn, "file://../../migrations"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	repo := NewRepository(conn)
	svc, err := NewService(repo, t.TempDir())
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	h := NewHandler(svc, repo, logger)

	return h, func() { conn.Close() }
}

func TestRegisterFirstAdmin(t *testing.T) {
	h, cleanup := setupTestAuth(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]string{
		"email":    "admin@example.com",
		"password": "password123",
		"name":     "Admin",
	})
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Register(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["token"] == nil {
		t.Fatal("expected token in response")
	}
}

func TestRegisterRejectsSecondUser(t *testing.T) {
	h, cleanup := setupTestAuth(t)
	defer cleanup()

	// First registration
	body, _ := json.Marshal(map[string]string{
		"email":    "admin@example.com",
		"password": "password123",
		"name":     "Admin",
	})
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	h.Register(httptest.NewRecorder(), req)

	// Second registration should fail
	body2, _ := json.Marshal(map[string]string{
		"email":    "user@example.com",
		"password": "password123",
		"name":     "User",
	})
	req2 := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()

	h.Register(w2, req2)

	if w2.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w2.Code)
	}
}

func TestLogin(t *testing.T) {
	h, cleanup := setupTestAuth(t)
	defer cleanup()

	// Register first
	body, _ := json.Marshal(map[string]string{
		"email":    "admin@example.com",
		"password": "password123",
		"name":     "Admin",
	})
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	h.Register(httptest.NewRecorder(), req)

	// Login
	loginBody, _ := json.Marshal(map[string]string{
		"email":    "admin@example.com",
		"password": "password123",
	})
	loginReq := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Login(w, loginReq)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["token"] == nil {
		t.Fatal("expected token in response")
	}

	// Check Set-Cookie header for refresh token
	cookies := w.Result().Cookies()
	var hasRefresh bool
	for _, c := range cookies {
		if c.Name == "refresh_token" {
			hasRefresh = true
			break
		}
	}
	if !hasRefresh {
		t.Fatal("expected refresh_token cookie")
	}
}

func TestLoginWrongPassword(t *testing.T) {
	h, cleanup := setupTestAuth(t)
	defer cleanup()

	// Register
	body, _ := json.Marshal(map[string]string{
		"email":    "admin@example.com",
		"password": "password123",
		"name":     "Admin",
	})
	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	h.Register(httptest.NewRecorder(), req)

	// Login with wrong password
	loginBody, _ := json.Marshal(map[string]string{
		"email":    "admin@example.com",
		"password": "wrongpassword",
	})
	loginReq := httptest.NewRequest("POST", "/api/auth/login", bytes.NewReader(loginBody))
	loginReq.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Login(w, loginReq)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
