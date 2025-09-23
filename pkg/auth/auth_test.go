package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/last-emo-boy/infra-core/pkg/config"
)

func TestNewAuth(t *testing.T) {
	tests := []struct {
		name           string
		config         *config.ConsoleConfig
		expectedError  bool
		secretNotEmpty bool
	}{
		{
			name: "with valid config and secret",
			config: &config.ConsoleConfig{
				Auth: config.AuthConfig{
					JWT: config.JWTConfig{
						Secret:       "test-secret-key",
						ExpiresHours: 24,
					},
				},
			},
			expectedError:  false,
			secretNotEmpty: true,
		},
		{
			name: "with empty secret generates random",
			config: &config.ConsoleConfig{
				Auth: config.AuthConfig{
					JWT: config.JWTConfig{
						Secret:       "",
						ExpiresHours: 24,
					},
				},
			},
			expectedError:  false,
			secretNotEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth, err := NewAuth(tt.config)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, auth)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, auth)
				assert.Equal(t, tt.config, auth.config)
				if tt.secretNotEmpty {
					assert.NotEmpty(t, auth.jwtSecret)
				}
			}
		})
	}
}

func TestHashPassword(t *testing.T) {
	auth := &Auth{}

	tests := []struct {
		name     string
		password string
	}{
		{
			name:     "normal password",
			password: "password123",
		},
		{
			name:     "complex password",
			password: "P@ssw0rd!@#$%^&*()",
		},
		{
			name:     "empty password",
			password: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := auth.HashPassword(tt.password)
			assert.NoError(t, err)
			assert.NotEmpty(t, hash)
			assert.NotEqual(t, tt.password, hash)
		})
	}
}

func TestCheckPassword(t *testing.T) {
	auth := &Auth{}
	password := "testpassword123"
	
	hash, err := auth.HashPassword(password)
	require.NoError(t, err)

	tests := []struct {
		name        string
		password    string
		hash        string
		expectError bool
	}{
		{
			name:        "correct password",
			password:    password,
			hash:        hash,
			expectError: false,
		},
		{
			name:        "incorrect password",
			password:    "wrongpassword",
			hash:        hash,
			expectError: true,
		},
		{
			name:        "empty password",
			password:    "",
			hash:        hash,
			expectError: true,
		},
		{
			name:        "invalid hash",
			password:    password,
			hash:        "invalid-hash",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := auth.CheckPassword(tt.password, tt.hash)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGenerateToken(t *testing.T) {
	auth := &Auth{
		config: &config.ConsoleConfig{
			Auth: config.AuthConfig{
				JWT: config.JWTConfig{
					Secret:       "test-secret",
					ExpiresHours: 24,
				},
			},
		},
		jwtSecret: []byte("test-secret"),
	}

	tests := []struct {
		name     string
		userID   int
		username string
		role     string
	}{
		{
			name:     "admin user",
			userID:   1,
			username: "admin",
			role:     "admin",
		},
		{
			name:     "regular user",
			userID:   2,
			username: "user",
			role:     "user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, expiresAt, err := auth.GenerateToken(tt.userID, tt.username, tt.role)
			
			assert.NoError(t, err)
			assert.NotEmpty(t, token)
			assert.Greater(t, expiresAt, time.Now().Unix())

			// Validate the generated token
			claims, err := auth.ValidateToken(token)
			assert.NoError(t, err)
			assert.Equal(t, tt.userID, claims.UserID)
			assert.Equal(t, tt.username, claims.Username)
			assert.Equal(t, tt.role, claims.Role)
		})
	}
}

func TestGenerateTokenWithSession(t *testing.T) {
	auth := &Auth{
		config: &config.ConsoleConfig{
			Auth: config.AuthConfig{
				JWT: config.JWTConfig{
					Secret:       "test-secret",
					ExpiresHours: 24,
				},
			},
		},
		jwtSecret: []byte("test-secret"),
	}

	userID := 1
	username := "testuser"
	role := "admin"
	sessionID := "session123"
	permissions := []string{"read", "write", "admin"}
	services := []string{"service1", "service2"}

	token, expiresAt, err := auth.GenerateTokenWithSession(userID, username, role, sessionID, permissions, services)
	
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Greater(t, expiresAt, time.Now().Unix())

	// Validate the generated token
	claims, err := auth.ValidateToken(token)
	assert.NoError(t, err)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, username, claims.Username)
	assert.Equal(t, role, claims.Role)
	assert.Equal(t, sessionID, claims.SessionID)
	assert.Equal(t, permissions, claims.Permissions)
	assert.Equal(t, services, claims.Services)
}

func TestGenerateSSOToken(t *testing.T) {
	auth := &Auth{
		config: &config.ConsoleConfig{
			Auth: config.AuthConfig{
				JWT: config.JWTConfig{
					Secret:       "test-secret",
					ExpiresHours: 24,
				},
			},
		},
		jwtSecret: []byte("test-secret"),
	}

	userID := 1
	username := "testuser"
	role := "admin"
	sessionID := "session123"
	sourceService := "service1"
	targetService := "service2"
	redirectURL := "https://example.com/callback"
	permissions := []string{"read", "write"}
	services := []string{"service1", "service2"}

	token, expiresAt, err := auth.GenerateSSOToken(userID, username, role, sessionID, sourceService, targetService, redirectURL, permissions, services)
	
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Greater(t, expiresAt, time.Now().Unix())
	// SSO tokens should expire within 5 minutes
	assert.Less(t, expiresAt, time.Now().Add(6*time.Minute).Unix())

	// Validate the generated SSO token
	claims, err := auth.ValidateSSOToken(token)
	assert.NoError(t, err)
	assert.Equal(t, userID, claims.Claims.UserID)
	assert.Equal(t, username, claims.Claims.Username)
	assert.Equal(t, role, claims.Claims.Role)
	assert.Equal(t, sessionID, claims.Claims.SessionID)
	assert.Equal(t, sourceService, claims.SourceService)
	assert.Equal(t, targetService, claims.TargetService)
	assert.Equal(t, redirectURL, claims.RedirectURL)
	assert.Equal(t, permissions, claims.Claims.Permissions)
	assert.Equal(t, services, claims.Claims.Services)
}

func TestValidateToken(t *testing.T) {
	auth := &Auth{
		config: &config.ConsoleConfig{
			Auth: config.AuthConfig{
				JWT: config.JWTConfig{
					Secret:       "test-secret",
					ExpiresHours: 24,
				},
			},
		},
		jwtSecret: []byte("test-secret"),
	}

	// Generate a valid token
	validToken, _, err := auth.GenerateToken(1, "testuser", "admin")
	require.NoError(t, err)

	// Generate an expired token
	expiredClaims := &Claims{
		UserID:   1,
		Username: "testuser",
		Role:     "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, expiredClaims)
	expiredTokenString, err := expiredToken.SignedString(auth.jwtSecret)
	require.NoError(t, err)

	tests := []struct {
		name        string
		token       string
		expectError bool
	}{
		{
			name:        "valid token",
			token:       validToken,
			expectError: false,
		},
		{
			name:        "expired token",
			token:       expiredTokenString,
			expectError: true,
		},
		{
			name:        "invalid token",
			token:       "invalid.token.here",
			expectError: true,
		},
		{
			name:        "empty token",
			token:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := auth.ValidateToken(tt.token)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
				assert.Equal(t, 1, claims.UserID)
				assert.Equal(t, "testuser", claims.Username)
				assert.Equal(t, "admin", claims.Role)
			}
		})
	}
}

func TestValidateSSOToken(t *testing.T) {
	auth := &Auth{
		config: &config.ConsoleConfig{
			Auth: config.AuthConfig{
				JWT: config.JWTConfig{
					Secret:       "test-secret",
					ExpiresHours: 24,
				},
			},
		},
		jwtSecret: []byte("test-secret"),
	}

	// Generate a valid SSO token
	validToken, _, err := auth.GenerateSSOToken(1, "testuser", "admin", "session123", "service1", "service2", "https://example.com", []string{"read"}, []string{"service1", "service2"})
	require.NoError(t, err)

	tests := []struct {
		name        string
		token       string
		expectError bool
	}{
		{
			name:        "valid SSO token",
			token:       validToken,
			expectError: false,
		},
		{
			name:        "invalid token",
			token:       "invalid.token.here",
			expectError: true,
		},
		{
			name:        "empty token",
			token:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := auth.ValidateSSOToken(tt.token)
			
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, claims)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, claims)
				assert.Equal(t, 1, claims.Claims.UserID)
				assert.Equal(t, "testuser", claims.Claims.Username)
				assert.Equal(t, "admin", claims.Claims.Role)
				assert.Equal(t, "service1", claims.SourceService)
				assert.Equal(t, "service2", claims.TargetService)
			}
		})
	}
}

func TestRequireRole(t *testing.T) {
	auth := &Auth{}

	tests := []struct {
		name         string
		userRole     string
		requiredRole string
		expected     bool
	}{
		{
			name:         "admin accessing admin endpoint",
			userRole:     "admin",
			requiredRole: "admin",
			expected:     true,
		},
		{
			name:         "admin accessing user endpoint",
			userRole:     "admin",
			requiredRole: "user",
			expected:     true,
		},
		{
			name:         "user accessing user endpoint",
			userRole:     "user",
			requiredRole: "user",
			expected:     true,
		},
		{
			name:         "user accessing admin endpoint",
			userRole:     "user",
			requiredRole: "admin",
			expected:     false,
		},
		{
			name:         "unknown role",
			userRole:     "unknown",
			requiredRole: "user",
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := auth.RequireRole(tt.userRole, tt.requiredRole)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasServiceAccess(t *testing.T) {
	auth := &Auth{}

	tests := []struct {
		name        string
		claims      *Claims
		serviceName string
		expected    bool
	}{
		{
			name: "admin user has access to any service",
			claims: &Claims{
				Role:     "admin",
				Services: []string{"service1"},
			},
			serviceName: "service2",
			expected:    true,
		},
		{
			name: "user has access to allowed service",
			claims: &Claims{
				Role:     "user",
				Services: []string{"service1", "service2"},
			},
			serviceName: "service1",
			expected:    true,
		},
		{
			name: "user does not have access to restricted service",
			claims: &Claims{
				Role:     "user",
				Services: []string{"service1"},
			},
			serviceName: "service2",
			expected:    false,
		},
		{
			name: "user with no services",
			claims: &Claims{
				Role:     "user",
				Services: []string{},
			},
			serviceName: "service1",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := auth.HasServiceAccess(tt.claims, tt.serviceName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHasPermission(t *testing.T) {
	auth := &Auth{}

	tests := []struct {
		name       string
		claims     *Claims
		permission string
		expected   bool
	}{
		{
			name: "admin user has all permissions",
			claims: &Claims{
				Role:        "admin",
				Permissions: []string{"read"},
			},
			permission: "write",
			expected:   true,
		},
		{
			name: "user has specific permission",
			claims: &Claims{
				Role:        "user",
				Permissions: []string{"read", "write"},
			},
			permission: "read",
			expected:   true,
		},
		{
			name: "user does not have permission",
			claims: &Claims{
				Role:        "user",
				Permissions: []string{"read"},
			},
			permission: "write",
			expected:   false,
		},
		{
			name: "user with no permissions",
			claims: &Claims{
				Role:        "user",
				Permissions: []string{},
			},
			permission: "read",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := auth.HasPermission(tt.claims, tt.permission)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateSessionToken(t *testing.T) {
	auth := &Auth{}

	token, hash, err := auth.GenerateSessionToken()
	
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, token, hash)
	assert.Len(t, token, 64) // 32 bytes hex encoded
	assert.Len(t, hash, 64)  // SHA256 hex encoded
}

func TestHashSessionToken(t *testing.T) {
	auth := &Auth{}
	token := "test-session-token"

	hash1 := auth.HashSessionToken(token)
	hash2 := auth.HashSessionToken(token)

	assert.NotEmpty(t, hash1)
	assert.Equal(t, hash1, hash2) // Same input should produce same hash
	assert.NotEqual(t, token, hash1) // Hash should be different from input
	assert.Len(t, hash1, 64) // SHA256 hex encoded
}

func TestTokenIntegration(t *testing.T) {
	auth := &Auth{
		config: &config.ConsoleConfig{
			Auth: config.AuthConfig{
				JWT: config.JWTConfig{
					Secret:       "integration-test-secret",
					ExpiresHours: 1,
				},
			},
		},
		jwtSecret: []byte("integration-test-secret"),
	}

	// Test complete flow: generate -> validate -> use
	userID := 42
	username := "integrationuser"
	role := "admin"
	permissions := []string{"read", "write", "admin"}
	services := []string{"service1", "service2", "service3"}

	// Generate token with session
	token, expiresAt, err := auth.GenerateTokenWithSession(userID, username, role, "session123", permissions, services)
	require.NoError(t, err)
	require.NotEmpty(t, token)
	require.Greater(t, expiresAt, time.Now().Unix())

	// Validate token
	claims, err := auth.ValidateToken(token)
	require.NoError(t, err)
	require.NotNil(t, claims)

	// Test role requirements
	assert.True(t, auth.RequireRole(claims.Role, "user"))
	assert.True(t, auth.RequireRole(claims.Role, "admin"))

	// Test service access
	assert.True(t, auth.HasServiceAccess(claims, "service1"))
	assert.True(t, auth.HasServiceAccess(claims, "nonexistent-service")) // Admin has access to all

	// Test permissions
	assert.True(t, auth.HasPermission(claims, "read"))
	assert.True(t, auth.HasPermission(claims, "nonexistent-permission")) // Admin has all permissions

	// Test session tokens
	sessionToken, sessionHash, err := auth.GenerateSessionToken()
	require.NoError(t, err)
	
	computedHash := auth.HashSessionToken(sessionToken)
	assert.Equal(t, sessionHash, computedHash)
}