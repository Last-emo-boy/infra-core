package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/last-emo-boy/infra-core/pkg/config"
)

// Auth handles authentication and authorization
type Auth struct {
	config    *config.ConsoleConfig
	jwtSecret []byte
}

// Claims represents JWT token claims
type Claims struct {
	UserID      int      `json:"user_id"`
	Username    string   `json:"username"`
	Role        string   `json:"role"`
	SessionID   string   `json:"session_id"`
	Permissions []string `json:"permissions"`
	Services    []string `json:"services"`
	jwt.RegisteredClaims
}

// SSOClaims represents SSO-specific JWT token claims
type SSOClaims struct {
	*Claims
	SourceService string `json:"source_service,omitempty"`
	TargetService string `json:"target_service,omitempty"`
	RedirectURL   string `json:"redirect_url,omitempty"`
}

// LoginRequest represents login request data
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents login response data
type LoginResponse struct {
	Token     string `json:"token"`
	UserID    int    `json:"user_id"`
	Username  string `json:"username"`
	Role      string `json:"role"`
	ExpiresAt int64  `json:"expires_at"`
}

// NewAuth creates a new Auth instance
func NewAuth(config *config.ConsoleConfig) (*Auth, error) {
	jwtSecret := []byte(config.Auth.JWT.Secret)
	if len(jwtSecret) == 0 {
		// Generate a random secret if not provided
		randomSecret := make([]byte, 32)
		if _, err := rand.Read(randomSecret); err != nil {
			return nil, fmt.Errorf("failed to generate JWT secret: %w", err)
		}
		jwtSecret = []byte(hex.EncodeToString(randomSecret))
	}

	return &Auth{
		config:    config,
		jwtSecret: jwtSecret,
	}, nil
}

// HashPassword hashes a password using bcrypt
func (a *Auth) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword compares a password with its hash
func (a *Auth) CheckPassword(password, hash string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

// GenerateToken generates a JWT token for a user
func (a *Auth) GenerateToken(userID int, username, role string) (string, int64, error) {
	return a.GenerateTokenWithSession(userID, username, role, "", nil, nil)
}

// GenerateTokenWithSession generates a JWT token with session and service permissions
func (a *Auth) GenerateTokenWithSession(userID int, username, role, sessionID string, permissions, services []string) (string, int64, error) {
	expirationTime := time.Now().Add(time.Duration(a.config.Auth.JWT.ExpiresHours) * time.Hour)

	claims := &Claims{
		UserID:      userID,
		Username:    username,
		Role:        role,
		SessionID:   sessionID,
		Permissions: permissions,
		Services:    services,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "infra-core-sso",
			Subject:   fmt.Sprintf("user:%d", userID),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(a.jwtSecret)
	if err != nil {
		return "", 0, fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, expirationTime.Unix(), nil
}

// GenerateSSOToken generates a JWT token for SSO authentication
func (a *Auth) GenerateSSOToken(userID int, username, role, sessionID, sourceService, targetService, redirectURL string, permissions, services []string) (string, int64, error) {
	expirationTime := time.Now().Add(5 * time.Minute) // Short-lived token for SSO flow

	claims := &SSOClaims{
		Claims: &Claims{
			UserID:      userID,
			Username:    username,
			Role:        role,
			SessionID:   sessionID,
			Permissions: permissions,
			Services:    services,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(expirationTime),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				Issuer:    "infra-core-sso",
				Subject:   fmt.Sprintf("user:%d", userID),
				Audience:  []string{targetService},
			},
		},
		SourceService: sourceService,
		TargetService: targetService,
		RedirectURL:   redirectURL,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(a.jwtSecret)
	if err != nil {
		return "", 0, fmt.Errorf("failed to sign SSO token: %w", err)
	}

	return tokenString, expirationTime.Unix(), nil
}

// ValidateToken validates a JWT token and returns the claims
func (a *Auth) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// ValidateSSOToken validates an SSO JWT token and returns the claims
func (a *Auth) ValidateSSOToken(tokenString string) (*SSOClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &SSOClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return a.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse SSO token: %w", err)
	}

	if claims, ok := token.Claims.(*SSOClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid SSO token")
}

// RequireRole checks if the user has the required role
func (a *Auth) RequireRole(userRole, requiredRole string) bool {
	roleHierarchy := map[string]int{
		"user":  1,
		"admin": 2,
	}

	userLevel := roleHierarchy[userRole]
	requiredLevel := roleHierarchy[requiredRole]

	return userLevel >= requiredLevel
}

// HasServiceAccess checks if the user has access to a specific service
func (a *Auth) HasServiceAccess(claims *Claims, serviceName string) bool {
	// Admin users have access to all services
	if claims.Role == "admin" {
		return true
	}

	// Check if service is in the user's service list
	for _, service := range claims.Services {
		if service == serviceName {
			return true
		}
	}

	return false
}

// HasPermission checks if the user has a specific permission
func (a *Auth) HasPermission(claims *Claims, permission string) bool {
	// Admin users have all permissions
	if claims.Role == "admin" {
		return true
	}

	// Check if permission is in the user's permission list
	for _, perm := range claims.Permissions {
		if perm == permission {
			return true
		}
	}

	return false
}

// GenerateSessionToken creates a hashed token for session storage
func (a *Auth) GenerateSessionToken() (string, string, error) {
	// Generate random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", "", fmt.Errorf("failed to generate session token: %w", err)
	}
	
	token := hex.EncodeToString(tokenBytes)
	
	// Create hash for storage
	hasher := sha256.New()
	hasher.Write([]byte(token))
	tokenHash := hex.EncodeToString(hasher.Sum(nil))
	
	return token, tokenHash, nil
}

// HashSessionToken creates a hash of a session token for secure storage
func (a *Auth) HashSessionToken(token string) string {
	hasher := sha256.New()
	hasher.Write([]byte(token))
	return hex.EncodeToString(hasher.Sum(nil))
}
