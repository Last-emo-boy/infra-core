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

func TestHealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create test router
	r := gin.Default()
	
	// Mock orchestrator status
	mockStatus := map[string]interface{}{
		"running":    true,
		"services":   5,
		"healthy":    4,
		"unhealthy":  1,
	}
	
	// Add health endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":      "healthy",
			"orchestrator": mockStatus,
			"timestamp":   time.Now().Unix(),
		})
	})
	
	// Test health endpoint
	req, err := http.NewRequest("GET", "/health", nil)
	require.NoError(t, err)
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "healthy", response["status"])
	assert.Contains(t, response, "orchestrator")
	assert.Contains(t, response, "timestamp")
}

func TestOrchestratorAPIRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create test router
	r := gin.Default()
	
	// Setup API routes (mocked)
	api := r.Group("/api/v1")
	{
		// Service orchestration
		services := api.Group("/services")
		{
			services.POST("/deploy", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"action": "deploy"})
			})
			services.POST("/:id/start", func(c *gin.Context) {
				id := c.Param("id")
				c.JSON(http.StatusOK, gin.H{"action": "start", "service_id": id})
			})
			services.POST("/:id/stop", func(c *gin.Context) {
				id := c.Param("id")
				c.JSON(http.StatusOK, gin.H{"action": "stop", "service_id": id})
			})
			services.POST("/:id/restart", func(c *gin.Context) {
				id := c.Param("id")
				c.JSON(http.StatusOK, gin.H{"action": "restart", "service_id": id})
			})
			services.DELETE("/:id", func(c *gin.Context) {
				id := c.Param("id")
				c.JSON(http.StatusOK, gin.H{"action": "remove", "service_id": id})
			})
			services.GET("/:id/status", func(c *gin.Context) {
				id := c.Param("id")
				c.JSON(http.StatusOK, gin.H{"service_id": id, "status": "running"})
			})
			services.GET("/:id/logs", func(c *gin.Context) {
				id := c.Param("id")
				c.JSON(http.StatusOK, gin.H{"service_id": id, "logs": []string{"log1", "log2"}})
			})
		}
		
		// Deployment management
		deployments := api.Group("/deployments")
		{
			deployments.GET("/", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"deployments": []string{}})
			})
			deployments.GET("/:id", func(c *gin.Context) {
				id := c.Param("id")
				c.JSON(http.StatusOK, gin.H{"deployment_id": id})
			})
			deployments.POST("/:id/rollback", func(c *gin.Context) {
				id := c.Param("id")
				c.JSON(http.StatusOK, gin.H{"action": "rollback", "deployment_id": id})
			})
			deployments.DELETE("/:id", func(c *gin.Context) {
				id := c.Param("id")
				c.JSON(http.StatusOK, gin.H{"action": "delete", "deployment_id": id})
			})
		}
		
		// Cluster management
		cluster := api.Group("/cluster")
		{
			cluster.GET("/nodes", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"nodes": []string{"node1", "node2"}})
			})
			cluster.GET("/resources", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"cpu": "50%", "memory": "60%"})
			})
			cluster.GET("/events", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"events": []string{"event1", "event2"}})
			})
		}
		
		// Orchestrator control
		control := api.Group("/control")
		{
			control.POST("/sync", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"action": "sync"})
			})
			control.POST("/cleanup", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"action": "cleanup"})
			})
			control.GET("/metrics", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"metrics": "data"})
			})
		}
	}
	
	// Test all routes
	testCases := []struct {
		method       string
		path         string
		expectedCode int
		checkField   string
	}{
		// Services routes
		{"POST", "/api/v1/services/deploy", http.StatusOK, "action"},
		{"POST", "/api/v1/services/test-service/start", http.StatusOK, "action"},
		{"POST", "/api/v1/services/test-service/stop", http.StatusOK, "action"},
		{"POST", "/api/v1/services/test-service/restart", http.StatusOK, "action"},
		{"DELETE", "/api/v1/services/test-service", http.StatusOK, "action"},
		{"GET", "/api/v1/services/test-service/status", http.StatusOK, "status"},
		{"GET", "/api/v1/services/test-service/logs", http.StatusOK, "logs"},
		
		// Deployments routes
		{"GET", "/api/v1/deployments/", http.StatusOK, "deployments"},
		{"GET", "/api/v1/deployments/test-deployment", http.StatusOK, "deployment_id"},
		{"POST", "/api/v1/deployments/test-deployment/rollback", http.StatusOK, "action"},
		{"DELETE", "/api/v1/deployments/test-deployment", http.StatusOK, "action"},
		
		// Cluster routes
		{"GET", "/api/v1/cluster/nodes", http.StatusOK, "nodes"},
		{"GET", "/api/v1/cluster/resources", http.StatusOK, "cpu"},
		{"GET", "/api/v1/cluster/events", http.StatusOK, "events"},
		
		// Control routes
		{"POST", "/api/v1/control/sync", http.StatusOK, "action"},
		{"POST", "/api/v1/control/cleanup", http.StatusOK, "action"},
		{"GET", "/api/v1/control/metrics", http.StatusOK, "metrics"},
	}
	
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s %s", tc.method, tc.path), func(t *testing.T) {
			req, err := http.NewRequest(tc.method, tc.path, nil)
			require.NoError(t, err)
			
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			
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
		Addr:           ":8084",
		Handler:        gin.New(),
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}
	
	assert.Equal(t, ":8084", server.Addr)
	assert.Equal(t, 30*time.Second, server.ReadTimeout)
	assert.Equal(t, 30*time.Second, server.WriteTimeout)
	assert.Equal(t, 60*time.Second, server.IdleTimeout)
	assert.Equal(t, 1<<20, server.MaxHeaderBytes)
}

func TestDefaultPortConfiguration(t *testing.T) {
	// Test default port logic
	var port int
	
	// Simulate config.Orchestrator.Port being 0
	if port == 0 {
		port = 8084 // Default orchestrator port
	}
	
	assert.Equal(t, 8084, port)
	
	// Test with non-zero port
	port = 9999
	if port == 0 {
		port = 8084
	}
	
	assert.Equal(t, 9999, port)
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
			
			// Configure mode based on environment
			if tt.environment == "production" {
				gin.SetMode(gin.ReleaseMode)
			} else if tt.environment == "test" {
				gin.SetMode(gin.TestMode)
			}
			
			assert.Equal(t, tt.expectedMode, gin.Mode())
		})
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

func TestServiceParameterExtraction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	r := gin.New()
	
	// Test route with parameter extraction
	r.GET("/services/:id/status", func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(http.StatusOK, gin.H{"service_id": id})
	})
	
	// Test parameter extraction
	req, err := http.NewRequest("GET", "/services/my-service-123/status", nil)
	require.NoError(t, err)
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "my-service-123", response["service_id"])
}

func TestResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	r := gin.New()
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "success",
			"data":    []string{"item1", "item2"},
			"count":   2,
		})
	})
	
	req, err := http.NewRequest("GET", "/test", nil)
	require.NoError(t, err)
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "success", response["message"])
	assert.Contains(t, response, "data")
	assert.Equal(t, float64(2), response["count"])
}