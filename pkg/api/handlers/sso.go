package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/last-emo-boy/infra-core/pkg/auth"
	"github.com/last-emo-boy/infra-core/pkg/database"
)

// SSOHandler handles SSO-related API endpoints
type SSOHandler struct {
	auth *auth.Auth
	db   *database.DB
}

// NewSSOHandler creates a new SSOHandler
func NewSSOHandler(auth *auth.Auth, db *database.DB) *SSOHandler {
	return &SSOHandler{
		auth: auth,
		db:   db,
	}
}

// RegisterServiceRequest represents service registration data
type RegisterServiceRequest struct {
	Name         string  `json:"name" binding:"required"`
	DisplayName  string  `json:"display_name" binding:"required"`
	Description  *string `json:"description"`
	ServiceURL   string  `json:"service_url" binding:"required,url"`
	CallbackURL  *string `json:"callback_url"`
	Icon         *string `json:"icon"`
	Category     string  `json:"category" binding:"required"`
	IsPublic     bool    `json:"is_public"`
	RequiredRole string  `json:"required_role" binding:"required"`
	HealthURL    *string `json:"health_url"`
}

// ServiceResponse represents service response data
type ServiceResponse struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	DisplayName  string     `json:"display_name"`
	Description  *string    `json:"description"`
	ServiceURL   string     `json:"service_url"`
	CallbackURL  *string    `json:"callback_url"`
	Icon         *string    `json:"icon"`
	Category     string     `json:"category"`
	IsPublic     bool       `json:"is_public"`
	RequiredRole string     `json:"required_role"`
	Status       string     `json:"status"`
	HealthURL    *string    `json:"health_url"`
	LastHealthy  *time.Time `json:"last_healthy"`
	IsHealthy    bool       `json:"is_healthy"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// SSOLoginRequest represents SSO login request
type SSOLoginRequest struct {
	ServiceName string `json:"service_name" binding:"required"`
	RedirectURL string `json:"redirect_url" binding:"required,url"`
}

// SSOLoginResponse represents SSO login response
type SSOLoginResponse struct {
	SSOToken    string `json:"sso_token"`
	RedirectURL string `json:"redirect_url"`
	ExpiresAt   int64  `json:"expires_at"`
}

// RegisterService registers a new service with the SSO gateway
func (h *SSOHandler) RegisterService(c *gin.Context) {
	var req RegisterServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create service record
	service := &database.RegisteredService{
		ID:           uuid.New().String(),
		Name:         req.Name,
		DisplayName:  req.DisplayName,
		Description:  req.Description,
		ServiceURL:   req.ServiceURL,
		CallbackURL:  req.CallbackURL,
		Icon:         req.Icon,
		Category:     req.Category,
		IsPublic:     req.IsPublic,
		RequiredRole: req.RequiredRole,
		Status:       "active",
		HealthURL:    req.HealthURL,
	}

	repo := h.db.RegisteredServiceRepository()
	if err := repo.Create(service); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register service"})
		return
	}

	response := h.convertToServiceResponse(service, false)
	c.JSON(http.StatusCreated, response)
}

// ListServices lists all registered services
func (h *SSOHandler) ListServices(c *gin.Context) {
	repo := h.db.RegisteredServiceRepository()
	services, err := repo.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list services"})
		return
	}

	var responses []ServiceResponse
	for _, service := range services {
		isHealthy := h.checkServiceHealth(service.ID)
		response := h.convertToServiceResponse(service, isHealthy)
		responses = append(responses, response)
	}

	c.JSON(http.StatusOK, gin.H{
		"services": responses,
		"count":    len(responses),
	})
}

// ListUserServices lists services accessible to the current user
func (h *SSOHandler) ListUserServices(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	userRole, _ := c.Get("role")
	
	repo := h.db.UserServicePermissionRepository()
	services, err := repo.ListUserServices(userID.(int))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list user services"})
		return
	}

	var responses []ServiceResponse
	for _, service := range services {
		// Check role requirements
		if !h.auth.RequireRole(userRole.(string), service.RequiredRole) && !service.IsPublic {
			continue
		}

		isHealthy := h.checkServiceHealth(service.ID)
		response := h.convertToServiceResponse(service, isHealthy)
		responses = append(responses, response)
	}

	c.JSON(http.StatusOK, gin.H{
		"services": responses,
		"count":    len(responses),
	})
}

// GetService gets a specific service by ID
func (h *SSOHandler) GetService(c *gin.Context) {
	serviceID := c.Param("id")
	
	repo := h.db.RegisteredServiceRepository()
	service, err := repo.GetByID(serviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	isHealthy := h.checkServiceHealth(service.ID)
	response := h.convertToServiceResponse(service, isHealthy)
	c.JSON(http.StatusOK, response)
}

// UpdateService updates a registered service
func (h *SSOHandler) UpdateService(c *gin.Context) {
	serviceID := c.Param("id")
	
	var req RegisterServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	repo := h.db.RegisteredServiceRepository()
	service, err := repo.GetByID(serviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	// Update service fields
	service.DisplayName = req.DisplayName
	service.Description = req.Description
	service.ServiceURL = req.ServiceURL
	service.CallbackURL = req.CallbackURL
	service.Icon = req.Icon
	service.Category = req.Category
	service.IsPublic = req.IsPublic
	service.RequiredRole = req.RequiredRole
	service.HealthURL = req.HealthURL

	if err := repo.Update(service); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update service"})
		return
	}

	isHealthy := h.checkServiceHealth(service.ID)
	response := h.convertToServiceResponse(service, isHealthy)
	c.JSON(http.StatusOK, response)
}

// DeleteService deletes a registered service
func (h *SSOHandler) DeleteService(c *gin.Context) {
	serviceID := c.Param("id")
	
	repo := h.db.RegisteredServiceRepository()
	if err := repo.Delete(serviceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete service"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Service deleted successfully"})
}

// InitiateSSO initiates SSO login flow for a service
func (h *SSOHandler) InitiateSSO(c *gin.Context) {
	var req SSOLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user information from context
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	username, _ := c.Get("username")
	role, _ := c.Get("role")
	sessionID, _ := c.Get("session_id")

	// Check if service exists and user has access
	serviceRepo := h.db.RegisteredServiceRepository()
	service, err := serviceRepo.GetByName(req.ServiceName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	// Check role requirements
	if !h.auth.RequireRole(role.(string), service.RequiredRole) && !service.IsPublic {
		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions for this service"})
		return
	}

	// Check explicit service permissions
	permRepo := h.db.UserServicePermissionRepository()
	hasPermission, err := permRepo.CheckPermission(userID.(int), service.ID)
	if err == nil && !hasPermission && !service.IsPublic {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied to this service"})
		return
	}

	// Get user services and permissions
	userServices, _ := permRepo.ListUserServices(userID.(int))
	services := make([]string, len(userServices))
	for i, svc := range userServices {
		services[i] = svc.Name
	}

	// Generate SSO token
	ssoToken, expiresAt, err := h.auth.GenerateSSOToken(
		userID.(int),
		username.(string),
		role.(string),
		sessionID.(string),
		"infra-core",
		service.Name,
		req.RedirectURL,
		[]string{}, // permissions
		services,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate SSO token"})
		return
	}

	// Build redirect URL with token
	redirectURL := fmt.Sprintf("%s?sso_token=%s", req.RedirectURL, ssoToken)

	response := SSOLoginResponse{
		SSOToken:    ssoToken,
		RedirectURL: redirectURL,
		ExpiresAt:   expiresAt,
	}

	c.JSON(http.StatusOK, response)
}

// ValidateSSO validates an SSO token
func (h *SSOHandler) ValidateSSO(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "SSO token required"})
		return
	}

	claims, err := h.auth.ValidateSSOToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid SSO token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"valid":      true,
		"user_id":    claims.UserID,
		"username":   claims.Username,
		"role":       claims.Role,
		"services":   claims.Services,
		"expires_at": claims.ExpiresAt.Unix(),
	})
}

// GrantServiceAccess grants a user access to a service
func (h *SSOHandler) GrantServiceAccess(c *gin.Context) {
	userIDStr := c.Param("user_id")
	serviceID := c.Param("service_id")

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	grantedBy, _ := c.Get("user_id")
	
	repo := h.db.UserServicePermissionRepository()
	if err := repo.Grant(userID, serviceID, grantedBy.(int), nil); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to grant service access"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Service access granted successfully"})
}

// RevokeServiceAccess revokes a user's access to a service
func (h *SSOHandler) RevokeServiceAccess(c *gin.Context) {
	userIDStr := c.Param("user_id")
	serviceID := c.Param("service_id")

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	repo := h.db.UserServicePermissionRepository()
	if err := repo.Revoke(userID, serviceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke service access"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Service access revoked successfully"})
}

// GetServiceHealth gets the health status of a service
func (h *SSOHandler) GetServiceHealth(c *gin.Context) {
	serviceID := c.Param("id")
	
	healthRepo := h.db.ServiceHealthCheckRepository()
	healthCheck, err := healthRepo.GetLatest(serviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No health check data available"})
		return
	}

	c.JSON(http.StatusOK, healthCheck)
}

// GetServiceHealthHistory gets the health check history for a service
func (h *SSOHandler) GetServiceHealthHistory(c *gin.Context) {
	serviceID := c.Param("id")
	limit := 50 // Default limit

	if limitStr := c.Query("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 && parsedLimit <= 1000 {
			limit = parsedLimit
		}
	}

	healthRepo := h.db.ServiceHealthCheckRepository()
	checks, err := healthRepo.GetHistory(serviceID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get health check history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"checks": checks,
		"count":  len(checks),
	})
}

// Helper methods

func (h *SSOHandler) convertToServiceResponse(service *database.RegisteredService, isHealthy bool) ServiceResponse {
	return ServiceResponse{
		ID:           service.ID,
		Name:         service.Name,
		DisplayName:  service.DisplayName,
		Description:  service.Description,
		ServiceURL:   service.ServiceURL,
		CallbackURL:  service.CallbackURL,
		Icon:         service.Icon,
		Category:     service.Category,
		IsPublic:     service.IsPublic,
		RequiredRole: service.RequiredRole,
		Status:       service.Status,
		HealthURL:    service.HealthURL,
		LastHealthy:  service.LastHealthy,
		IsHealthy:    isHealthy,
		CreatedAt:    service.CreatedAt,
		UpdatedAt:    service.UpdatedAt,
	}
}

func (h *SSOHandler) checkServiceHealth(serviceID string) bool {
	healthRepo := h.db.ServiceHealthCheckRepository()
	healthCheck, err := healthRepo.GetLatest(serviceID)
	if err != nil {
		return false
	}

	// Consider healthy if checked within last 5 minutes and was healthy
	if time.Since(healthCheck.CheckedAt) < 5*time.Minute && healthCheck.IsHealthy {
		return true
	}

	return false
}