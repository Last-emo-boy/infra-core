package orchestrator

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/last-emo-boy/infra-core/pkg/config"
	"github.com/last-emo-boy/infra-core/pkg/database"
)

// Orchestrator manages service deployments and lifecycle
type Orchestrator struct {
	db          *database.DB
	config      *config.Config
	services    map[string]*ServiceInstance
	deployments map[string]*Deployment
	nodes       map[string]*Node
	mutex       sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	running     bool
}

// ServiceInstance represents a running service instance
type ServiceInstance struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Image       string                 `json:"image"`
	Port        int                    `json:"port"`
	Status      string                 `json:"status"`
	Health      string                 `json:"health"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Environment map[string]string      `json:"environment"`
	Resources   *ResourceRequirements  `json:"resources"`
	Config      map[string]interface{} `json:"config"`
}

// Deployment represents a deployment operation
type Deployment struct {
	ID          string                 `json:"id"`
	ServiceName string                 `json:"service_name"`
	Version     string                 `json:"version"`
	Status      string                 `json:"status"`
	Strategy    string                 `json:"strategy"`
	Config      map[string]interface{} `json:"config"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Logs        []string               `json:"logs"`
}

// Node represents a cluster node
type Node struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Status        string                 `json:"status"`
	Resources     *NodeResources         `json:"resources"`
	Services      []string               `json:"services"`
	LastHeartbeat time.Time              `json:"last_heartbeat"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// ResourceRequirements defines resource requirements for a service
type ResourceRequirements struct {
	CPU     string `json:"cpu"`
	Memory  string `json:"memory"`
	Storage string `json:"storage"`
}

// NodeResources represents available resources on a node
type NodeResources struct {
	CPU       ResourceUsage `json:"cpu"`
	Memory    ResourceUsage `json:"memory"`
	Storage   ResourceUsage `json:"storage"`
	Network   ResourceUsage `json:"network"`
	Pods      ResourceUsage `json:"pods"`
}

// ResourceUsage represents resource usage information
type ResourceUsage struct {
	Used      string  `json:"used"`
	Available string  `json:"available"`
	Total     string  `json:"total"`
	Percent   float64 `json:"percent"`
}

// DeployRequest represents a service deployment request
type DeployRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Image       string                 `json:"image" binding:"required"`
	Port        int                    `json:"port"`
	Replicas    int                    `json:"replicas"`
	Environment map[string]string      `json:"environment"`
	Resources   *ResourceRequirements  `json:"resources"`
	Config      map[string]interface{} `json:"config"`
	Strategy    string                 `json:"strategy"`
}

// New creates a new orchestrator instance
func New(db *database.DB, config *config.Config) *Orchestrator {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &Orchestrator{
		db:          db,
		config:      config,
		services:    make(map[string]*ServiceInstance),
		deployments: make(map[string]*Deployment),
		nodes:       make(map[string]*Node),
		ctx:         ctx,
		cancel:      cancel,
		running:     false,
	}
}

// Start starts the orchestrator
func (o *Orchestrator) Start() error {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if o.running {
		return fmt.Errorf("orchestrator is already running")
	}

	log.Println("🎭 Starting orchestrator engine...")

	// Initialize nodes
	if err := o.initializeNodes(); err != nil {
		return fmt.Errorf("failed to initialize nodes: %w", err)
	}

	// Start background tasks
	go o.healthCheckLoop()
	go o.resourceMonitorLoop()
	go o.cleanupLoop()

	o.running = true
	log.Println("✅ Orchestrator started successfully")

	return nil
}

// Stop stops the orchestrator
func (o *Orchestrator) Stop() {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	if !o.running {
		return
	}

	log.Println("🛑 Stopping orchestrator...")

	// Cancel context to stop background tasks
	o.cancel()

	// Stop all services
	for _, service := range o.services {
		if service.Status == "running" {
			if err := o.stopServiceInstance(service); err != nil {
				log.Printf("Failed to stop service %s: %v", service.ID, err)
			}
		}
	}

	o.running = false
	log.Println("✅ Orchestrator stopped")
}

// GetStatus returns the orchestrator status
func (o *Orchestrator) GetStatus() map[string]interface{} {
	o.mutex.RLock()
	defer o.mutex.RUnlock()

	runningServices := 0
	for _, service := range o.services {
		if service.Status == "running" {
			runningServices++
		}
	}

	return map[string]interface{}{
		"running":           o.running,
		"total_services":    len(o.services),
		"running_services":  runningServices,
		"total_deployments": len(o.deployments),
		"nodes":             len(o.nodes),
		"uptime":            time.Since(time.Now()).String(),
	}
}

// initializeNodes discovers and initializes cluster nodes
func (o *Orchestrator) initializeNodes() error {
	// For now, initialize with localhost as single node
	localNode := &Node{
		ID:     "localhost",
		Name:   "localhost",
		Status: "ready",
		Resources: &NodeResources{
			CPU:     ResourceUsage{Used: "0", Available: "4", Total: "4", Percent: 0},
			Memory:  ResourceUsage{Used: "0GB", Available: "8GB", Total: "8GB", Percent: 0},
			Storage: ResourceUsage{Used: "0GB", Available: "100GB", Total: "100GB", Percent: 0},
			Network: ResourceUsage{Used: "0Mbps", Available: "1000Mbps", Total: "1000Mbps", Percent: 0},
			Pods:    ResourceUsage{Used: "0", Available: "100", Total: "100", Percent: 0},
		},
		Services:      []string{},
		LastHeartbeat: time.Now(),
		Metadata:      make(map[string]interface{}),
	}

	o.nodes[localNode.ID] = localNode
	return nil
}

// healthCheckLoop performs periodic health checks
func (o *Orchestrator) healthCheckLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-o.ctx.Done():
			return
		case <-ticker.C:
			o.performHealthChecks()
		}
	}
}

// resourceMonitorLoop monitors resource usage
func (o *Orchestrator) resourceMonitorLoop() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-o.ctx.Done():
			return
		case <-ticker.C:
			o.updateResourceUsage()
		}
	}
}

// cleanupLoop performs periodic cleanup
func (o *Orchestrator) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-o.ctx.Done():
			return
		case <-ticker.C:
			o.performCleanup()
		}
	}
}

// performHealthChecks checks health of all services
func (o *Orchestrator) performHealthChecks() {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	for _, service := range o.services {
		if service.Status == "running" {
			// Simple health check by attempting to connect to service port
			health := "healthy"
			if service.Port > 0 {
				url := fmt.Sprintf("http://localhost:%d/health", service.Port)
				client := &http.Client{Timeout: 5 * time.Second}
				if _, err := client.Get(url); err != nil {
					health = "unhealthy"
				}
			}
			service.Health = health
			service.UpdatedAt = time.Now()
		}
	}
}

// updateResourceUsage updates resource usage information
func (o *Orchestrator) updateResourceUsage() {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	for _, node := range o.nodes {
		// Update node heartbeat
		node.LastHeartbeat = time.Now()

		// Calculate resource usage (simplified)
		runningServices := 0
		for _, service := range o.services {
			if service.Status == "running" {
				runningServices++
			}
		}

		// Update pod usage
		node.Resources.Pods.Used = fmt.Sprintf("%d", runningServices)
		if total, _ := strconv.Atoi(node.Resources.Pods.Total); total > 0 {
			node.Resources.Pods.Percent = float64(runningServices) / float64(total) * 100
		}
	}
}

// performCleanup cleans up stopped services and old deployments
func (o *Orchestrator) performCleanup() {
	o.mutex.Lock()
	defer o.mutex.Unlock()

	// Clean up stopped services older than 1 hour
	cutoff := time.Now().Add(-1 * time.Hour)
	for id, service := range o.services {
		if service.Status == "stopped" && service.UpdatedAt.Before(cutoff) {
			delete(o.services, id)
			log.Printf("🧹 Cleaned up stopped service: %s", service.Name)
		}
	}

	// Clean up old deployments (keep last 10)
	if len(o.deployments) > 10 {
		// Implementation would sort by date and remove oldest
		log.Printf("🧹 Deployment cleanup needed (current: %d)", len(o.deployments))
	}
}

// stopServiceInstance stops a service instance
func (o *Orchestrator) stopServiceInstance(service *ServiceInstance) error {
	// In a real implementation, this would interact with container runtime
	// For now, just update status
	service.Status = "stopped"
	service.UpdatedAt = time.Now()
	log.Printf("🛑 Stopped service instance: %s", service.Name)
	return nil
}