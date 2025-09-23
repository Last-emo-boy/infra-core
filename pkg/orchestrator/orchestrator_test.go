package orchestrator

import (
	"encoding/json"
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
	
	orchestrator := New(mockDB, mockConfig)
	
	assert.NotNil(t, orchestrator)
	assert.Equal(t, mockDB, orchestrator.db)
	assert.Equal(t, mockConfig, orchestrator.config)
	assert.NotNil(t, orchestrator.services)
	assert.NotNil(t, orchestrator.deployments)
	assert.NotNil(t, orchestrator.nodes)
	assert.False(t, orchestrator.running)
}

func TestStart(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	orchestrator := New(mockDB, mockConfig)
	
	err := orchestrator.Start()
	assert.NoError(t, err)
	assert.True(t, orchestrator.running)
	
	// Test starting already running orchestrator
	err = orchestrator.Start()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already running")
	
	// Cleanup
	orchestrator.Stop()
}

func TestStop(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	orchestrator := New(mockDB, mockConfig)
	
	// Start first
	err := orchestrator.Start()
	require.NoError(t, err)
	assert.True(t, orchestrator.running)
	
	// Add a service to test stopping
	orchestrator.services["test-service"] = &ServiceInstance{
		ID:     "test-service",
		Name:   "test",
		Status: "running",
	}
	
	// Stop orchestrator
	orchestrator.Stop()
	assert.False(t, orchestrator.running)
	
	// Verify service was stopped
	service := orchestrator.services["test-service"]
	assert.Equal(t, "stopped", service.Status)
}

func TestGetStatus(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	orchestrator := New(mockDB, mockConfig)
	
	// Add test data
	orchestrator.services["service1"] = &ServiceInstance{Status: "running"}
	orchestrator.services["service2"] = &ServiceInstance{Status: "stopped"}
	orchestrator.deployments["deploy1"] = &Deployment{ID: "deploy1"}
	orchestrator.nodes["node1"] = &Node{ID: "node1"}
	
	status := orchestrator.GetStatus()
	
	assert.Equal(t, false, status["running"])
	assert.Equal(t, 2, status["total_services"])
	assert.Equal(t, 1, status["running_services"])
	assert.Equal(t, 1, status["total_deployments"])
	assert.Equal(t, 1, status["nodes"])
	assert.Contains(t, status, "uptime")
}

func TestServiceInstance(t *testing.T) {
	service := &ServiceInstance{
		ID:        "test-service",
		Name:      "test",
		Image:     "nginx:alpine",
		Port:      8080,
		Status:    "running",
		Health:    "healthy",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Environment: map[string]string{
			"ENV": "test",
		},
		Resources: &ResourceRequirements{
			CPU:     "500m",
			Memory:  "512Mi",
			Storage: "1Gi",
		},
	}
	
	assert.Equal(t, "test-service", service.ID)
	assert.Equal(t, "nginx:alpine", service.Image)
	assert.Equal(t, 8080, service.Port)
	assert.Equal(t, "running", service.Status)
	assert.Equal(t, "healthy", service.Health)
	assert.Equal(t, "test", service.Environment["ENV"])
	assert.Equal(t, "500m", service.Resources.CPU)
}

func TestDeployment(t *testing.T) {
	deployment := &Deployment{
		ID:          "deploy-123",
		ServiceName: "test-service",
		Version:     "v1.0.0",
		Status:      "deploying",
		Strategy:    "rolling",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Logs:        []string{"Starting deployment"},
		Config: map[string]interface{}{
			"replicas": 3,
		},
	}
	
	assert.Equal(t, "deploy-123", deployment.ID)
	assert.Equal(t, "test-service", deployment.ServiceName)
	assert.Equal(t, "v1.0.0", deployment.Version)
	assert.Equal(t, "deploying", deployment.Status)
	assert.Equal(t, "rolling", deployment.Strategy)
	assert.Equal(t, 3, deployment.Config["replicas"])
	assert.Len(t, deployment.Logs, 1)
}

func TestNode(t *testing.T) {
	node := &Node{
		ID:     "node-1",
		Name:   "worker-node-1",
		Status: "ready",
		Resources: &NodeResources{
			CPU: ResourceUsage{
				Used:      "2",
				Available: "2",
				Total:     "4",
				Percent:   50.0,
			},
			Memory: ResourceUsage{
				Used:      "4GB",
				Available: "4GB", 
				Total:     "8GB",
				Percent:   50.0,
			},
		},
		Services:      []string{"service1", "service2"},
		LastHeartbeat: time.Now(),
		Metadata: map[string]interface{}{
			"zone": "us-west-1a",
		},
	}
	
	assert.Equal(t, "node-1", node.ID)
	assert.Equal(t, "worker-node-1", node.Name)
	assert.Equal(t, "ready", node.Status)
	assert.Equal(t, 50.0, node.Resources.CPU.Percent)
	assert.Equal(t, "4GB", node.Resources.Memory.Used)
	assert.Len(t, node.Services, 2)
	assert.Equal(t, "us-west-1a", node.Metadata["zone"])
}

func TestResourceRequirements(t *testing.T) {
	resources := &ResourceRequirements{
		CPU:     "1000m",
		Memory:  "1Gi",
		Storage: "10Gi",
	}
	
	assert.Equal(t, "1000m", resources.CPU)
	assert.Equal(t, "1Gi", resources.Memory)
	assert.Equal(t, "10Gi", resources.Storage)
}

func TestResourceUsage(t *testing.T) {
	usage := ResourceUsage{
		Used:      "2GB",
		Available: "6GB",
		Total:     "8GB",
		Percent:   25.0,
	}
	
	assert.Equal(t, "2GB", usage.Used)
	assert.Equal(t, "6GB", usage.Available)
	assert.Equal(t, "8GB", usage.Total)
	assert.Equal(t, 25.0, usage.Percent)
}

func TestDeployRequest(t *testing.T) {
	deployReq := DeployRequest{
		Name:     "my-service",
		Image:    "nginx:1.21",
		Port:     8080,
		Replicas: 3,
		Environment: map[string]string{
			"NODE_ENV": "production",
		},
		Resources: &ResourceRequirements{
			CPU:     "500m",
			Memory:  "512Mi",
			Storage: "1Gi",
		},
		Config: map[string]interface{}{
			"healthCheck": true,
		},
		Strategy: "rolling",
	}
	
	assert.Equal(t, "my-service", deployReq.Name)
	assert.Equal(t, "nginx:1.21", deployReq.Image)
	assert.Equal(t, 8080, deployReq.Port)
	assert.Equal(t, 3, deployReq.Replicas)
	assert.Equal(t, "production", deployReq.Environment["NODE_ENV"])
	assert.Equal(t, "500m", deployReq.Resources.CPU)
	assert.Equal(t, true, deployReq.Config["healthCheck"])
	assert.Equal(t, "rolling", deployReq.Strategy)
}

func TestInitializeNodes(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	orchestrator := New(mockDB, mockConfig)
	
	err := orchestrator.initializeNodes()
	assert.NoError(t, err)
	
	// Check that localhost node was initialized
	node, exists := orchestrator.nodes["localhost"]
	assert.True(t, exists)
	assert.Equal(t, "localhost", node.ID)
	assert.Equal(t, "localhost", node.Name)
	assert.Equal(t, "ready", node.Status)
	assert.NotNil(t, node.Resources)
}

func TestPerformHealthChecks(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	orchestrator := New(mockDB, mockConfig)
	
	// Add a test service
	orchestrator.services["test-service"] = &ServiceInstance{
		ID:     "test-service",
		Name:   "test",
		Status: "running",
		Port:   8080,
		Health: "unknown",
	}
	
	// Perform health checks
	orchestrator.performHealthChecks()
	
	// Verify health status was updated
	service := orchestrator.services["test-service"]
	assert.Contains(t, []string{"healthy", "unhealthy"}, service.Health)
}

func TestUpdateResourceUsage(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	orchestrator := New(mockDB, mockConfig)
	
	// Initialize nodes
	err := orchestrator.initializeNodes()
	require.NoError(t, err)
	
	// Add running services
	orchestrator.services["service1"] = &ServiceInstance{Status: "running"}
	orchestrator.services["service2"] = &ServiceInstance{Status: "running"}
	orchestrator.services["service3"] = &ServiceInstance{Status: "stopped"}
	
	orchestrator.updateResourceUsage()
	
	// Check node resource updates
	node := orchestrator.nodes["localhost"]
	assert.Equal(t, "2", node.Resources.Pods.Used)
	assert.Equal(t, 2.0, node.Resources.Pods.Percent)
}

func TestPerformCleanup(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	orchestrator := New(mockDB, mockConfig)
	
	// Add old stopped service
	oldService := &ServiceInstance{
		ID:        "old-service",
		Status:    "stopped",
		UpdatedAt: time.Now().Add(-2 * time.Hour),
	}
	orchestrator.services["old-service"] = oldService
	
	// Add recent stopped service
	recentService := &ServiceInstance{
		ID:        "recent-service",
		Status:    "stopped",
		UpdatedAt: time.Now().Add(-30 * time.Minute),
	}
	orchestrator.services["recent-service"] = recentService
	
	orchestrator.performCleanup()
	
	// Old service should be cleaned up
	_, exists := orchestrator.services["old-service"]
	assert.False(t, exists)
	
	// Recent service should remain
	_, exists = orchestrator.services["recent-service"]
	assert.True(t, exists)
}

func TestStopServiceInstance(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	orchestrator := New(mockDB, mockConfig)
	
	service := &ServiceInstance{
		ID:     "test-service",
		Name:   "test",
		Status: "running",
	}
	
	err := orchestrator.stopServiceInstance(service)
	assert.NoError(t, err)
	assert.Equal(t, "stopped", service.Status)
}

func setupTestRouter(orchestrator *Orchestrator) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	
	// Add routes
	r.POST("/deploy", orchestrator.DeployService)
	r.POST("/services/:id/start", orchestrator.StartService)
	r.POST("/services/:id/stop", orchestrator.StopService)
	r.POST("/services/:id/restart", orchestrator.RestartService)
	r.DELETE("/services/:id", orchestrator.RemoveService)
	r.GET("/services/:id", orchestrator.GetServiceStatus)
	r.GET("/services/:id/logs", orchestrator.GetServiceLogs)
	r.GET("/deployments", orchestrator.ListDeployments)
	r.GET("/deployments/:id", orchestrator.GetDeployment)
	r.POST("/deployments/:id/rollback", orchestrator.RollbackDeployment)
	r.DELETE("/deployments/:id", orchestrator.DeleteDeployment)
	r.GET("/nodes", orchestrator.ListNodes)
	r.GET("/cluster/resources", orchestrator.GetClusterResources)
	r.GET("/cluster/events", orchestrator.GetClusterEvents)
	r.POST("/sync", orchestrator.SyncServices)
	r.POST("/cleanup", orchestrator.CleanupResources)
	r.GET("/metrics", orchestrator.GetMetrics)
	
	return r
}

func TestDeployServiceHandler(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	orchestrator := New(mockDB, mockConfig)
	r := setupTestRouter(orchestrator)
	
	deployReq := DeployRequest{
		Name:     "test-service",
		Image:    "nginx:alpine",
		Port:     8080,
		Replicas: 2,
		Environment: map[string]string{
			"ENV": "test",
		},
		Strategy: "rolling",
	}
	
	reqBody, err := json.Marshal(deployReq)
	require.NoError(t, err)
	
	req, err := http.NewRequest("POST", "/deploy", strings.NewReader(string(reqBody)))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusCreated, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Contains(t, response, "deployment_id")
	assert.Contains(t, response, "services")
	assert.Equal(t, "deploying", response["status"])
}

func TestStartServiceHandler(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	orchestrator := New(mockDB, mockConfig)
	r := setupTestRouter(orchestrator)
	
	// Add a stopped service
	orchestrator.services["test-service"] = &ServiceInstance{
		ID:     "test-service",
		Status: "stopped",
	}
	
	req, err := http.NewRequest("POST", "/services/test-service/start", nil)
	require.NoError(t, err)
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "test-service", response["service_id"])
	assert.Equal(t, "starting", response["status"])
}

func TestStopServiceHandler(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	orchestrator := New(mockDB, mockConfig)
	r := setupTestRouter(orchestrator)
	
	// Add a running service
	orchestrator.services["test-service"] = &ServiceInstance{
		ID:     "test-service",
		Status: "running",
	}
	
	req, err := http.NewRequest("POST", "/services/test-service/stop", nil)
	require.NoError(t, err)
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "test-service", response["service_id"])
	assert.Equal(t, "stopped", response["status"])
}

func TestGetServiceStatusHandler(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	orchestrator := New(mockDB, mockConfig)
	r := setupTestRouter(orchestrator)
	
	// Add a test service
	testService := &ServiceInstance{
		ID:     "test-service",
		Name:   "test",
		Status: "running",
		Health: "healthy",
	}
	orchestrator.services["test-service"] = testService
	
	req, err := http.NewRequest("GET", "/services/test-service", nil)
	require.NoError(t, err)
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response ServiceInstance
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "test-service", response.ID)
	assert.Equal(t, "test", response.Name)
	assert.Equal(t, "running", response.Status)
	assert.Equal(t, "healthy", response.Health)
}

func TestGetServiceLogsHandler(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	orchestrator := New(mockDB, mockConfig)
	r := setupTestRouter(orchestrator)
	
	// Add a test service
	orchestrator.services["test-service"] = &ServiceInstance{
		ID:   "test-service",
		Name: "test",
	}
	
	req, err := http.NewRequest("GET", "/services/test-service/logs", nil)
	require.NoError(t, err)
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, "test-service", response["service_id"])
	assert.Contains(t, response, "logs")
}

func TestListDeploymentsHandler(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	orchestrator := New(mockDB, mockConfig)
	r := setupTestRouter(orchestrator)
	
	// Add test deployments
	orchestrator.deployments["deploy1"] = &Deployment{ID: "deploy1", ServiceName: "service1"}
	orchestrator.deployments["deploy2"] = &Deployment{ID: "deploy2", ServiceName: "service2"}
	
	req, err := http.NewRequest("GET", "/deployments", nil)
	require.NoError(t, err)
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Equal(t, float64(2), response["total"])
	assert.Contains(t, response, "deployments")
}

func TestGetMetricsHandler(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	orchestrator := New(mockDB, mockConfig)
	r := setupTestRouter(orchestrator)
	
	// Add test data
	orchestrator.services["service1"] = &ServiceInstance{Status: "running"}
	orchestrator.services["service2"] = &ServiceInstance{Status: "stopped"}
	orchestrator.deployments["deploy1"] = &Deployment{Status: "deploying"}
	orchestrator.nodes["node1"] = &Node{ID: "node1"}
	
	req, err := http.NewRequest("GET", "/metrics", nil)
	require.NoError(t, err)
	
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)
	
	assert.Contains(t, response, "metrics")
	assert.Contains(t, response, "timestamp")
	
	metrics := response["metrics"].(map[string]interface{})
	services := metrics["services"].(map[string]interface{})
	assert.Equal(t, float64(2), services["total"])
	assert.Equal(t, float64(1), services["running"])
	assert.Equal(t, float64(1), services["stopped"])
}

func TestHandlerErrorCases(t *testing.T) {
	mockDB := &database.DB{}
	mockConfig := &config.Config{}
	orchestrator := New(mockDB, mockConfig)
	r := setupTestRouter(orchestrator)
	
	tests := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
	}{
		{
			name:           "service not found",
			method:         "GET",
			path:           "/services/nonexistent",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "deployment not found",
			method:         "GET", 
			path:           "/deployments/nonexistent",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "start already running service",
			method:         "POST",
			path:           "/services/running-service/start",
			expectedStatus: http.StatusBadRequest,
		},
	}
	
	// Add a running service for the start test
	orchestrator.services["running-service"] = &ServiceInstance{
		ID:     "running-service",
		Status: "running",
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequest(tt.method, tt.path, nil)
			require.NoError(t, err)
			
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			
			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}