package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/last-emo-boy/infra-core/pkg/auth"
	"github.com/last-emo-boy/infra-core/pkg/config"
	"github.com/last-emo-boy/infra-core/pkg/database"
)

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create mock auth service with basic configuration
	authConfig := &config.ConsoleConfig{
		Auth: config.AuthConfig{
			JWT: config.JWTConfig{
				Secret:       "test-secret-key-for-testing",
				ExpiresHours: 24,
			},
		},
	}
	
	mockAuth, err := auth.NewAuth(authConfig)
	require.NoError(t, err)
	
	// Create mock database
	mockDB := &database.DB{}
	
	// Create a valid test user and token
	testUser := &database.User{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
		Role:     "user",
	}
	
	validToken, _, err := mockAuth.GenerateToken(testUser.ID, testUser.Username, testUser.Role)
	require.NoError(t, err)
	
	tests := []struct {
		name         string
		token        string
		authHeader   string
		expectedCode int
		expectedKeys []string
	}{
		{
			name:         "missing token",
			token:        "",
			authHeader:   "",
			expectedCode: http.StatusUnauthorized,
			expectedKeys: []string{},
		},
		{
			name:         "bearer token present",
			token:        validToken,
			authHeader:   "Bearer " + validToken,
			expectedCode: http.StatusOK,
			expectedKeys: []string{"user_id", "username", "role"},
		},
		{
			name:         "query token present",
			token:        validToken,
			authHeader:   "",
			expectedCode: http.StatusOK,
			expectedKeys: []string{"user_id", "username", "role"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(AuthMiddleware(mockAuth, mockDB))
			r.GET("/protected", func(c *gin.Context) {
				// Check for user context
				if userID, exists := c.Get("user_id"); exists {
					c.JSON(http.StatusOK, gin.H{"user_id": userID})
				} else {
					c.JSON(http.StatusOK, gin.H{"message": "authenticated"})
				}
			})
			
			req, err := http.NewRequest("GET", "/protected", nil)
			require.NoError(t, err)
			
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			if tt.token != "" && tt.authHeader == "" {
				req.URL.RawQuery = "token=" + tt.token
			}
			
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedCode, w.Code)
			
			if tt.expectedCode == http.StatusUnauthorized {
				assert.Contains(t, w.Body.String(), "error")
			}
		})
	}
}

func TestRequireRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create mock auth service with basic configuration
	authConfig := &config.ConsoleConfig{
		Auth: config.AuthConfig{
			JWT: config.JWTConfig{
				Secret:       "test-secret-key-for-testing",
				ExpiresHours: 24,
			},
		},
	}
	
	mockAuth, err := auth.NewAuth(authConfig)
	require.NoError(t, err)
	
	tests := []struct {
		name         string
		userRole     string
		requiredRole string
		expectedCode int
		setContext   bool
	}{
		{
			name:         "admin role sufficient for user access",
			userRole:     "admin",
			requiredRole: "user",
			expectedCode: http.StatusOK,
			setContext:   true,
		},
		{
			name:         "user role insufficient for admin access",
			userRole:     "user",
			requiredRole: "admin",
			expectedCode: http.StatusForbidden,
			setContext:   true,
		},
		{
			name:         "missing role context",
			userRole:     "",
			requiredRole: "user",
			expectedCode: http.StatusUnauthorized,
			setContext:   false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(func(c *gin.Context) {
				if tt.setContext && tt.userRole != "" {
					c.Set("role", tt.userRole)
				}
				c.Next()
			})
			r.Use(RequireRole(mockAuth, tt.requiredRole))
			r.GET("/admin", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "admin access granted"})
			})
			
			req, err := http.NewRequest("GET", "/admin", nil)
			require.NoError(t, err)
			
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedCode, w.Code)
			
			if tt.expectedCode != http.StatusOK {
				assert.Contains(t, w.Body.String(), "error")
			}
		})
	}
}

func TestSSOAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create mock auth service with basic configuration
	authConfig := &config.ConsoleConfig{
		Auth: config.AuthConfig{
			JWT: config.JWTConfig{
				Secret:       "test-secret-key-for-testing",
				ExpiresHours: 24,
			},
		},
	}
	
	mockAuth, err := auth.NewAuth(authConfig)
	require.NoError(t, err)
	
	mockDB := &database.DB{}
	
	// Create a simple test that doesn't rely on database operations
	tests := []struct {
		name         string
		token        string
		expectedCode int
	}{
		{
			name:         "missing SSO token",
			token:        "",
			expectedCode: http.StatusTemporaryRedirect,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(SSOAuthMiddleware(mockAuth, mockDB))
			r.GET("/sso", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "SSO authenticated"})
			})
			
			req, err := http.NewRequest("GET", "/sso", nil)
			require.NoError(t, err)
			
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}
			
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

func TestSSOServiceAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Skip database-dependent tests for now
	t.Skip("Skipping SSO service middleware test due to database mock limitations")
}

func TestSSOTokenValidationMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create mock auth service with basic configuration
	authConfig := &config.ConsoleConfig{
		Auth: config.AuthConfig{
			JWT: config.JWTConfig{
				Secret:       "test-secret-key-for-testing",
				ExpiresHours: 24,
			},
		},
	}
	
	mockAuth, err := auth.NewAuth(authConfig)
	require.NoError(t, err)
	
	// Generate a valid SSO token for testing
	validSSOToken, _, err := mockAuth.GenerateSSOToken(1, "testuser", "user", "session123", "source-service", "target-service", "http://redirect.com", []string{"read"}, []string{"test-service"})
	require.NoError(t, err)
	
	tests := []struct {
		name         string
		ssoToken     string
		headerToken  string
		expectedCode int
	}{
		{
			name:         "missing SSO token",
			ssoToken:     "",
			headerToken:  "",
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "valid SSO token in query",
			ssoToken:     validSSOToken,
			headerToken:  "",
			expectedCode: http.StatusOK,
		},
		{
			name:         "valid SSO token in header",
			ssoToken:     "",
			headerToken:  validSSOToken,
			expectedCode: http.StatusOK,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.Use(SSOTokenValidationMiddleware(mockAuth))
			r.GET("/validate", func(c *gin.Context) {
				claims, _ := c.Get("sso_claims")
				c.JSON(http.StatusOK, gin.H{"claims": claims != nil})
			})
			
			req, err := http.NewRequest("GET", "/validate", nil)
			require.NoError(t, err)
			
			if tt.ssoToken != "" {
				req.URL.RawQuery = "sso_token=" + tt.ssoToken
			}
			if tt.headerToken != "" {
				req.Header.Set("X-SSO-Token", tt.headerToken)
			}
			
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

func TestRequireServicePermission(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Skip database-dependent tests for now
	t.Skip("Skipping service permission middleware test due to database mock limitations")
}

func TestExtractToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name           string
		authHeader     string
		queryToken     string
		ssoToken       string
		cookie         string
		expectedToken  string
	}{
		{
			name:           "bearer token in header",
			authHeader:     "Bearer test-token",
			expectedToken:  "test-token",
		},
		{
			name:           "query parameter token",
			queryToken:     "query-token",
			expectedToken:  "query-token",
		},
		{
			name:           "SSO token parameter",
			ssoToken:       "sso-token",
			expectedToken:  "sso-token",
		},
		{
			name:           "invalid auth header",
			authHeader:     "Invalid format",
			expectedToken:  "",
		},
		{
			name:           "no token provided",
			expectedToken:  "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/test", func(c *gin.Context) {
				token := extractToken(c)
				c.JSON(http.StatusOK, gin.H{"token": token})
			})
			
			req, err := http.NewRequest("GET", "/test", nil)
			require.NoError(t, err)
			
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			
			query := req.URL.Query()
			if tt.queryToken != "" {
				query.Add("token", tt.queryToken)
			}
			if tt.ssoToken != "" {
				query.Add("sso_token", tt.ssoToken)
			}
			req.URL.RawQuery = query.Encode()
			
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{
					Name:  "auth_token",
					Value: tt.cookie,
				})
			}
			
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			
			assert.Equal(t, http.StatusOK, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedToken)
		})
	}
}

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	r := gin.New()
	r.Use(CORSMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "test"})
	})
	r.OPTIONS("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	
	tests := []struct {
		name           string
		method         string
		expectedCode   int
		checkHeaders   bool
	}{
		{
			name:           "GET request with CORS headers",
			method:         "GET",
			expectedCode:   http.StatusOK,
			checkHeaders:   true,
		},
		{
			name:           "OPTIONS preflight request",
			method:         "OPTIONS",
			expectedCode:   http.StatusNoContent,
			checkHeaders:   true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, "/test", nil)
			require.NoError(t, err)
			
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedCode, w.Code)
			
			if tt.checkHeaders {
				assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
				assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
				assert.Contains(t, w.Header().Get("Access-Control-Allow-Headers"), "Authorization")
				assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "GET")
			}
		})
	}
}

func TestLoggingMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Test that logging middleware can be created
	middleware := LoggingMiddleware()
	assert.NotNil(t, middleware)
	
	r := gin.New()
	r.Use(middleware)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "logged"})
	})
	
	req, err := http.NewRequest("GET", "/test", nil)
	require.NoError(t, err)
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRecoveryMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Test that recovery middleware can be created
	middleware := RecoveryMiddleware()
	assert.NotNil(t, middleware)
	
	r := gin.New()
	r.Use(middleware)
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "recovered"})
	})
	
	req, err := http.NewRequest("GET", "/test", nil)
	require.NoError(t, err)
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRedirectToSSOLogin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name         string
		path         string
		expectedCode int
		checkJSON    bool
	}{
		{
			name:         "API request returns JSON",
			path:         "/api/test",
			expectedCode: http.StatusUnauthorized,
			checkJSON:    true,
		},
		{
			name:         "Web request redirects",
			path:         "/web/test",
			expectedCode: http.StatusTemporaryRedirect,
			checkJSON:    false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/*path", func(c *gin.Context) {
				redirectToSSOLogin(c)
			})
			
			req, err := http.NewRequest("GET", tt.path, nil)
			require.NoError(t, err)
			
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedCode, w.Code)
			
			if tt.checkJSON {
				body := w.Body.String()
				assert.Contains(t, body, "Authentication required")
				assert.Contains(t, body, "login_url")
			} else {
				location := w.Header().Get("Location")
				assert.Contains(t, location, "/login")
			}
		})
	}
}

func TestMiddlewareIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create mock auth service with basic configuration
	authConfig := &config.ConsoleConfig{
		Auth: config.AuthConfig{
			JWT: config.JWTConfig{
				Secret:       "test-secret-key-for-testing",
				ExpiresHours: 24,
			},
		},
	}
	
	mockAuth, err := auth.NewAuth(authConfig)
	require.NoError(t, err)
	
	mockDB := &database.DB{}
	
	// Test middleware chain
	r := gin.New()
	r.Use(CORSMiddleware())
	r.Use(LoggingMiddleware())
	r.Use(RecoveryMiddleware())
	r.Use(AuthMiddleware(mockAuth, mockDB))
	r.Use(RequireRole(mockAuth, "user"))
	
	r.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "fully protected"})
	})
	
	req, err := http.NewRequest("GET", "/protected", nil)
	require.NoError(t, err)
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	// Should be unauthorized due to missing auth
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	
	// Check CORS headers are still present
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}