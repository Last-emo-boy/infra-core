package handlers

import (
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/last-emo-boy/infra-core/pkg/database"
)

// ServiceHandler handles service-related API endpoints
type ServiceHandler struct {
	db *database.DB
}

// NewServiceHandler creates a new ServiceHandler
func NewServiceHandler(db *database.DB) *ServiceHandler {
	return &ServiceHandler{db: db}
}

// CreateServiceRequest represents service creation data
type CreateServiceRequest struct {
	Name        string            `json:"name" binding:"required"`
	Image       string            `json:"image" binding:"required"`
	Port        int               `json:"port" binding:"required"`
	Environment map[string]string `json:"environment,omitempty"`
	Command     []string          `json:"command,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Replicas    int               `json:"replicas,omitempty"`
	HealthCheck *struct {
		Path     string `json:"path"`
		Interval int    `json:"interval"`
		Timeout  int    `json:"timeout"`
		Retries  int    `json:"retries"`
	} `json:"health_check,omitempty"`
}

// UpdateServiceRequest represents service update data
type UpdateServiceRequest struct {
	Image       *string           `json:"image,omitempty"`
	Port        *int              `json:"port,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Command     []string          `json:"command,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Replicas    *int              `json:"replicas,omitempty"`
	Status      *string           `json:"status,omitempty"` // running, stopped, error
}

// CreateService creates a new service
func (h *ServiceHandler) CreateService(c *gin.Context) {
	var req CreateServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default values
	if req.Replicas == 0 {
		req.Replicas = 1
	}

	service := &database.Service{
		ID:       uuid.New().String(),
		Name:     req.Name,
		Image:    req.Image,
		Port:     req.Port,
		Replicas: req.Replicas,
		Status:   "stopped",
	}

	// Convert request to service format
	if len(req.Environment) > 0 {
		service.Environment = req.Environment
	}
	if len(req.Command) > 0 {
		service.Command = req.Command
	}
	if len(req.Args) > 0 {
		service.Args = req.Args
	}

	repo := h.db.ServiceRepository()
	if err := repo.Create(service); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create service"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":    "Service created successfully",
		"service_id": service.ID,
		"service":    service,
	})
}

// ListServices returns a list of all services
func (h *ServiceHandler) ListServices(c *gin.Context) {
	repo := h.db.ServiceRepository()
	services, err := repo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch services"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"services": services,
		"total":    len(services),
	})
}

// GetService returns service details by ID
func (h *ServiceHandler) GetService(c *gin.Context) {
	serviceID := c.Param("id")

	repo := h.db.ServiceRepository()
	service, err := repo.GetByID(serviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"service": service})
}

// UpdateService updates service configuration
func (h *ServiceHandler) UpdateService(c *gin.Context) {
	serviceID := c.Param("id")

	var req UpdateServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	repo := h.db.ServiceRepository()
	service, err := repo.GetByID(serviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	// Update fields
	if req.Image != nil {
		service.Image = *req.Image
	}
	if req.Port != nil {
		service.Port = *req.Port
	}
	if req.Replicas != nil {
		service.Replicas = *req.Replicas
	}
	if req.Status != nil {
		service.Status = *req.Status
	}
	if req.Environment != nil {
		service.Environment = req.Environment
	}
	if req.Command != nil {
		service.Command = req.Command
	}
	if req.Args != nil {
		service.Args = req.Args
	}

	if err := repo.Update(service); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update service"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Service updated successfully",
		"service": service,
	})
}

// DeleteService deletes a service
func (h *ServiceHandler) DeleteService(c *gin.Context) {
	serviceID := c.Param("id")

	repo := h.db.ServiceRepository()
	if err := repo.Delete(serviceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete service"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Service deleted successfully"})
}

// StartService starts a service
func (h *ServiceHandler) StartService(c *gin.Context) {
	serviceID := c.Param("id")

	repo := h.db.ServiceRepository()
	service, err := repo.GetByID(serviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	// Update status to running
	service.Status = "running"
	if err := repo.Update(service); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start service"})
		return
	}

	// TODO: Implement actual service orchestration
	c.JSON(http.StatusOK, gin.H{
		"message":    "Service start initiated",
		"service_id": serviceID,
		"status":     "running",
	})
}

// StopService stops a service
func (h *ServiceHandler) StopService(c *gin.Context) {
	serviceID := c.Param("id")

	repo := h.db.ServiceRepository()
	service, err := repo.GetByID(serviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	// Update status to stopped
	service.Status = "stopped"
	if err := repo.Update(service); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop service"})
		return
	}

	// TODO: Implement actual service orchestration
	c.JSON(http.StatusOK, gin.H{
		"message":    "Service stop initiated",
		"service_id": serviceID,
		"status":     "stopped",
	})
}

// GetServiceLogs returns service logs
func (h *ServiceHandler) GetServiceLogs(c *gin.Context) {
	serviceID := c.Param("id")

	// Get query parameters
	tail := c.DefaultQuery("tail", "100")
	since := c.Query("since")

	// TODO: Implement actual log retrieval
	c.JSON(http.StatusOK, gin.H{
		"service_id": serviceID,
		"logs": []string{
			"2025-01-22T21:30:00Z [INFO] Service started",
			"2025-01-22T21:30:01Z [INFO] Listening on port 3000",
			"2025-01-22T21:30:05Z [INFO] Health check passed",
		},
		"tail":  tail,
		"since": since,
	})
}

// ServiceSummaryResponse represents aggregated service information
type ServiceSummaryResponse struct {
	Counts      ServiceStatusCounts `json:"counts"`
	Recent      []*database.Service `json:"recent"`
	LastUpdated *time.Time          `json:"last_updated,omitempty"`
	GeneratedAt time.Time           `json:"generated_at"`
}

// ServiceStatusCounts groups service counts by status
type ServiceStatusCounts struct {
	Total   int `json:"total"`
	Running int `json:"running"`
	Stopped int `json:"stopped"`
	Error   int `json:"error"`
}

// GetServiceSummary returns aggregated information about services
func (h *ServiceHandler) GetServiceSummary(c *gin.Context) {
	repo := h.db.ServiceRepository()
	services, err := repo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch services"})
		return
	}

	counts := ServiceStatusCounts{}
	counts.Total = len(services)

	for _, service := range services {
		switch service.Status {
		case "running":
			counts.Running++
		case "stopped":
			counts.Stopped++
		case "error":
			counts.Error++
		}

		if service.Environment == nil {
			service.Environment = make(map[string]string)
		}
		if service.Command == nil {
			service.Command = []string{}
		}
		if service.Args == nil {
			service.Args = []string{}
		}
	}

	sorted := make([]*database.Service, len(services))
	copy(sorted, services)

	sort.Slice(sorted, func(i, j int) bool {
		left := sorted[i].UpdatedAt
		right := sorted[j].UpdatedAt

		if left.IsZero() {
			left = sorted[i].CreatedAt
		}
		if right.IsZero() {
			right = sorted[j].CreatedAt
		}

		if left.Equal(right) {
			leftCreated := sorted[i].CreatedAt
			rightCreated := sorted[j].CreatedAt

			if leftCreated.Equal(rightCreated) {
				return sorted[i].ID < sorted[j].ID
			}

			return leftCreated.After(rightCreated)
		}

		return left.After(right)
	})

	summary := ServiceSummaryResponse{
		Counts:      counts,
		Recent:      []*database.Service{},
		GeneratedAt: time.Now().UTC(),
	}

	if len(sorted) > 0 {
		limit := 5
		if len(sorted) < limit {
			limit = len(sorted)
		}

		summary.Recent = sorted[:limit]

		mostRecent := sorted[0].UpdatedAt
		if mostRecent.IsZero() {
			mostRecent = sorted[0].CreatedAt
		}

		if !mostRecent.IsZero() {
			lastUpdated := mostRecent
			summary.LastUpdated = &lastUpdated
		}
	}

	c.JSON(http.StatusOK, summary)
}
