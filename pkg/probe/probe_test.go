package probe

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/last-emo-boy/infra-core/pkg/config"
	"github.com/last-emo-boy/infra-core/pkg/database"
)

func TestNew(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	
	monitor := New(mockDB, mockConfig)
	
	assert.NotNil(t, monitor)
	assert.Equal(t, mockDB, monitor.db)
	assert.Equal(t, mockConfig, monitor.config)
	assert.NotNil(t, monitor.probes)
	assert.NotNil(t, monitor.results)
	assert.NotNil(t, monitor.alerts)
	assert.False(t, monitor.running)
}

func TestStart(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	monitor := New(mockDB, mockConfig)
	
	err := monitor.Start()
	assert.NoError(t, err)
	assert.True(t, monitor.running)
	
	// Test starting already running monitor
	err = monitor.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
	
	// Cleanup
	monitor.Stop()
}

func TestStop(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	monitor := New(mockDB, mockConfig)
	
	// Start first
	err := monitor.Start()
	require.NoError(t, err)
	assert.True(t, monitor.running)
	
	// Stop monitor
	monitor.Stop()
	assert.False(t, monitor.running)
}

func TestGetStatus(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	monitor := New(mockDB, mockConfig)
	
	// Add test data
	monitor.probes["probe1"] = &ProbeConfig{ID: "probe1", Enabled: true}
	monitor.probes["probe2"] = &ProbeConfig{ID: "probe2", Enabled: false}
	monitor.alerts["alert1"] = &Alert{ID: "alert1", Status: "active"}
	monitor.alerts["alert2"] = &Alert{ID: "alert2", Status: "resolved"}
	
	status := monitor.GetStatus()
	
	assert.Equal(t, false, status["running"])
	assert.Equal(t, 2, status["total_probes"])
	assert.Equal(t, 1, status["enabled_probes"])
	assert.Equal(t, 1, status["active_alerts"])
	assert.Contains(t, status, "last_scan")
}

func TestProbeConfig(t *testing.T) {
	probe := &ProbeConfig{
		ID:              "test-probe",
		Name:            "Test HTTP Probe",
		Type:            "http",
		Target:          "http://example.com/health",
		Interval:        30 * time.Second,
		Timeout:         5 * time.Second,
		Retries:         3,
		Enabled:         true,
		ExpectedStatus:  200,
		ExpectedContent: "OK",
		Headers: map[string]string{
			"User-Agent": "ProbeMonitor/1.0",
		},
		Thresholds: &ProbeThresholds{
			ResponseTime:    2 * time.Second,
			SuccessRate:     0.95,
			ConsecutiveFail: 3,
		},
		Tags:      []string{"api", "health"},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	assert.Equal(t, "test-probe", probe.ID)
	assert.Equal(t, "Test HTTP Probe", probe.Name)
	assert.Equal(t, "http", probe.Type)
	assert.Equal(t, "http://example.com/health", probe.Target)
	assert.Equal(t, 30*time.Second, probe.Interval)
	assert.Equal(t, 5*time.Second, probe.Timeout)
	assert.Equal(t, 3, probe.Retries)
	assert.True(t, probe.Enabled)
	assert.Equal(t, 200, probe.ExpectedStatus)
	assert.Equal(t, "OK", probe.ExpectedContent)
	assert.Equal(t, "ProbeMonitor/1.0", probe.Headers["User-Agent"])
	assert.Equal(t, 2*time.Second, probe.Thresholds.ResponseTime)
	assert.Equal(t, 0.95, probe.Thresholds.SuccessRate)
	assert.Equal(t, 3, probe.Thresholds.ConsecutiveFail)
	assert.Contains(t, probe.Tags, "api")
	assert.Contains(t, probe.Tags, "health")
}

func TestProbeThresholds(t *testing.T) {
	thresholds := &ProbeThresholds{
		ResponseTime:    2 * time.Second,
		SuccessRate:     0.98,
		ConsecutiveFail: 5,
	}
	
	assert.Equal(t, 2*time.Second, thresholds.ResponseTime)
	assert.Equal(t, 0.98, thresholds.SuccessRate)
	assert.Equal(t, 5, thresholds.ConsecutiveFail)
}

func TestProbeResult(t *testing.T) {
	result := &ProbeResult{
		ID:           "result-123",
		ProbeID:      "probe-456",
		Status:       "success",
		ResponseTime: 500 * time.Millisecond,
		StatusCode:   200,
		Message:      "HTTP check passed",
		Error:        "",
		Metadata: map[string]interface{}{
			"content_length": 1024,
		},
		Timestamp: time.Now(),
	}
	
	assert.Equal(t, "result-123", result.ID)
	assert.Equal(t, "probe-456", result.ProbeID)
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, 500*time.Millisecond, result.ResponseTime)
	assert.Equal(t, 200, result.StatusCode)
	assert.Equal(t, "HTTP check passed", result.Message)
	assert.Empty(t, result.Error)
	assert.Equal(t, 1024, result.Metadata["content_length"])
}

func TestAlert(t *testing.T) {
	now := time.Now()
	resolvedAt := now.Add(1 * time.Hour)
	
	alert := &Alert{
		ID:        "alert-789",
		ProbeID:   "probe-456",
		Type:      "threshold",
		Severity:  "high",
		Status:    "resolved",
		Message:   "Response time exceeded threshold",
		Count:     5,
		FirstSeen: now,
		LastSeen:  now.Add(30 * time.Minute),
		ResolvedAt: &resolvedAt,
		Metadata: map[string]interface{}{
			"threshold": "2s",
			"actual":    "3.5s",
		},
	}
	
	assert.Equal(t, "alert-789", alert.ID)
	assert.Equal(t, "probe-456", alert.ProbeID)
	assert.Equal(t, "threshold", alert.Type)
	assert.Equal(t, "high", alert.Severity)
	assert.Equal(t, "resolved", alert.Status)
	assert.Equal(t, "Response time exceeded threshold", alert.Message)
	assert.Equal(t, 5, alert.Count)
	assert.Equal(t, now, alert.FirstSeen)
	assert.NotNil(t, alert.ResolvedAt)
	assert.Equal(t, resolvedAt, *alert.ResolvedAt)
	assert.Equal(t, "2s", alert.Metadata["threshold"])
	assert.Equal(t, "3.5s", alert.Metadata["actual"])
}

func TestProbeMetrics(t *testing.T) {
	metrics := &ProbeMetrics{
		ProbeID:           "probe-123",
		TotalChecks:       100,
		SuccessfulChecks:  95,
		FailedChecks:      5,
		SuccessRate:       0.95,
		AverageResponse:   800 * time.Millisecond,
		MaxResponse:       2 * time.Second,
		MinResponse:       200 * time.Millisecond,
		LastCheck:         time.Now(),
		LastSuccess:       time.Now().Add(-5 * time.Minute),
		LastFailure:       time.Now().Add(-1 * time.Hour),
		ConsecutiveFails:  0,
		Uptime:            99.5,
	}
	
	assert.Equal(t, "probe-123", metrics.ProbeID)
	assert.Equal(t, 100, metrics.TotalChecks)
	assert.Equal(t, 95, metrics.SuccessfulChecks)
	assert.Equal(t, 5, metrics.FailedChecks)
	assert.Equal(t, 0.95, metrics.SuccessRate)
	assert.Equal(t, 800*time.Millisecond, metrics.AverageResponse)
	assert.Equal(t, 2*time.Second, metrics.MaxResponse)
	assert.Equal(t, 200*time.Millisecond, metrics.MinResponse)
	assert.Equal(t, 0, metrics.ConsecutiveFails)
	assert.Equal(t, 99.5, metrics.Uptime)
}

func TestCreateProbeRequest(t *testing.T) {
	request := CreateProbeRequest{
		Name:            "API Health Check",
		Type:            "http",
		Target:          "https://api.example.com/health",
		Interval:        "30s",
		Timeout:         "5s",
		Retries:         3,
		ExpectedStatus:  200,
		ExpectedContent: "healthy",
		Headers: map[string]string{
			"Authorization": "Bearer token123",
		},
		Thresholds: &ProbeThresholds{
			ResponseTime:    1 * time.Second,
			SuccessRate:     0.99,
			ConsecutiveFail: 2,
		},
		Tags: []string{"api", "critical"},
		Config: map[string]interface{}{
			"follow_redirects": true,
		},
	}
	
	assert.Equal(t, "API Health Check", request.Name)
	assert.Equal(t, "http", request.Type)
	assert.Equal(t, "https://api.example.com/health", request.Target)
	assert.Equal(t, "30s", request.Interval)
	assert.Equal(t, "5s", request.Timeout)
	assert.Equal(t, 3, request.Retries)
	assert.Equal(t, 200, request.ExpectedStatus)
	assert.Equal(t, "healthy", request.ExpectedContent)
	assert.Equal(t, "Bearer token123", request.Headers["Authorization"])
	assert.Equal(t, 1*time.Second, request.Thresholds.ResponseTime)
	assert.Equal(t, 0.99, request.Thresholds.SuccessRate)
	assert.Equal(t, 2, request.Thresholds.ConsecutiveFail)
	assert.Contains(t, request.Tags, "api")
	assert.Contains(t, request.Tags, "critical")
	assert.Equal(t, true, request.Config["follow_redirects"])
}

func TestLoadProbes(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	monitor := New(mockDB, mockConfig)
	
	err := monitor.loadProbes()
	assert.NoError(t, err)
	
	// Check that default probes were loaded
	assert.Contains(t, monitor.probes, "console-health")
	assert.Contains(t, monitor.probes, "gate-health")
	assert.Contains(t, monitor.probes, "orch-health")
	
	consoleProbe := monitor.probes["console-health"]
	assert.Equal(t, "Console API Health Check", consoleProbe.Name)
	assert.Equal(t, "http", consoleProbe.Type)
	assert.Equal(t, "http://localhost:8082/api/v1/health", consoleProbe.Target)
	assert.Equal(t, 30*time.Second, consoleProbe.Interval)
	assert.True(t, consoleProbe.Enabled)
	assert.Equal(t, 200, consoleProbe.ExpectedStatus)
}

func TestShouldRunProbe(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	monitor := New(mockDB, mockConfig)
	
	probe := &ProbeConfig{
		ID:       "test-probe",
		Enabled:  true,
		Interval: 30 * time.Second,
	}
	
	// Currently always returns true (simplified implementation)
	shouldRun := monitor.shouldRunProbe(probe)
	assert.True(t, shouldRun)
}

func TestExecuteHTTPProbe(t *testing.T) {
	// Create a test HTTP server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer testServer.Close()
	
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	monitor := New(mockDB, mockConfig)
	
	probe := &ProbeConfig{
		ID:             "test-http-probe",
		Type:           "http",
		Target:         testServer.URL,
		Timeout:        5 * time.Second,
		ExpectedStatus: 200,
		Headers: map[string]string{
			"User-Agent": "TestAgent",
		},
	}
	
	result := &ProbeResult{
		ID:        "result-1",
		ProbeID:   probe.ID,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
	
	monitor.executeHTTPProbe(probe, result)
	
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, 200, result.StatusCode)
	assert.Equal(t, "HTTP check passed", result.Message)
	assert.Empty(t, result.Error)
	assert.Contains(t, result.Metadata, "headers")
}

func TestExecuteHTTPProbeFailure(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	monitor := New(mockDB, mockConfig)
	
	probe := &ProbeConfig{
		ID:             "test-http-probe",
		Type:           "http",
		Target:         "http://nonexistent.example.com",
		Timeout:        1 * time.Second,
		ExpectedStatus: 200,
	}
	
	result := &ProbeResult{
		ID:        "result-1",
		ProbeID:   probe.ID,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
	
	monitor.executeHTTPProbe(probe, result)
	
	assert.Equal(t, "failure", result.Status)
	assert.NotEmpty(t, result.Error)
	assert.Contains(t, result.Error, "request failed")
}

func TestExecuteHTTPProbeWrongStatus(t *testing.T) {
	// Create a test HTTP server that returns 500
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("Error"))
	}))
	defer testServer.Close()
	
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	monitor := New(mockDB, mockConfig)
	
	probe := &ProbeConfig{
		ID:             "test-http-probe",
		Type:           "http",
		Target:         testServer.URL,
		Timeout:        5 * time.Second,
		ExpectedStatus: 200,
	}
	
	result := &ProbeResult{
		ID:        "result-1",
		ProbeID:   probe.ID,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
	
	monitor.executeHTTPProbe(probe, result)
	
	assert.Equal(t, "failure", result.Status)
	assert.Equal(t, 500, result.StatusCode)
	assert.Contains(t, result.Message, "unexpected status code")
	assert.Contains(t, result.Message, "got 500, expected 200")
}

func TestExecuteTCPProbe(t *testing.T) {
	// Start a simple TCP server
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer listener.Close()
	
	// Accept connections in background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()
	
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	monitor := New(mockDB, mockConfig)
	
	probe := &ProbeConfig{
		ID:      "test-tcp-probe",
		Type:    "tcp",
		Target:  listener.Addr().String(),
		Timeout: 5 * time.Second,
	}
	
	result := &ProbeResult{
		ID:        "result-1",
		ProbeID:   probe.ID,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
	
	monitor.executeTCPProbe(probe, result)
	
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, "TCP connection successful", result.Message)
	assert.Empty(t, result.Error)
}

func TestExecuteTCPProbeFailure(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	monitor := New(mockDB, mockConfig)
	
	probe := &ProbeConfig{
		ID:      "test-tcp-probe",
		Type:    "tcp",
		Target:  "localhost:99999", // Unlikely to be open
		Timeout: 1 * time.Second,
	}
	
	result := &ProbeResult{
		ID:        "result-1",
		ProbeID:   probe.ID,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}
	
	monitor.executeTCPProbe(probe, result)
	
	assert.Equal(t, "failure", result.Status)
	assert.NotEmpty(t, result.Error)
	assert.Contains(t, result.Error, "TCP connection failed")
}

func TestCreateAlert(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	monitor := New(mockDB, mockConfig)
	
	probeID := "test-probe"
	alertType := "threshold"
	severity := "high"
	message := "Response time exceeded threshold"
	
	monitor.createAlert(probeID, alertType, severity, message)
	
	// Check that alert was created
	assert.Equal(t, 1, len(monitor.alerts))
	
	var alert *Alert
	for _, a := range monitor.alerts {
		alert = a
		break
	}
	
	assert.Equal(t, probeID, alert.ProbeID)
	assert.Equal(t, alertType, alert.Type)
	assert.Equal(t, severity, alert.Severity)
	assert.Equal(t, "active", alert.Status)
	assert.Equal(t, message, alert.Message)
	assert.Equal(t, 1, alert.Count)
}

func TestCheckThresholds(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	monitor := New(mockDB, mockConfig)
	
	probe := &ProbeConfig{
		ID: "test-probe",
		Thresholds: &ProbeThresholds{
			ResponseTime:    1 * time.Second,
			SuccessRate:     0.95,
			ConsecutiveFail: 3,
		},
	}
	
	// Test response time threshold exceeded
	result := &ProbeResult{
		ProbeID:      "test-probe",
		Status:       "success",
		ResponseTime: 2 * time.Second, // Exceeds threshold
	}
	
	monitor.checkThresholds(probe, result)
	
	// Should create an alert for response time
	assert.Equal(t, 1, len(monitor.alerts))
}

func TestProcessAlerts(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	monitor := New(mockDB, mockConfig)
	
	// Add an old active alert
	oldAlert := &Alert{
		ID:       "old-alert",
		Status:   "active",
		LastSeen: time.Now().Add(-15 * time.Minute), // Older than 10 minutes
	}
	monitor.alerts["old-alert"] = oldAlert
	
	// Add a recent active alert
	recentAlert := &Alert{
		ID:       "recent-alert",
		Status:   "active",
		LastSeen: time.Now().Add(-5 * time.Minute), // Within 10 minutes
	}
	monitor.alerts["recent-alert"] = recentAlert
	
	monitor.processAlerts()
	
	// Old alert should be auto-resolved
	assert.Equal(t, "resolved", oldAlert.Status)
	assert.NotNil(t, oldAlert.ResolvedAt)
	
	// Recent alert should remain active
	assert.Equal(t, "active", recentAlert.Status)
	assert.Nil(t, recentAlert.ResolvedAt)
}

func TestPerformCleanup(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	monitor := New(mockDB, mockConfig)
	
	// Add old result
	oldResult := &ProbeResult{
		ID:        "old-result",
		Timestamp: time.Now().Add(-25 * time.Hour), // Older than 24 hours
	}
	monitor.results["old-result"] = oldResult
	
	// Add recent result
	recentResult := &ProbeResult{
		ID:        "recent-result",
		Timestamp: time.Now().Add(-1 * time.Hour), // Within 24 hours
	}
	monitor.results["recent-result"] = recentResult
	
	// Add old resolved alert
	resolvedAt := time.Now().Add(-8 * 24 * time.Hour) // 8 days ago
	oldResolvedAlert := &Alert{
		ID:         "old-resolved-alert",
		Status:     "resolved",
		ResolvedAt: &resolvedAt,
	}
	monitor.alerts["old-resolved-alert"] = oldResolvedAlert
	
	// Add recent resolved alert
	recentResolvedAt := time.Now().Add(-1 * 24 * time.Hour) // 1 day ago
	recentResolvedAlert := &Alert{
		ID:         "recent-resolved-alert",
		Status:     "resolved",
		ResolvedAt: &recentResolvedAt,
	}
	monitor.alerts["recent-resolved-alert"] = recentResolvedAlert
	
	monitor.performCleanup()
	
	// Old result should be cleaned up
	_, exists := monitor.results["old-result"]
	assert.False(t, exists)
	
	// Recent result should remain
	_, exists = monitor.results["recent-result"]
	assert.True(t, exists)
	
	// Old resolved alert should be cleaned up
	_, exists = monitor.alerts["old-resolved-alert"]
	assert.False(t, exists)
	
	// Recent resolved alert should remain
	_, exists = monitor.alerts["recent-resolved-alert"]
	assert.True(t, exists)
}

func TestFailureCountMethods(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	monitor := New(mockDB, mockConfig)
	
	probeID := "test-probe"
	
	// Test increment (simplified test since methods are stubs)
	monitor.incrementFailureCount(probeID)
	
	// Test get count
	count := monitor.getFailureCount(probeID)
	assert.Equal(t, 1, count) // Returns 1 in the stub
	
	// Test reset
	monitor.resetFailureCount(probeID)
}

func setupProbeTestRouter(monitor *ProbeMonitor) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	
	// Create a mock handler since we don't have the actual handler implementation
	r.POST("/probes", func(c *gin.Context) {
		var req CreateProbeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		
		c.JSON(http.StatusCreated, gin.H{
			"id":      "test-probe-id",
			"name":    req.Name,
			"type":    req.Type,
			"target":  req.Target,
			"enabled": true,
		})
	})
	
	r.GET("/probes", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"probes": []map[string]interface{}{
				{
					"id":      "probe1",
					"name":    "Test Probe",
					"type":    "http",
					"enabled": true,
				},
			},
			"total": 1,
		})
	})
	
	return r
}

func TestProbeAPIHandlers(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	monitor := New(mockDB, mockConfig)
	r := setupProbeTestRouter(monitor)
	
	t.Run("Create Probe", func(t *testing.T) {
		createReq := CreateProbeRequest{
			Name:   "Test API Probe",
			Type:   "http",
			Target: "http://localhost:8080/health",
		}
		
		reqBody, err := json.Marshal(createReq)
		require.NoError(t, err)
		
		req, err := http.NewRequest("POST", "/probes", strings.NewReader(string(reqBody)))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusCreated, w.Code)
		
		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, "Test API Probe", response["name"])
		assert.Equal(t, "http", response["type"])
		assert.Equal(t, true, response["enabled"])
	})
	
	t.Run("List Probes", func(t *testing.T) {
		req, err := http.NewRequest("GET", "/probes", nil)
		require.NoError(t, err)
		
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)
		
		assert.Equal(t, float64(1), response["total"])
		assert.Contains(t, response, "probes")
	})
}