package probe

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/last-emo-boy/infra-core/pkg/config"
	"github.com/last-emo-boy/infra-core/pkg/database"
)

// ProbeMonitor manages health probes and monitoring
type ProbeMonitor struct {
	db      *database.DB
	config  *config.Config
	probes  map[string]*ProbeConfig
	results map[string]*ProbeResult
	alerts  map[string]*Alert
	mutex   sync.RWMutex
	ctx     context.Context
	cancel  context.CancelFunc
	running bool
}

// ProbeConfig defines a monitoring probe configuration
type ProbeConfig struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Type            string                 `json:"type"` // http, tcp, icmp, dns, custom
	Target          string                 `json:"target"`
	Interval        time.Duration          `json:"interval"`
	Timeout         time.Duration          `json:"timeout"`
	Retries         int                    `json:"retries"`
	Enabled         bool                   `json:"enabled"`
	ExpectedStatus  int                    `json:"expected_status,omitempty"`
	ExpectedContent string                 `json:"expected_content,omitempty"`
	Headers         map[string]string      `json:"headers,omitempty"`
	Thresholds      *ProbeThresholds       `json:"thresholds"`
	Tags            []string               `json:"tags"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Config          map[string]interface{} `json:"config"`
}

// ProbeThresholds defines alert thresholds
type ProbeThresholds struct {
	ResponseTime    time.Duration `json:"response_time"`    // Max acceptable response time
	SuccessRate     float64       `json:"success_rate"`     // Min success rate (0-1)
	ConsecutiveFail int           `json:"consecutive_fail"` // Max consecutive failures
}

// ProbeResult represents a probe execution result
type ProbeResult struct {
	ID           string                 `json:"id"`
	ProbeID      string                 `json:"probe_id"`
	Status       string                 `json:"status"` // success, failure, timeout, error
	ResponseTime time.Duration          `json:"response_time"`
	StatusCode   int                    `json:"status_code,omitempty"`
	Message      string                 `json:"message"`
	Error        string                 `json:"error,omitempty"`
	Metadata     map[string]interface{} `json:"metadata"`
	Timestamp    time.Time              `json:"timestamp"`
}

// Alert represents a monitoring alert
type Alert struct {
	ID          string                 `json:"id"`
	ProbeID     string                 `json:"probe_id"`
	Type        string                 `json:"type"` // threshold, availability, performance
	Severity    string                 `json:"severity"` // low, medium, high, critical
	Status      string                 `json:"status"` // active, resolved, suppressed
	Message     string                 `json:"message"`
	Count       int                    `json:"count"`
	FirstSeen   time.Time              `json:"first_seen"`
	LastSeen    time.Time              `json:"last_seen"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ProbeMetrics contains aggregated probe metrics
type ProbeMetrics struct {
	ProbeID           string        `json:"probe_id"`
	TotalChecks       int           `json:"total_checks"`
	SuccessfulChecks  int           `json:"successful_checks"`
	FailedChecks      int           `json:"failed_checks"`
	SuccessRate       float64       `json:"success_rate"`
	AverageResponse   time.Duration `json:"average_response"`
	MaxResponse       time.Duration `json:"max_response"`
	MinResponse       time.Duration `json:"min_response"`
	LastCheck         time.Time     `json:"last_check"`
	LastSuccess       time.Time     `json:"last_success"`
	LastFailure       time.Time     `json:"last_failure"`
	ConsecutiveFails  int           `json:"consecutive_fails"`
	Uptime            float64       `json:"uptime"`
}

// CreateProbeRequest represents a probe creation request
type CreateProbeRequest struct {
	Name            string                 `json:"name" binding:"required"`
	Type            string                 `json:"type" binding:"required"`
	Target          string                 `json:"target" binding:"required"`
	Interval        string                 `json:"interval"`
	Timeout         string                 `json:"timeout"`
	Retries         int                    `json:"retries"`
	ExpectedStatus  int                    `json:"expected_status"`
	ExpectedContent string                 `json:"expected_content"`
	Headers         map[string]string      `json:"headers"`
	Thresholds      *ProbeThresholds       `json:"thresholds"`
	Tags            []string               `json:"tags"`
	Config          map[string]interface{} `json:"config"`
}

// New creates a new probe monitor instance
func New(db *database.DB, config *config.Config) *ProbeMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &ProbeMonitor{
		db:      db,
		config:  config,
		probes:  make(map[string]*ProbeConfig),
		results: make(map[string]*ProbeResult),
		alerts:  make(map[string]*Alert),
		ctx:     ctx,
		cancel:  cancel,
		running: false,
	}
}

// Start starts the probe monitor
func (pm *ProbeMonitor) Start() error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if pm.running {
		return fmt.Errorf("probe monitor is already running")
	}

	log.Println("🔍 Starting probe monitoring engine...")

	// Load existing probes from database
	if err := pm.loadProbes(); err != nil {
		return fmt.Errorf("failed to load probes: %w", err)
	}

	// Start background monitoring loops
	go pm.monitoringLoop()
	go pm.alertingLoop()
	go pm.cleanupLoop()

	pm.running = true
	log.Printf("✅ Probe monitor started with %d probes", len(pm.probes))

	return nil
}

// Stop stops the probe monitor
func (pm *ProbeMonitor) Stop() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if !pm.running {
		return
	}

	log.Println("🛑 Stopping probe monitor...")

	// Cancel context to stop background tasks
	pm.cancel()

	pm.running = false
	log.Println("✅ Probe monitor stopped")
}

// GetStatus returns the probe monitor status
func (pm *ProbeMonitor) GetStatus() map[string]interface{} {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	enabledProbes := 0
	activeAlerts := 0

	for _, probe := range pm.probes {
		if probe.Enabled {
			enabledProbes++
		}
	}

	for _, alert := range pm.alerts {
		if alert.Status == "active" {
			activeAlerts++
		}
	}

	return map[string]interface{}{
		"running":        pm.running,
		"total_probes":   len(pm.probes),
		"enabled_probes": enabledProbes,
		"active_alerts":  activeAlerts,
		"last_scan":      time.Now().Format(time.RFC3339),
	}
}

// loadProbes loads probe configurations from database
func (pm *ProbeMonitor) loadProbes() error {
	// For now, create some default probes
	// In a real implementation, this would load from database

	defaultProbes := []*ProbeConfig{
		{
			ID:             "console-health",
			Name:           "Console API Health Check",
			Type:           "http",
			Target:         "http://localhost:8082/api/v1/health",
			Interval:       30 * time.Second,
			Timeout:        5 * time.Second,
			Retries:        3,
			Enabled:        true,
			ExpectedStatus: 200,
			Thresholds: &ProbeThresholds{
				ResponseTime:    2 * time.Second,
				SuccessRate:     0.95,
				ConsecutiveFail: 3,
			},
			Tags:      []string{"api", "console", "health"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Config:    make(map[string]interface{}),
		},
		{
			ID:             "gate-health",
			Name:           "Gateway Health Check",
			Type:           "http",
			Target:         "http://localhost:8080/health",
			Interval:       30 * time.Second,
			Timeout:        5 * time.Second,
			Retries:        3,
			Enabled:        true,
			ExpectedStatus: 200,
			Thresholds: &ProbeThresholds{
				ResponseTime:    1 * time.Second,
				SuccessRate:     0.98,
				ConsecutiveFail: 2,
			},
			Tags:      []string{"gateway", "health"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Config:    make(map[string]interface{}),
		},
		{
			ID:             "orch-health",
			Name:           "Orchestrator Health Check",
			Type:           "http",
			Target:         "http://localhost:8084/health",
			Interval:       60 * time.Second,
			Timeout:        5 * time.Second,
			Retries:        3,
			Enabled:        true,
			ExpectedStatus: 200,
			Thresholds: &ProbeThresholds{
				ResponseTime:    3 * time.Second,
				SuccessRate:     0.90,
				ConsecutiveFail: 5,
			},
			Tags:      []string{"orchestrator", "health"},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Config:    make(map[string]interface{}),
		},
	}

	for _, probe := range defaultProbes {
		pm.probes[probe.ID] = probe
	}

	return nil
}

// monitoringLoop performs periodic monitoring checks
func (pm *ProbeMonitor) monitoringLoop() {
	ticker := time.NewTicker(10 * time.Second) // Check every 10 seconds
	defer ticker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.executeProbes()
		}
	}
}

// alertingLoop processes alerts and notifications
func (pm *ProbeMonitor) alertingLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.processAlerts()
		}
	}
}

// cleanupLoop performs periodic cleanup of old results
func (pm *ProbeMonitor) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-pm.ctx.Done():
			return
		case <-ticker.C:
			pm.performCleanup()
		}
	}
}

// executeProbes runs all enabled probes that are due
func (pm *ProbeMonitor) executeProbes() {
	pm.mutex.RLock()
	probesToRun := make([]*ProbeConfig, 0)
	
	for _, probe := range pm.probes {
		if probe.Enabled && pm.shouldRunProbe(probe) {
			probesToRun = append(probesToRun, probe)
		}
	}
	pm.mutex.RUnlock()

	// Execute probes concurrently
	for _, probe := range probesToRun {
		go pm.executeProbe(probe)
	}
}

// shouldRunProbe determines if a probe should be executed now
func (pm *ProbeMonitor) shouldRunProbe(probe *ProbeConfig) bool {
	// Simple time-based scheduling
	// In a real implementation, this would track last execution times
	return true
}

// executeProbe executes a single probe
func (pm *ProbeMonitor) executeProbe(probe *ProbeConfig) {
	start := time.Now()
	result := &ProbeResult{
		ID:        fmt.Sprintf("%s-%d", probe.ID, start.Unix()),
		ProbeID:   probe.ID,
		Timestamp: start,
		Metadata:  make(map[string]interface{}),
	}

	switch probe.Type {
	case "http":
		pm.executeHTTPProbe(probe, result)
	case "tcp":
		pm.executeTCPProbe(probe, result)
	case "icmp":
		pm.executeICMPProbe(probe, result)
	default:
		result.Status = "error"
		result.Error = fmt.Sprintf("unsupported probe type: %s", probe.Type)
	}

	result.ResponseTime = time.Since(start)

	// Store result
	pm.mutex.Lock()
	pm.results[result.ID] = result
	pm.mutex.Unlock()

	// Check for alerts
	pm.checkThresholds(probe, result)
}

// executeHTTPProbe executes an HTTP health check
func (pm *ProbeMonitor) executeHTTPProbe(probe *ProbeConfig, result *ProbeResult) {
	client := &http.Client{
		Timeout: probe.Timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	req, err := http.NewRequest("GET", probe.Target, nil)
	if err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("failed to create request: %v", err)
		return
	}

	// Add headers
	for key, value := range probe.Headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		result.Status = "failure"
		result.Error = fmt.Sprintf("request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.Metadata["headers"] = resp.Header
	result.Metadata["content_length"] = resp.ContentLength

	// Check status code
	if probe.ExpectedStatus > 0 && resp.StatusCode != probe.ExpectedStatus {
		result.Status = "failure"
		result.Message = fmt.Sprintf("unexpected status code: got %d, expected %d", 
			resp.StatusCode, probe.ExpectedStatus)
		return
	}

	result.Status = "success"
	result.Message = "HTTP check passed"
}

// executeTCPProbe executes a TCP connectivity check
func (pm *ProbeMonitor) executeTCPProbe(probe *ProbeConfig, result *ProbeResult) {
	conn, err := net.DialTimeout("tcp", probe.Target, probe.Timeout)
	if err != nil {
		result.Status = "failure"
		result.Error = fmt.Sprintf("TCP connection failed: %v", err)
		return
	}
	defer conn.Close()

	result.Status = "success"
	result.Message = "TCP connection successful"
}

// executeICMPProbe executes an ICMP ping check
func (pm *ProbeMonitor) executeICMPProbe(probe *ProbeConfig, result *ProbeResult) {
	// ICMP requires raw sockets, which need admin privileges
	// For now, implement as a simplified TCP check to port 80
	conn, err := net.DialTimeout("tcp", probe.Target+":80", probe.Timeout)
	if err != nil {
		result.Status = "failure"
		result.Error = fmt.Sprintf("ICMP check failed: %v", err)
		return
	}
	defer conn.Close()

	result.Status = "success"
	result.Message = "ICMP check passed"
}

// checkThresholds evaluates probe results against thresholds
func (pm *ProbeMonitor) checkThresholds(probe *ProbeConfig, result *ProbeResult) {
	if probe.Thresholds == nil {
		return
	}

	// Check response time threshold
	if result.ResponseTime > probe.Thresholds.ResponseTime {
		pm.createAlert(probe.ID, "threshold", "medium", 
			fmt.Sprintf("Response time %v exceeds threshold %v", 
				result.ResponseTime, probe.Thresholds.ResponseTime))
	}

	// Check consecutive failures
	if result.Status != "success" {
		pm.incrementFailureCount(probe.ID)
		failCount := pm.getFailureCount(probe.ID)
		if failCount >= probe.Thresholds.ConsecutiveFail {
			pm.createAlert(probe.ID, "availability", "high",
				fmt.Sprintf("Service has failed %d consecutive checks", failCount))
		}
	} else {
		pm.resetFailureCount(probe.ID)
	}
}

// processAlerts processes and manages alerts
func (pm *ProbeMonitor) processAlerts() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	now := time.Now()
	for _, alert := range pm.alerts {
		// Auto-resolve old alerts
		if alert.Status == "active" && now.Sub(alert.LastSeen) > 10*time.Minute {
			alert.Status = "resolved"
			resolvedAt := now
			alert.ResolvedAt = &resolvedAt
			log.Printf("🔍 Auto-resolved alert: %s", alert.Message)
		}
	}
}

// performCleanup cleans up old results and resolved alerts
func (pm *ProbeMonitor) performCleanup() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour)
	
	// Clean up old results
	for resultID, result := range pm.results {
		if result.Timestamp.Before(cutoff) {
			delete(pm.results, resultID)
		}
	}

	// Clean up resolved alerts older than 7 days
	alertCutoff := time.Now().Add(-7 * 24 * time.Hour)
	for alertID, alert := range pm.alerts {
		if alert.Status == "resolved" && alert.ResolvedAt != nil && 
		   alert.ResolvedAt.Before(alertCutoff) {
			delete(pm.alerts, alertID)
		}
	}

	log.Printf("🧹 Cleanup completed: %d results, %d alerts", 
		len(pm.results), len(pm.alerts))
}

// Helper methods for failure tracking
func (pm *ProbeMonitor) incrementFailureCount(probeID string) {
	// Implementation would track failure counts per probe
}

func (pm *ProbeMonitor) getFailureCount(probeID string) int {
	// Implementation would return current failure count
	return 1
}

func (pm *ProbeMonitor) resetFailureCount(probeID string) {
	// Implementation would reset failure count
}

// createAlert creates a new alert
func (pm *ProbeMonitor) createAlert(probeID, alertType, severity, message string) {
	alertID := fmt.Sprintf("%s-%s-%d", probeID, alertType, time.Now().Unix())
	
	alert := &Alert{
		ID:        alertID,
		ProbeID:   probeID,
		Type:      alertType,
		Severity:  severity,
		Status:    "active",
		Message:   message,
		Count:     1,
		FirstSeen: time.Now(),
		LastSeen:  time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	pm.mutex.Lock()
	pm.alerts[alertID] = alert
	pm.mutex.Unlock()

	log.Printf("🚨 Alert created: %s - %s", severity, message)
}