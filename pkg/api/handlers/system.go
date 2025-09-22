package handlers

import (
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/last-emo-boy/infra-core/pkg/database"
)

// SystemHandler handles system-related API endpoints
type SystemHandler struct {
	db        *database.DB
	startTime time.Time
}

// NewSystemHandler creates a new SystemHandler
func NewSystemHandler(db *database.DB) *SystemHandler {
	return &SystemHandler{
		db:        db,
		startTime: time.Now(),
	}
}

// HealthCheck returns the health status of the system
func (h *SystemHandler) HealthCheck(c *gin.Context) {
	// Check database connectivity
	if err := h.db.HealthCheck(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":    "unhealthy",
			"database":  "disconnected",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"database":  "connected",
		"uptime":    time.Since(h.startTime).String(),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// GetSystemInfo returns detailed system information
func (h *SystemHandler) GetSystemInfo(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Get database statistics
	stats, err := h.db.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get database stats"})
		return
	}

	systemInfo := gin.H{
		"uptime":     time.Since(h.startTime).String(),
		"version":    "1.0.0",
		"go_version": runtime.Version(),
		"platform":   runtime.GOOS + "/" + runtime.GOARCH,
		"memory": gin.H{
			"alloc_bytes":       m.Alloc,
			"total_alloc_bytes": m.TotalAlloc,
			"sys_bytes":         m.Sys,
			"gc_runs":           m.NumGC,
		},
		"goroutines": runtime.NumGoroutine(),
		"database":   stats,
		"timestamp":  time.Now().UTC().Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, systemInfo)
}

// GetMetrics returns system metrics
func (h *SystemHandler) GetMetrics(c *gin.Context) {
	// Get query parameters
	service := c.Query("service")
	limit := c.DefaultQuery("limit", "100")

	repo := h.db.MetricRepository()

	var metrics []*database.Metric
	var err error

	if service != "" {
		// Get metrics for specific service
		metrics, err = repo.GetByService(service, 100) // TODO: Use limit parameter
	} else {
		// Get recent metrics
		metrics, err = repo.GetRecent(100) // TODO: Use limit parameter
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch metrics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"metrics": metrics,
		"total":   len(metrics),
		"service": service,
		"limit":   limit,
	})
}

// GetAuditLogs returns audit logs
func (h *SystemHandler) GetAuditLogs(c *gin.Context) {
	// Get query parameters
	userID := c.Query("user_id")
	action := c.Query("action")
	limit := c.DefaultQuery("limit", "50")

	// TODO: Implement audit log filtering
	auditLogs := []gin.H{
		{
			"id":         1,
			"user_id":    1,
			"username":   "admin",
			"action":     "service.create",
			"resource":   "hello-service",
			"details":    gin.H{"image": "nginx:latest"},
			"timestamp":  time.Now().Add(-1 * time.Hour).UTC().Format(time.RFC3339),
			"ip_address": "127.0.0.1",
		},
		{
			"id":         2,
			"user_id":    1,
			"username":   "admin",
			"action":     "user.login",
			"resource":   "",
			"details":    gin.H{},
			"timestamp":  time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339),
			"ip_address": "127.0.0.1",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"audit_logs": auditLogs,
		"total":      len(auditLogs),
		"filters": gin.H{
			"user_id": userID,
			"action":  action,
			"limit":   limit,
		},
	})
}

// GetDashboardData returns data for the dashboard
func (h *SystemHandler) GetDashboardData(c *gin.Context) {
	// Get service counts
	serviceRepo := h.db.ServiceRepository()
	services, err := serviceRepo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch services"})
		return
	}

	serviceCounts := map[string]int{
		"total":   len(services),
		"running": 0,
		"stopped": 0,
		"error":   0,
	}

	for _, service := range services {
		serviceCounts[service.Status]++
	}

	// Get user count
	userRepo := h.db.UserRepository()
	users, err := userRepo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}

	// Get recent metrics
	metricRepo := h.db.MetricRepository()
	recentMetrics, err := metricRepo.GetRecent(10)
	if err != nil {
		recentMetrics = []*database.Metric{} // Empty slice on error
	}

	dashboardData := gin.H{
		"services": serviceCounts,
		"users": gin.H{
			"total": len(users),
		},
		"system": gin.H{
			"uptime":     time.Since(h.startTime).String(),
			"goroutines": runtime.NumGoroutine(),
		},
		"recent_metrics": recentMetrics,
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
	}

	c.JSON(http.StatusOK, dashboardData)
}
