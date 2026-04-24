package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/golang-jwt/jwt/v5"
)

const (
	jwtSecretKey   = "jwt_secret"
	jwtSecretFile  = "jwt.key"
	accessTokenTTL = 15 * time.Minute
	refreshTokenTTL= 7 * 24 * time.Hour
)

// Service holds auth business logic.
type Service struct {
	repo      *Repository
	dataDir   string
	secret    []byte
	bcryptCost int
}

// NewService creates the auth service and loads/generates the JWT secret.
func NewService(repo *Repository, dataDir string) (*Service, error) {
	secret, err := loadJWTSecret(repo, dataDir)
	if err != nil {
		return nil, fmt.Errorf("load jwt secret: %w", err)
	}

	cost := 12
	if c := os.Getenv("BCRYPT_COST"); c != "" {
		if v, err := strconv.Atoi(c); err == nil && v >= 4 && v <= 31 {
			cost = v
		}
	}

	return &Service{
		repo:       repo,
		dataDir:    dataDir,
		secret:     secret,
		bcryptCost: cost,
	}, nil
}

func loadJWTSecret(repo *Repository, dataDir string) ([]byte, error) {
	// 1. Try env var
	if s := os.Getenv("JWT_SECRET"); s != "" {
		return []byte(s), nil
	}

	// 2. Try database settings
	ctx := context.Background()
	if s, err := repo.GetSetting(ctx, jwtSecretKey); err == nil && s != "" {
		return []byte(s), nil
	}

	// 3. Try file on disk
	keyPath := filepath.Join(dataDir, jwtSecretFile)
	if b, err := os.ReadFile(keyPath); err == nil && len(b) >= 32 {
		return b, nil
	}

	// 4. Generate new secret
	b := make([]byte, 64)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("generate secret: %w", err)
	}

	// Persist to file
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	if err := os.WriteFile(keyPath, b, 0600); err != nil {
		return nil, fmt.Errorf("write secret file: %w", err)
	}

	// Also persist to DB for portability
	_ = repo.UpsertSetting(ctx, jwtSecretKey, base64.StdEncoding.EncodeToString(b))

	return b, nil
}

// HashPassword hashes a plaintext password with bcrypt.
func (s *Service) HashPassword(plain string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(plain), s.bcryptCost)
	return string(bytes), err
}

// CheckPassword compares a plaintext password with a bcrypt hash.
func (s *Service) CheckPassword(plain, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
	return err == nil
}

// IssueTokens creates a new access and refresh token pair for a user.
func (s *Service) IssueTokens(user UserInfo) (*TokenPair, *RefreshTokenMeta, error) {
	now := time.Now().UTC()

	// Access token
	accessClaims := jwt.MapClaims{
		"sub":   user.ID,
		"email": user.Email,
		"name":  user.Name,
		"role":  user.Role,
		"iat":   now.Unix(),
		"exp":   now.Add(accessTokenTTL).Unix(),
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessString, err := accessToken.SignedString(s.secret)
	if err != nil {
		return nil, nil, fmt.Errorf("sign access token: %w", err)
	}

	// Refresh token
	refreshID := GenerateID()
	refreshPlain := make([]byte, 32)
	if _, err := rand.Read(refreshPlain); err != nil {
		return nil, nil, fmt.Errorf("generate refresh token: %w", err)
	}
	refreshString := base64.URLEncoding.EncodeToString(refreshPlain)
	refreshHash, err := bcrypt.GenerateFromPassword([]byte(refreshString), s.bcryptCost)
	if err != nil {
		return nil, nil, fmt.Errorf("hash refresh token: %w", err)
	}

	meta := &RefreshTokenMeta{
		ID:        refreshID,
		Token:     refreshString,
		TokenHash: string(refreshHash),
		UserID:    user.ID,
		ExpiresAt: now.Add(refreshTokenTTL),
	}

	return &TokenPair{
		AccessToken:  accessString,
		RefreshToken: refreshString,
	}, meta, nil
}

// ParseAccessToken validates an access token and returns the user info.
func (s *Service) ParseAccessToken(tokenString string) (UserInfo, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return UserInfo{}, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return UserInfo{}, fmt.Errorf("invalid token claims")
	}

	return UserInfo{
		ID:    strClaim(claims, "sub"),
		Email: strClaim(claims, "email"),
		Name:  strClaim(claims, "name"),
		Role:  strClaim(claims, "role"),
	}, nil
}

// RotateRefreshToken validates a refresh token, invalidates the old one, and creates a new pair.
// tokenPlain is expected in format "id.plaintext" where id is the token DB id.
func (s *Service) RotateRefreshToken(ctx context.Context, tokenPlain string) (*TokenPair, *RefreshTokenMeta, UserInfo, error) {
	parts := strings.SplitN(tokenPlain, ".", 2)
	if len(parts) != 2 {
		return nil, nil, UserInfo{}, fmt.Errorf("invalid refresh token format")
	}
	id, plain := parts[0], parts[1]

	rt, err := s.repo.GetRefreshToken(ctx, id)
	if err != nil {
		return nil, nil, UserInfo{}, fmt.Errorf("refresh token not found")
	}

	if time.Now().UTC().After(time.Unix(rt.ExpiresAt, 0)) {
		_ = s.repo.DeleteRefreshToken(ctx, id)
		return nil, nil, UserInfo{}, fmt.Errorf("refresh token expired")
	}

	if !s.ValidateRefreshToken(rt.TokenHash, plain) {
		return nil, nil, UserInfo{}, fmt.Errorf("invalid refresh token")
	}

	// Fetch user
	user, err := s.repo.GetUserByID(ctx, rt.UserID)
	if err != nil {
		return nil, nil, UserInfo{}, fmt.Errorf("user not found")
	}

	u := ScanUser(user)

	// Issue new tokens
	tp, meta, err := s.IssueTokens(u)
	if err != nil {
		return nil, nil, UserInfo{}, err
	}

	// Save new refresh token
	if err := s.repo.CreateRefreshToken(ctx, meta.ID, meta.UserID, meta.TokenHash, meta.ExpiresAt); err != nil {
		return nil, nil, UserInfo{}, fmt.Errorf("store refresh token: %w", err)
	}

	// Delete old refresh token
	_ = s.repo.DeleteRefreshToken(ctx, id)

	return tp, meta, u, nil
}

// ValidateRefreshToken checks a refresh token hash against the stored hash.
func (s *Service) ValidateRefreshToken(storedHash, plainToken string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(plainToken))
	return err == nil
}

func strClaim(claims jwt.MapClaims, key string) string {
	if v, ok := claims[key].(string); ok {
		return v
	}
	return ""
}

// ConstantTimeCompare compares two strings in constant time to prevent timing attacks.
func ConstantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
