package services

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/last-emo-boy/infra-core/pkg/database"
)

// HealthChecker performs health checks on registered services
type HealthChecker struct {
	db       *database.DB
	client   *http.Client
	interval time.Duration
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(db *database.DB) *HealthChecker {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &HealthChecker{
		db:       db,
		client:   &http.Client{Timeout: 10 * time.Second},
		interval: 1 * time.Minute, // Check every minute
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start starts the health checker
func (hc *HealthChecker) Start() {
	hc.wg.Add(1)
	go hc.run()
}

// Stop stops the health checker
func (hc *HealthChecker) Stop() {
	hc.cancel()
	hc.wg.Wait()
}

// run is the main health check loop
func (hc *HealthChecker) run() {
	defer hc.wg.Done()
	
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	// Initial check
	hc.checkAllServices()

	for {
		select {
		case <-hc.ctx.Done():
			return
		case <-ticker.C:
			hc.checkAllServices()
		}
	}
}

// checkAllServices checks the health of all registered services
func (hc *HealthChecker) checkAllServices() {
	serviceRepo := hc.db.RegisteredServiceRepository()
	services, err := serviceRepo.List()
	if err != nil {
		return
	}

	var wg sync.WaitGroup
	for _, service := range services {
		if service.HealthURL == nil || *service.HealthURL == "" {
			continue
		}

		wg.Add(1)
		go func(svc *database.RegisteredService) {
			defer wg.Done()
			hc.checkService(svc)
		}(service)
	}

	wg.Wait()
}

// checkService checks the health of a single service
func (hc *HealthChecker) checkService(service *database.RegisteredService) {
	if service.HealthURL == nil {
		return
	}

	start := time.Now()
	isHealthy := true
	var errorMessage *string

	resp, err := hc.client.Get(*service.HealthURL)
	if err != nil {
		isHealthy = false
		errMsg := err.Error()
		errorMessage = &errMsg
	} else {
		defer resp.Body.Close()
		
		// Consider 2xx status codes as healthy
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			isHealthy = false
			errMsg := resp.Status
			errorMessage = &errMsg
		}
	}

	responseTime := int(time.Since(start).Milliseconds())

	// Record health check result
	healthCheck := &database.ServiceHealthCheck{
		ServiceID:    service.ID,
		IsHealthy:    isHealthy,
		ResponseTime: responseTime,
		ErrorMessage: errorMessage,
		CheckedAt:    time.Now(),
	}

	healthRepo := hc.db.ServiceHealthCheckRepository()
	if err := healthRepo.Record(healthCheck); err != nil {
		// Log error but don't fail
		return
	}

	// Update service health status
	serviceRepo := hc.db.RegisteredServiceRepository()
	if err := serviceRepo.UpdateHealthStatus(service.ID, isHealthy); err != nil {
		// Log error but don't fail
		return
	}
}

// CheckService performs an immediate health check on a specific service
func (hc *HealthChecker) CheckService(serviceID string) (*database.ServiceHealthCheck, error) {
	serviceRepo := hc.db.RegisteredServiceRepository()
	service, err := serviceRepo.GetByID(serviceID)
	if err != nil {
		return nil, err
	}

	hc.checkService(service)

	// Return latest health check
	healthRepo := hc.db.ServiceHealthCheckRepository()
	return healthRepo.GetLatest(serviceID)
}

// CleanupOldHealthChecks removes old health check records
func (hc *HealthChecker) CleanupOldHealthChecks() {
	// Keep health checks for 7 days
	cutoff := time.Now().AddDate(0, 0, -7)
	
	healthRepo := hc.db.ServiceHealthCheckRepository()
	if err := healthRepo.CleanupOldChecks(cutoff); err != nil {
		// Log error but don't fail
		return
	}
}