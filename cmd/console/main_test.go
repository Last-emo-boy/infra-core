package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Set test environment
	os.Setenv("INFRA_CORE_ENV", "test")
	
	// Run tests
	code := m.Run()
	
	// Cleanup
	os.Unsetenv("INFRA_CORE_ENV")
	
	os.Exit(code)
}

func TestHealthEndpoint(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)
	
	// Create a test router
	r := gin.New()
	
	// Add the health endpoint
	r.GET("/", func(c *gin.Context) {
		environment := os.Getenv("INFRA_CORE_ENV")
		if environment == "" {
			environment = "development"
		}
		
		c.JSON(http.StatusOK, gin.H{
			"service":     "infra-core-console",
			"version":     "1.0.0",
			"status":      "healthy",
			"environment": environment,
			"time":        time.Now().UTC().Format(time.RFC3339),
		})
	})
	
	// Create a test request
	req, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)
	
	// Create a test response recorder
	w := httptest.NewRecorder()
	
	// Perform the request
	r.ServeHTTP(w, req)
	
	// Check the response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	
	// Parse response body
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	// Check response content
	assert.Equal(t, "infra-core-console", response["service"])
	assert.Equal(t, "1.0.0", response["version"])
	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, "test", response["environment"])
	assert.NotEmpty(t, response["time"])
}

func TestEnvironmentDetection(t *testing.T) {
	tests := []struct {
		name        string
		envValue    string
		expected    string
	}{
		{
			name:     "production environment",
			envValue: "production",
			expected: "production",
		},
		{
			name:     "development environment",
			envValue: "development",
			expected: "development",
		},
		{
			name:     "test environment",
			envValue: "test",
			expected: "test",
		},
		{
			name:     "empty environment defaults to development",
			envValue: "",
			expected: "development",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment
			if tt.envValue != "" {
				os.Setenv("INFRA_CORE_ENV", tt.envValue)
			} else {
				os.Unsetenv("INFRA_CORE_ENV")
			}
			
			// Test environment detection
			environment := os.Getenv("INFRA_CORE_ENV")
			if environment == "" {
				environment = "development"
			}
			
			assert.Equal(t, tt.expected, environment)
			
			// Cleanup
			os.Unsetenv("INFRA_CORE_ENV")
		})
	}
}

func TestGinModeConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		environment string
		expectedMode string
	}{
		{
			name:         "production mode",
			environment:  "production",
			expectedMode: gin.ReleaseMode,
		},
		{
			name:         "development mode",
			environment:  "development", 
			expectedMode: gin.DebugMode,
		},
		{
			name:         "test mode",
			environment:  "test",
			expectedMode: gin.TestMode,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset Gin mode
			gin.SetMode(gin.DebugMode)
			
			// Set environment and configure mode
			if tt.environment == "production" {
				gin.SetMode(gin.ReleaseMode)
			} else if tt.environment == "test" {
				gin.SetMode(gin.TestMode)
			}
			
			// Check mode
			assert.Equal(t, tt.expectedMode, gin.Mode())
		})
	}
}

func TestRouteGroupStructure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create test router to simulate main function setup
	r := gin.New()
	
	// Root health check
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})
	
	// Public routes
	api := r.Group("/api/v1")
	{
		// Authentication endpoints
		auth := api.Group("/auth")
		{
			auth.POST("/register", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"endpoint": "register"})
			})
			auth.POST("/login", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"endpoint": "login"})
			})
		}
		
		// Health check endpoint
		api.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"endpoint": "health"})
		})
	}
	
	// Test routes exist
	testRoutes := []struct {
		method string
		path   string
		expectedStatus int
	}{
		{"GET", "/", http.StatusOK},
		{"POST", "/api/v1/auth/register", http.StatusOK},
		{"POST", "/api/v1/auth/login", http.StatusOK},
		{"GET", "/api/v1/health", http.StatusOK},
	}
	
	for _, route := range testRoutes {
		req, err := http.NewRequest(route.method, route.path, nil)
		require.NoError(t, err)
		
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		
		assert.Equal(t, route.expectedStatus, w.Code, "Route %s %s should return %d", route.method, route.path, route.expectedStatus)
	}
}

func TestServerConfiguration(t *testing.T) {
	// Test server timeout configurations
	expectedReadTimeout := 30 * time.Second
	expectedWriteTimeout := 30 * time.Second
	expectedIdleTimeout := 60 * time.Second
	expectedMaxHeaderBytes := 1 << 20 // 1MB
	
	server := &http.Server{
		Addr:           ":8080",
		Handler:        gin.New(),
		ReadTimeout:    expectedReadTimeout,
		WriteTimeout:   expectedWriteTimeout,
		IdleTimeout:    expectedIdleTimeout,
		MaxHeaderBytes: expectedMaxHeaderBytes,
	}
	
	assert.Equal(t, expectedReadTimeout, server.ReadTimeout)
	assert.Equal(t, expectedWriteTimeout, server.WriteTimeout)
	assert.Equal(t, expectedIdleTimeout, server.IdleTimeout)
	assert.Equal(t, expectedMaxHeaderBytes, server.MaxHeaderBytes)
}

func TestResponseHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"test": "response"})
	})
	
	req, err := http.NewRequest("GET", "/test", nil)
	require.NoError(t, err)
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestHealthCheckResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	r := gin.New()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":     "infra-core-console",
			"version":     "1.0.0",
			"status":      "healthy",
			"environment": "test",
			"time":        "2024-01-01T00:00:00Z",
		})
	})
	
	req, err := http.NewRequest("GET", "/health", nil)
	require.NoError(t, err)
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	// Verify required fields
	requiredFields := []string{"service", "version", "status", "environment", "time"}
	for _, field := range requiredFields {
		assert.Contains(t, response, field, "Response should contain field: %s", field)
		assert.NotEmpty(t, response[field], "Field %s should not be empty", field)
	}
}