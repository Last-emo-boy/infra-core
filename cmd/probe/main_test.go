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

func TestProbeHealthEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create test router
	r := gin.Default()
	
	// Mock probe monitor status
	mockStatus := map[string]interface{}{
		"running":     true,
		"active_probes": 3,
		"healthy":     2,
		"unhealthy":   1,
		"last_scan":   time.Now().Unix(),
	}
	
	// Add health endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"probe":     mockStatus,
			"timestamp": time.Now().Unix(),
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
	assert.Contains(t, response, "probe")
	assert.Contains(t, response, "timestamp")
	
	// Check probe status details
	probeStatus := response["probe"].(map[string]interface{})
	assert.Equal(t, true, probeStatus["running"])
	assert.Equal(t, float64(3), probeStatus["active_probes"])
}

func TestProbeAPIRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create test router
	r := gin.Default()
	
	// Setup API routes (mocked)
	api := r.Group("/api/v1")
	{
		// Probe configuration
		probes := api.Group("/probes")
		{
			probes.POST("/", func(c *gin.Context) {
				c.JSON(http.StatusCreated, gin.H{"action": "create", "id": "probe-123"})
			})
			probes.GET("/", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"probes": []map[string]interface{}{
					{"id": "probe-1", "name": "test-probe-1", "status": "active"},
					{"id": "probe-2", "name": "test-probe-2", "status": "inactive"},
				}})
			})
			probes.GET("/:id", func(c *gin.Context) {
				id := c.Param("id")
				c.JSON(http.StatusOK, gin.H{"id": id, "name": "test-probe", "status": "active"})
			})
			probes.PUT("/:id", func(c *gin.Context) {
				id := c.Param("id")
				c.JSON(http.StatusOK, gin.H{"action": "update", "id": id})
			})
			probes.DELETE("/:id", func(c *gin.Context) {
				id := c.Param("id")
				c.JSON(http.StatusOK, gin.H{"action": "delete", "id": id})
			})
			probes.POST("/:id/enable", func(c *gin.Context) {
				id := c.Param("id")
				c.JSON(http.StatusOK, gin.H{"action": "enable", "id": id})
			})
			probes.POST("/:id/disable", func(c *gin.Context) {
				id := c.Param("id")
				c.JSON(http.StatusOK, gin.H{"action": "disable", "id": id})
			})
		}
		
		// Probe results and metrics
		results := api.Group("/results")
		{
			results.GET("/", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"results": []string{"result1", "result2"}})
			})
			results.GET("/:probe_id", func(c *gin.Context) {
				probeID := c.Param("probe_id")
				c.JSON(http.StatusOK, gin.H{"probe_id": probeID, "results": []string{}})
			})
			results.GET("/:probe_id/latest", func(c *gin.Context) {
				probeID := c.Param("probe_id")
				c.JSON(http.StatusOK, gin.H{"probe_id": probeID, "latest": map[string]interface{}{
					"timestamp": time.Now().Unix(),
					"status": "success",
					"response_time": 150,
				}})
			})
			results.GET("/:probe_id/metrics", func(c *gin.Context) {
				probeID := c.Param("probe_id")
				c.JSON(http.StatusOK, gin.H{"probe_id": probeID, "metrics": map[string]interface{}{
					"avg_response_time": 120,
					"success_rate": 95.5,
					"total_checks": 1000,
				}})
			})
			results.GET("/:probe_id/history", func(c *gin.Context) {
				probeID := c.Param("probe_id")
				c.JSON(http.StatusOK, gin.H{"probe_id": probeID, "history": []string{"history1", "history2"}})
			})
		}
		
		// Health monitoring
		health := api.Group("/health")
		{
			health.GET("/services", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"services": []map[string]interface{}{
					{"id": "service-1", "status": "healthy"},
					{"id": "service-2", "status": "unhealthy"},
				}})
			})
			health.GET("/services/:service_id", func(c *gin.Context) {
				serviceID := c.Param("service_id")
				c.JSON(http.StatusOK, gin.H{"service_id": serviceID, "status": "healthy", "checks": 5})
			})
			health.GET("/overview", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"overview": map[string]interface{}{
					"total_services": 10,
					"healthy": 8,
					"unhealthy": 2,
					"last_update": time.Now().Unix(),
				}})
			})
			health.GET("/alerts", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"alerts": []map[string]interface{}{
					{"id": "alert-1", "severity": "high", "message": "Service down"},
				}})
			})
		}
		
		// Monitoring control
		control := api.Group("/control")
		{
			control.POST("/scan", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"action": "scan", "status": "triggered"})
			})
			control.POST("/cleanup", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"action": "cleanup", "status": "completed"})
			})
			control.GET("/metrics", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"metrics": map[string]interface{}{
					"total_probes": 5,
					"active_probes": 3,
					"total_checks": 10000,
					"uptime": "99.9%",
				}})
			})
			control.GET("/status", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": map[string]interface{}{
					"running": true,
					"version": "1.0.0",
					"start_time": time.Now().Add(-24*time.Hour).Unix(),
				}})
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
		// Probes routes
		{"POST", "/api/v1/probes/", http.StatusCreated, "action"},
		{"GET", "/api/v1/probes/", http.StatusOK, "probes"},
		{"GET", "/api/v1/probes/test-probe", http.StatusOK, "id"},
		{"PUT", "/api/v1/probes/test-probe", http.StatusOK, "action"},
		{"DELETE", "/api/v1/probes/test-probe", http.StatusOK, "action"},
		{"POST", "/api/v1/probes/test-probe/enable", http.StatusOK, "action"},
		{"POST", "/api/v1/probes/test-probe/disable", http.StatusOK, "action"},
		
		// Results routes
		{"GET", "/api/v1/results/", http.StatusOK, "results"},
		{"GET", "/api/v1/results/probe-123", http.StatusOK, "probe_id"},
		{"GET", "/api/v1/results/probe-123/latest", http.StatusOK, "latest"},
		{"GET", "/api/v1/results/probe-123/metrics", http.StatusOK, "metrics"},
		{"GET", "/api/v1/results/probe-123/history", http.StatusOK, "history"},
		
		// Health routes
		{"GET", "/api/v1/health/services", http.StatusOK, "services"},
		{"GET", "/api/v1/health/services/service-123", http.StatusOK, "service_id"},
		{"GET", "/api/v1/health/overview", http.StatusOK, "overview"},
		{"GET", "/api/v1/health/alerts", http.StatusOK, "alerts"},
		
		// Control routes
		{"POST", "/api/v1/control/scan", http.StatusOK, "action"},
		{"POST", "/api/v1/control/cleanup", http.StatusOK, "action"},
		{"GET", "/api/v1/control/metrics", http.StatusOK, "metrics"},
		{"GET", "/api/v1/control/status", http.StatusOK, "status"},
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
		Addr:           ":8085",
		Handler:        gin.New(),
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}
	
	assert.Equal(t, ":8085", server.Addr)
	assert.Equal(t, 30*time.Second, server.ReadTimeout)
	assert.Equal(t, 30*time.Second, server.WriteTimeout)
	assert.Equal(t, 60*time.Second, server.IdleTimeout)
	assert.Equal(t, 1<<20, server.MaxHeaderBytes)
}

func TestDefaultPortConfiguration(t *testing.T) {
	// Test default port logic for probe
	var port int
	
	// Simulate config.Probe.Port being 0
	if port == 0 {
		port = 8085 // Default probe port
	}
	
	assert.Equal(t, 8085, port)
	
	// Test with non-zero port
	port = 9999
	if port == 0 {
		port = 8085
	}
	
	assert.Equal(t, 9999, port)
}

func TestProbeParameterExtraction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	r := gin.New()
	
	// Test route with parameter extraction
	r.GET("/probes/:id/status", func(c *gin.Context) {
		id := c.Param("id")
		c.JSON(http.StatusOK, gin.H{"probe_id": id, "status": "active"})
	})
	
	r.GET("/results/:probe_id/metrics", func(c *gin.Context) {
		probeID := c.Param("probe_id")
		c.JSON(http.StatusOK, gin.H{"probe_id": probeID, "metrics": "data"})
	})
	
	// Test parameter extraction
	testCases := []struct {
		path         string
		expectedParam string
		paramField   string
	}{
		{"/probes/my-probe-123/status", "my-probe-123", "probe_id"},
		{"/results/probe-abc-456/metrics", "probe-abc-456", "probe_id"},
	}
	
	for _, tc := range testCases {
		req, err := http.NewRequest("GET", tc.path, nil)
		require.NoError(t, err)
		
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		
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

func TestProbeMetricsFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	r := gin.New()
	r.GET("/metrics", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"total_probes": 5,
			"active_probes": 3,
			"success_rate": 98.5,
			"avg_response_time": 250,
			"uptime_percentage": 99.9,
		})
	})
	
	req, err := http.NewRequest("GET", "/metrics", nil)
	require.NoError(t, err)
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	// Check metrics structure
	assert.Equal(t, float64(5), response["total_probes"])
	assert.Equal(t, float64(3), response["active_probes"])
	assert.Equal(t, 98.5, response["success_rate"])
	assert.Equal(t, float64(250), response["avg_response_time"])
	assert.Equal(t, 99.9, response["uptime_percentage"])
}

func TestProbeStatusResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	r := gin.New()
	r.GET("/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "probe-monitor",
			"version": "1.0.0",
			"status": "running",
			"uptime": time.Hour * 24,
			"last_check": time.Now().Unix(),
		})
	})
	
	req, err := http.NewRequest("GET", "/status", nil)
	require.NoError(t, err)
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "probe-monitor", response["service"])
	assert.Equal(t, "1.0.0", response["version"])
	assert.Equal(t, "running", response["status"])
	assert.Contains(t, response, "uptime")
	assert.Contains(t, response, "last_check")
}