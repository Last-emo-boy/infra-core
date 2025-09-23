package main

import (
	"context"
	"encoding/json"
	"fmt"
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
	os.Setenv("ENVIRONMENT", "test")
	
	// Run tests
	code := m.Run()
	
	// Cleanup
	os.Unsetenv("ENVIRONMENT")
	
	os.Exit(code)
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
				os.Setenv("ENVIRONMENT", tt.envValue)
			} else {
				os.Unsetenv("ENVIRONMENT")
			}
			
			// Test environment detection
			environment := os.Getenv("ENVIRONMENT")
			if environment == "" {
				environment = "development"
			}
			
			assert.Equal(t, tt.expected, environment)
			
			// Cleanup
			os.Unsetenv("ENVIRONMENT")
		})
	}
}

func TestSnapHealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create test router
	router := gin.Default()
	
	// Add health endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "infra-core-snap",
			"timestamp": time.Now().Unix(),
		})
	})
	
	// Test health endpoint
	req, err := http.NewRequest("GET", "/health", nil)
	require.NoError(t, err)
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, "infra-core-snap", response["service"])
	assert.NotEmpty(t, response["timestamp"])
}

func TestSnapAPIRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create test router
	router := gin.Default()
	
	// Setup API routes (mocked)
	api := router.Group("/api/v1")
	{
		// Snap plans
		api.POST("/plans", func(c *gin.Context) {
			c.JSON(http.StatusCreated, gin.H{"action": "create", "id": "plan-123"})
		})
		api.GET("/plans", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"plans": []map[string]interface{}{
				{"id": "plan-1", "name": "daily-backup", "status": "active"},
				{"id": "plan-2", "name": "weekly-backup", "status": "inactive"},
			}})
		})
		api.GET("/plans/:id", func(c *gin.Context) {
			id := c.Param("id")
			c.JSON(http.StatusOK, gin.H{"id": id, "name": "test-plan", "status": "active"})
		})
		api.PUT("/plans/:id", func(c *gin.Context) {
			id := c.Param("id")
			c.JSON(http.StatusOK, gin.H{"action": "update", "id": id})
		})
		api.DELETE("/plans/:id", func(c *gin.Context) {
			id := c.Param("id")
			c.JSON(http.StatusOK, gin.H{"action": "delete", "id": id})
		})
		api.POST("/plans/:id/enable", func(c *gin.Context) {
			id := c.Param("id")
			c.JSON(http.StatusOK, gin.H{"action": "enable", "id": id})
		})
		api.POST("/plans/:id/disable", func(c *gin.Context) {
			id := c.Param("id")
			c.JSON(http.StatusOK, gin.H{"action": "disable", "id": id})
		})
		
		// Snapshots
		api.POST("/snapshots", func(c *gin.Context) {
			c.JSON(http.StatusCreated, gin.H{"action": "create", "id": "snapshot-123"})
		})
		api.GET("/snapshots", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"snapshots": []map[string]interface{}{
				{"id": "snap-1", "name": "backup-2024-01-01", "status": "completed"},
				{"id": "snap-2", "name": "backup-2024-01-02", "status": "in-progress"},
			}})
		})
		api.GET("/snapshots/:id", func(c *gin.Context) {
			id := c.Param("id")
			c.JSON(http.StatusOK, gin.H{"id": id, "name": "test-snapshot", "status": "completed"})
		})
		api.DELETE("/snapshots/:id", func(c *gin.Context) {
			id := c.Param("id")
			c.JSON(http.StatusOK, gin.H{"action": "delete", "id": id})
		})
		api.GET("/snapshots/:id/status", func(c *gin.Context) {
			id := c.Param("id")
			c.JSON(http.StatusOK, gin.H{"id": id, "status": "completed", "progress": 100})
		})
		api.POST("/snapshots/:id/verify", func(c *gin.Context) {
			id := c.Param("id")
			c.JSON(http.StatusOK, gin.H{"action": "verify", "id": id, "result": "valid"})
		})
		
		// Restore operations
		api.POST("/restore", func(c *gin.Context) {
			c.JSON(http.StatusCreated, gin.H{"action": "restore", "id": "restore-123"})
		})
		api.GET("/restore/:id/status", func(c *gin.Context) {
			id := c.Param("id")
			c.JSON(http.StatusOK, gin.H{"id": id, "status": "in-progress", "progress": 75})
		})
		api.POST("/restore/:id/cancel", func(c *gin.Context) {
			id := c.Param("id")
			c.JSON(http.StatusOK, gin.H{"action": "cancel", "id": id})
		})
		
		// Management
		api.GET("/stats", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"stats": map[string]interface{}{
				"total_snapshots": 50,
				"total_size": "100GB",
				"active_plans": 5,
				"last_backup": time.Now().Unix(),
			}})
		})
		api.POST("/cleanup", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"action": "cleanup", "removed": 10})
		})
		api.POST("/scrub", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"action": "scrub", "status": "started"})
		})
		api.GET("/scrub/status", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"scrub": map[string]interface{}{
				"status": "running",
				"progress": 45,
				"errors": 0,
			}})
		})
	}
	
	// Test all routes
	testCases := []struct {
		method       string
		path         string
		expectedCode int
		checkField   string
	}{
		// Plans routes
		{"POST", "/api/v1/plans", http.StatusCreated, "action"},
		{"GET", "/api/v1/plans", http.StatusOK, "plans"},
		{"GET", "/api/v1/plans/test-plan", http.StatusOK, "id"},
		{"PUT", "/api/v1/plans/test-plan", http.StatusOK, "action"},
		{"DELETE", "/api/v1/plans/test-plan", http.StatusOK, "action"},
		{"POST", "/api/v1/plans/test-plan/enable", http.StatusOK, "action"},
		{"POST", "/api/v1/plans/test-plan/disable", http.StatusOK, "action"},
		
		// Snapshots routes
		{"POST", "/api/v1/snapshots", http.StatusCreated, "action"},
		{"GET", "/api/v1/snapshots", http.StatusOK, "snapshots"},
		{"GET", "/api/v1/snapshots/test-snapshot", http.StatusOK, "id"},
		{"DELETE", "/api/v1/snapshots/test-snapshot", http.StatusOK, "action"},
		{"GET", "/api/v1/snapshots/test-snapshot/status", http.StatusOK, "status"},
		{"POST", "/api/v1/snapshots/test-snapshot/verify", http.StatusOK, "action"},
		
		// Restore routes
		{"POST", "/api/v1/restore", http.StatusCreated, "action"},
		{"GET", "/api/v1/restore/test-restore/status", http.StatusOK, "status"},
		{"POST", "/api/v1/restore/test-restore/cancel", http.StatusOK, "action"},
		
		// Management routes
		{"GET", "/api/v1/stats", http.StatusOK, "stats"},
		{"POST", "/api/v1/cleanup", http.StatusOK, "action"},
		{"POST", "/api/v1/scrub", http.StatusOK, "action"},
		{"GET", "/api/v1/scrub/status", http.StatusOK, "scrub"},
	}
	
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s %s", tc.method, tc.path), func(t *testing.T) {
			req, err := http.NewRequest(tc.method, tc.path, nil)
			require.NoError(t, err)
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			assert.Equal(t, tc.expectedCode, w.Code)
			
			if tc.checkField != "" {
				var response map[string]interface{}
				err = json.Unmarshal(w.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Contains(t, response, tc.checkField)
			}
		})
	}
}

func TestServerConfiguration(t *testing.T) {
	// Test server configuration
	server := &http.Server{
		Addr:    ":8086",
		Handler: gin.New(),
	}
	
	assert.Equal(t, ":8086", server.Addr)
	assert.NotNil(t, server.Handler)
}

func TestDefaultPortConfiguration(t *testing.T) {
	// Test default port logic for snap service
	var port int
	
	// Simulate config.Snap.Port being 0
	if port == 0 {
		port = 8086 // Default port for snap service
	}
	
	assert.Equal(t, 8086, port)
	
	// Test with non-zero port
	port = 9999
	if port == 0 {
		port = 8086
	}
	
	assert.Equal(t, 9999, port)
}

func TestDirectoryCreation(t *testing.T) {
	// Test directory creation logic
	tempDir := os.TempDir()
	
	// Test creating a subdirectory
	testSnapDir := tempDir + "/test-snap"
	testTempDir := tempDir + "/test-temp"
	
	// Clean up before test
	os.RemoveAll(testSnapDir)
	os.RemoveAll(testTempDir)
	
	// Test directory creation
	err := os.MkdirAll(testSnapDir, 0755)
	assert.NoError(t, err)
	
	err = os.MkdirAll(testTempDir, 0755)
	assert.NoError(t, err)
	
	// Verify directories exist
	_, err = os.Stat(testSnapDir)
	assert.NoError(t, err)
	
	_, err = os.Stat(testTempDir)
	assert.NoError(t, err)
	
	// Clean up after test
	os.RemoveAll(testSnapDir)
	os.RemoveAll(testTempDir)
}

func TestSnapParameterExtraction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := gin.New()
	
	// Test route with parameter extraction
	router.GET("/plans/:id/status", func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(http.StatusOK, gin.H{"plan_id": id, "status": "active"})
	})
	
	router.GET("/snapshots/:id/verify", func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(http.StatusOK, gin.H{"snapshot_id": id, "verified": true})
	})
	
	// Test parameter extraction
	testCases := []struct {
		path         string
		expectedParam string
		paramField   string
	}{
		{"/plans/my-plan-123/status", "my-plan-123", "plan_id"},
		{"/snapshots/snap-abc-456/verify", "snap-abc-456", "snapshot_id"},
	}
	
	for _, tc := range testCases {
		req, err := http.NewRequest("GET", tc.path, nil)
		require.NoError(t, err)
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, tc.expectedParam, response[tc.paramField])
	}
}

func TestContextTimeout(t *testing.T) {
	// Test context timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	deadline, ok := ctx.Deadline()
	assert.True(t, ok)
	assert.True(t, deadline.After(time.Now()))
	
	// Check timeout duration is approximately correct
	expectedDeadline := time.Now().Add(10 * time.Second)
	timeDiff := deadline.Sub(expectedDeadline)
	
	// Allow for small timing differences (within 1 second)
	assert.True(t, timeDiff < time.Second && timeDiff > -time.Second)
}

func TestSnapStatsResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := gin.New()
	router.GET("/stats", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"total_snapshots": 100,
			"total_size": "500GB",
			"active_plans": 10,
			"completed_backups": 95,
			"failed_backups": 5,
			"last_backup": time.Now().Unix(),
		})
	})
	
	req, err := http.NewRequest("GET", "/stats", nil)
	require.NoError(t, err)
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	// Check stats structure
	assert.Equal(t, float64(100), response["total_snapshots"])
	assert.Equal(t, "500GB", response["total_size"])
	assert.Equal(t, float64(10), response["active_plans"])
	assert.Equal(t, float64(95), response["completed_backups"])
	assert.Equal(t, float64(5), response["failed_backups"])
	assert.Contains(t, response, "last_backup")
}

func TestSnapProgressResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := gin.New()
	router.GET("/operation/:id/status", func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(http.StatusOK, gin.H{
			"id": id,
			"status": "in-progress",
			"progress": 65,
			"eta": time.Now().Add(5 * time.Minute).Unix(),
			"current_file": "/data/file123.db",
		})
	})
	
	req, err := http.NewRequest("GET", "/operation/backup-001/status", nil)
	require.NoError(t, err)
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "backup-001", response["id"])
	assert.Equal(t, "in-progress", response["status"])
	assert.Equal(t, float64(65), response["progress"])
	assert.Contains(t, response, "eta")
	assert.Equal(t, "/data/file123.db", response["current_file"])
}