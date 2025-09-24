package orchestrator

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DeployService handles service deployment requests
func (o *Orchestrator) DeployService(c *gin.Context) {
	var req DeployRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	o.mutex.Lock()
	defer o.mutex.Unlock()

	// Generate deployment ID
	deploymentID := uuid.New().String()

	// Create deployment record
	deployment := &Deployment{
		ID:          deploymentID,
		ServiceName: req.Name,
		Version:     "latest", // Could be extracted from image tag
		Status:      "deploying",
		Strategy:    req.Strategy,
		Config:      req.Config,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Logs:        []string{},
	}

	o.deployments[deploymentID] = deployment

	// Create service instances
	replicas := req.Replicas
	if replicas <= 0 {
		replicas = 1
	}

	var createdServices []string
	for i := 0; i < replicas; i++ {
		serviceID := fmt.Sprintf("%s-%d", req.Name, i)
		port := req.Port
		if port > 0 && i > 0 {
			port += i // Avoid port conflicts
		}

		service := &ServiceInstance{
			ID:          serviceID,
			Name:        req.Name,
			Image:       req.Image,
			Port:        port,
			Status:      "starting",
			Health:      "unknown",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Environment: req.Environment,
			Resources:   req.Resources,
			Config:      req.Config,
		}

		o.services[serviceID] = service
		createdServices = append(createdServices, serviceID)

		// Simulate deployment process
		go o.deployServiceInstance(service, deployment)
	}

	deployment.Logs = append(deployment.Logs, 
		fmt.Sprintf("Created %d service instances: %v", replicas, createdServices))

	c.JSON(http.StatusCreated, gin.H{
		"deployment_id": deploymentID,
		"services":      createdServices,
		"status":        "deploying",
	})
}

// StartService starts a specific service
func (o *Orchestrator) StartService(c *gin.Context) {
	serviceID := c.Param("id")

	o.mutex.Lock()
	defer o.mutex.Unlock()

	service, exists := o.services[serviceID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	if service.Status == "running" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Service is already running"})
		return
	}

	service.Status = "starting"
	service.UpdatedAt = time.Now()

	// Simulate starting service
	go func() {
		time.Sleep(2 * time.Second)
		o.mutex.Lock()
		service.Status = "running"
		service.Health = "healthy"
		service.UpdatedAt = time.Now()
		o.mutex.Unlock()
	}()

	c.JSON(http.StatusOK, gin.H{
		"service_id": serviceID,
		"status":     "starting",
	})
}

// StopService stops a specific service
func (o *Orchestrator) StopService(c *gin.Context) {
	serviceID := c.Param("id")

	o.mutex.Lock()
	defer o.mutex.Unlock()

	service, exists := o.services[serviceID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	if service.Status == "stopped" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Service is already stopped"})
		return
	}

	if err := o.stopServiceInstance(service); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"service_id": serviceID,
		"status":     "stopped",
	})
}

// RestartService restarts a specific service
func (o *Orchestrator) RestartService(c *gin.Context) {
	serviceID := c.Param("id")

	o.mutex.Lock()
	defer o.mutex.Unlock()

	service, exists := o.services[serviceID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	service.Status = "restarting"
	service.UpdatedAt = time.Now()

	// Simulate restart
	go func() {
		time.Sleep(3 * time.Second)
		o.mutex.Lock()
		service.Status = "running"
		service.Health = "healthy"
		service.UpdatedAt = time.Now()
		o.mutex.Unlock()
	}()

	c.JSON(http.StatusOK, gin.H{
		"service_id": serviceID,
		"status":     "restarting",
	})
}

// RemoveService removes a service completely
func (o *Orchestrator) RemoveService(c *gin.Context) {
	serviceID := c.Param("id")

	o.mutex.Lock()
	defer o.mutex.Unlock()

	service, exists := o.services[serviceID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	// Stop service if running
	if service.Status == "running" {
		if err := o.stopServiceInstance(service); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to stop service: %v", err)})
			return
		}
	}

	// Remove from services map
	delete(o.services, serviceID)

	c.JSON(http.StatusOK, gin.H{
		"service_id": serviceID,
		"status":     "removed",
	})
}

// GetServiceStatus returns the status of a specific service
func (o *Orchestrator) GetServiceStatus(c *gin.Context) {
	serviceID := c.Param("id")

	o.mutex.RLock()
	defer o.mutex.RUnlock()

	service, exists := o.services[serviceID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	c.JSON(http.StatusOK, service)
}

// GetServiceLogs returns logs for a specific service
func (o *Orchestrator) GetServiceLogs(c *gin.Context) {
	serviceID := c.Param("id")

	o.mutex.RLock()
	defer o.mutex.RUnlock()

	service, exists := o.services[serviceID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	// In a real implementation, this would fetch actual container logs
	logs := []string{
		fmt.Sprintf("[%s] Service %s started", time.Now().Format(time.RFC3339), service.Name),
		fmt.Sprintf("[%s] Health check passed", time.Now().Add(-30*time.Second).Format(time.RFC3339)),
		fmt.Sprintf("[%s] Service ready to accept connections", time.Now().Add(-25*time.Second).Format(time.RFC3339)),
	}

	c.JSON(http.StatusOK, gin.H{
		"service_id": serviceID,
		"logs":       logs,
	})
}

// ListDeployments returns all deployments
func (o *Orchestrator) ListDeployments(c *gin.Context) {
	o.mutex.RLock()
	defer o.mutex.RUnlock()

	deployments := make([]*Deployment, 0, len(o.deployments))
	for _, deployment := range o.deployments {
		deployments = append(deployments, deployment)
	}

	c.JSON(http.StatusOK, gin.H{
		"deployments": deployments,
		"total":       len(deployments),
	})
}

// GetDeployment returns a specific deployment
func (o *Orchestrator) GetDeployment(c *gin.Context) {
	deploymentID := c.Param("id")

	o.mutex.RLock()
	defer o.mutex.RUnlock()

	deployment, exists := o.deployments[deploymentID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Deployment not found"})
		return
	}

	c.JSON(http.StatusOK, deployment)
}

// RollbackDeployment rolls back a deployment
func (o *Orchestrator) RollbackDeployment(c *gin.Context) {
	deploymentID := c.Param("id")

	o.mutex.Lock()
	defer o.mutex.Unlock()

	deployment, exists := o.deployments[deploymentID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Deployment not found"})
		return
	}

	deployment.Status = "rolling_back"
	deployment.UpdatedAt = time.Now()
	deployment.Logs = append(deployment.Logs, 
		fmt.Sprintf("Rollback initiated at %s", time.Now().Format(time.RFC3339)))

	// Simulate rollback process
	go func() {
		time.Sleep(5 * time.Second)
		o.mutex.Lock()
		deployment.Status = "rolled_back"
		deployment.UpdatedAt = time.Now()
		deployment.Logs = append(deployment.Logs, 
			fmt.Sprintf("Rollback completed at %s", time.Now().Format(time.RFC3339)))
		o.mutex.Unlock()
	}()

	c.JSON(http.StatusOK, gin.H{
		"deployment_id": deploymentID,
		"status":        "rolling_back",
	})
}

// DeleteDeployment deletes a deployment record
func (o *Orchestrator) DeleteDeployment(c *gin.Context) {
	deploymentID := c.Param("id")

	o.mutex.Lock()
	defer o.mutex.Unlock()

	if _, exists := o.deployments[deploymentID]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Deployment not found"})
		return
	}

	delete(o.deployments, deploymentID)

	c.JSON(http.StatusOK, gin.H{
		"deployment_id": deploymentID,
		"status":        "deleted",
	})
}

// ListNodes returns all cluster nodes
func (o *Orchestrator) ListNodes(c *gin.Context) {
	o.mutex.RLock()
	defer o.mutex.RUnlock()

	nodes := make([]*Node, 0, len(o.nodes))
	for _, node := range o.nodes {
		nodes = append(nodes, node)
	}

	c.JSON(http.StatusOK, gin.H{
		"nodes": nodes,
		"total": len(nodes),
	})
}

// GetClusterResources returns cluster resource usage
func (o *Orchestrator) GetClusterResources(c *gin.Context) {
	o.mutex.RLock()
	defer o.mutex.RUnlock()

	totalResources := &NodeResources{
		CPU:     ResourceUsage{Used: "0", Available: "0", Total: "0", Percent: 0},
		Memory:  ResourceUsage{Used: "0GB", Available: "0GB", Total: "0GB", Percent: 0},
		Storage: ResourceUsage{Used: "0GB", Available: "0GB", Total: "0GB", Percent: 0},
		Network: ResourceUsage{Used: "0Mbps", Available: "0Mbps", Total: "0Mbps", Percent: 0},
		Pods:    ResourceUsage{Used: "0", Available: "0", Total: "0", Percent: 0},
	}

	// Aggregate resources from all nodes
	for _, node := range o.nodes {
		// Simple aggregation (in practice would be more complex)
		if totalCPU, err := strconv.Atoi(node.Resources.CPU.Total); err == nil {
			if currentTotal, err := strconv.Atoi(totalResources.CPU.Total); err == nil {
				totalResources.CPU.Total = strconv.Itoa(currentTotal + totalCPU)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"cluster_resources": totalResources,
		"node_count":        len(o.nodes),
	})
}

// GetClusterEvents returns cluster events
func (o *Orchestrator) GetClusterEvents(c *gin.Context) {
	// Simulate cluster events
	events := []map[string]interface{}{
		{
			"timestamp": time.Now().Add(-5 * time.Minute).Unix(),
			"type":      "Normal",
			"reason":    "ServiceStarted",
			"message":   "Service hello-service-0 started successfully",
		},
		{
			"timestamp": time.Now().Add(-10 * time.Minute).Unix(),
			"type":      "Warning",
			"reason":    "HighMemoryUsage",
			"message":   "Node localhost memory usage is above 80%",
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"events": events,
		"total":  len(events),
	})
}

// SyncServices synchronizes service states
func (o *Orchestrator) SyncServices(c *gin.Context) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	syncedCount := 0
	for _, service := range o.services {
		// Simulate sync operation
		service.UpdatedAt = time.Now()
		syncedCount++
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Services synchronized",
		"synced_count":   syncedCount,
		"total_services": len(o.services),
	})
}

// CleanupResources performs resource cleanup
func (o *Orchestrator) CleanupResources(c *gin.Context) {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	cleanedCount := 0
	
	// Clean up stopped services
	for id, service := range o.services {
		if service.Status == "stopped" {
			delete(o.services, id)
			cleanedCount++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message":        "Resources cleaned up",
		"cleaned_count":  cleanedCount,
		"remaining_services": len(o.services),
	})
}

// GetMetrics returns orchestrator metrics
func (o *Orchestrator) GetMetrics(c *gin.Context) {
	o.mutex.RLock()
	defer o.mutex.RUnlock()

	runningServices := 0
	stoppedServices := 0
	for _, service := range o.services {
		if service.Status == "running" {
			runningServices++
		} else {
			stoppedServices++
		}
	}

	activeDeployments := 0
	for _, deployment := range o.deployments {
		if deployment.Status == "deploying" || deployment.Status == "rolling_back" {
			activeDeployments++
		}
	}

	metrics := map[string]interface{}{
		"services": map[string]int{
			"total":   len(o.services),
			"running": runningServices,
			"stopped": stoppedServices,
		},
		"deployments": map[string]int{
			"total":  len(o.deployments),
			"active": activeDeployments,
		},
		"nodes": map[string]int{
			"total": len(o.nodes),
		},
		"uptime": time.Since(time.Now().Add(-time.Hour)).String(), // Placeholder
	}

	c.JSON(http.StatusOK, gin.H{
		"metrics":   metrics,
		"timestamp": time.Now().Unix(),
	})
}

// deployServiceInstance simulates deploying a service instance
func (o *Orchestrator) deployServiceInstance(service *ServiceInstance, deployment *Deployment) {
	// Simulate deployment time
	time.Sleep(3 * time.Second)

	o.mutex.Lock()
	defer o.mutex.Unlock()

	service.Status = "running"
	service.Health = "healthy"
	service.UpdatedAt = time.Now()

	deployment.Status = "deployed"
	deployment.UpdatedAt = time.Now()
	deployment.Logs = append(deployment.Logs, 
		fmt.Sprintf("Service %s deployed successfully at %s", 
			service.ID, time.Now().Format(time.RFC3339)))
}