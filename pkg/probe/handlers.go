package probe

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateProbe creates a new monitoring probe
func (pm *ProbeMonitor) CreateProbe(c *gin.Context) {
	var req CreateProbeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse interval and timeout
	interval, err := time.ParseDuration(req.Interval)
	if err != nil {
		interval = 60 * time.Second // Default
	}

	timeout, err := time.ParseDuration(req.Timeout)
	if err != nil {
		timeout = 10 * time.Second // Default
	}

	// Create probe configuration
	probe := &ProbeConfig{
		ID:              uuid.New().String(),
		Name:            req.Name,
		Type:            req.Type,
		Target:          req.Target,
		Interval:        interval,
		Timeout:         timeout,
		Retries:         req.Retries,
		Enabled:         true,
		ExpectedStatus:  req.ExpectedStatus,
		ExpectedContent: req.ExpectedContent,
		Headers:         req.Headers,
		Thresholds:      req.Thresholds,
		Tags:            req.Tags,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		Config:          req.Config,
	}

	// Set defaults
	if probe.Retries <= 0 {
		probe.Retries = 3
	}
	if probe.ExpectedStatus <= 0 && probe.Type == "http" {
		probe.ExpectedStatus = 200
	}
	if probe.Thresholds == nil {
		probe.Thresholds = &ProbeThresholds{
			ResponseTime:    5 * time.Second,
			SuccessRate:     0.95,
			ConsecutiveFail: 3,
		}
	}

	pm.mutex.Lock()
	pm.probes[probe.ID] = probe
	pm.mutex.Unlock()

	c.JSON(http.StatusCreated, gin.H{
		"probe_id": probe.ID,
		"message":  "Probe created successfully",
		"probe":    probe,
	})
}

// ListProbes returns all monitoring probes
func (pm *ProbeMonitor) ListProbes(c *gin.Context) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	probes := make([]*ProbeConfig, 0, len(pm.probes))
	for _, probe := range pm.probes {
		probes = append(probes, probe)
	}

	c.JSON(http.StatusOK, gin.H{
		"probes": probes,
		"total":  len(probes),
	})
}

// GetProbe returns a specific probe configuration
func (pm *ProbeMonitor) GetProbe(c *gin.Context) {
	probeID := c.Param("id")

	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	probe, exists := pm.probes[probeID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Probe not found"})
		return
	}

	c.JSON(http.StatusOK, probe)
}

// UpdateProbe updates a probe configuration
func (pm *ProbeMonitor) UpdateProbe(c *gin.Context) {
	probeID := c.Param("id")

	var req CreateProbeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	probe, exists := pm.probes[probeID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Probe not found"})
		return
	}

	// Update probe configuration
	probe.Name = req.Name
	probe.Target = req.Target
	probe.ExpectedStatus = req.ExpectedStatus
	probe.ExpectedContent = req.ExpectedContent
	probe.Headers = req.Headers
	probe.Thresholds = req.Thresholds
	probe.Tags = req.Tags
	probe.Config = req.Config
	probe.UpdatedAt = time.Now()

	// Parse and update interval/timeout if provided
	if req.Interval != "" {
		if interval, err := time.ParseDuration(req.Interval); err == nil {
			probe.Interval = interval
		}
	}
	if req.Timeout != "" {
		if timeout, err := time.ParseDuration(req.Timeout); err == nil {
			probe.Timeout = timeout
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Probe updated successfully",
		"probe":   probe,
	})
}

// DeleteProbe removes a monitoring probe
func (pm *ProbeMonitor) DeleteProbe(c *gin.Context) {
	probeID := c.Param("id")

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if _, exists := pm.probes[probeID]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Probe not found"})
		return
	}

	delete(pm.probes, probeID)

	c.JSON(http.StatusOK, gin.H{
		"message":  "Probe deleted successfully",
		"probe_id": probeID,
	})
}

// EnableProbe enables a monitoring probe
func (pm *ProbeMonitor) EnableProbe(c *gin.Context) {
	probeID := c.Param("id")

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	probe, exists := pm.probes[probeID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Probe not found"})
		return
	}

	probe.Enabled = true
	probe.UpdatedAt = time.Now()

	c.JSON(http.StatusOK, gin.H{
		"message":  "Probe enabled",
		"probe_id": probeID,
	})
}

// DisableProbe disables a monitoring probe
func (pm *ProbeMonitor) DisableProbe(c *gin.Context) {
	probeID := c.Param("id")

	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	probe, exists := pm.probes[probeID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Probe not found"})
		return
	}

	probe.Enabled = false
	probe.UpdatedAt = time.Now()

	c.JSON(http.StatusOK, gin.H{
		"message":  "Probe disabled",
		"probe_id": probeID,
	})
}

// ListResults returns probe execution results
func (pm *ProbeMonitor) ListResults(c *gin.Context) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	// Get query parameters
	limitStr := c.DefaultQuery("limit", "100")
	limit, _ := strconv.Atoi(limitStr)

	results := make([]*ProbeResult, 0)
	count := 0
	for _, result := range pm.results {
		if count >= limit {
			break
		}
		results = append(results, result)
		count++
	}

	c.JSON(http.StatusOK, gin.H{
		"results": results,
		"total":   len(pm.results),
		"limit":   limit,
	})
}

// GetProbeResults returns results for a specific probe
func (pm *ProbeMonitor) GetProbeResults(c *gin.Context) {
	probeID := c.Param("probe_id")

	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	results := make([]*ProbeResult, 0)
	for _, result := range pm.results {
		if result.ProbeID == probeID {
			results = append(results, result)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"probe_id": probeID,
		"results":  results,
		"total":    len(results),
	})
}

// GetLatestResult returns the latest result for a probe
func (pm *ProbeMonitor) GetLatestResult(c *gin.Context) {
	probeID := c.Param("probe_id")

	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	var latestResult *ProbeResult
	for _, result := range pm.results {
		if result.ProbeID == probeID {
			if latestResult == nil || result.Timestamp.After(latestResult.Timestamp) {
				latestResult = result
			}
		}
	}

	if latestResult == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No results found for probe"})
		return
	}

	c.JSON(http.StatusOK, latestResult)
}

// GetProbeMetrics returns aggregated metrics for a probe
func (pm *ProbeMonitor) GetProbeMetrics(c *gin.Context) {
	probeID := c.Param("probe_id")

	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	// Calculate metrics from results
	var totalChecks, successfulChecks, failedChecks int
	var totalResponse, maxResponse, minResponse time.Duration
	var lastCheck, lastSuccess, lastFailure time.Time
	var consecutiveFails int

	minResponse = time.Hour // Initialize to large value

	for _, result := range pm.results {
		if result.ProbeID == probeID {
			totalChecks++
			
			if result.Status == "success" {
				successfulChecks++
				consecutiveFails = 0
				if result.Timestamp.After(lastSuccess) {
					lastSuccess = result.Timestamp
				}
			} else {
				failedChecks++
				consecutiveFails++
				if result.Timestamp.After(lastFailure) {
					lastFailure = result.Timestamp
				}
			}

			totalResponse += result.ResponseTime
			if result.ResponseTime > maxResponse {
				maxResponse = result.ResponseTime
			}
			if result.ResponseTime < minResponse {
				minResponse = result.ResponseTime
			}
			if result.Timestamp.After(lastCheck) {
				lastCheck = result.Timestamp
			}
		}
	}

	// Calculate averages and rates
	var averageResponse time.Duration
	var successRate, uptime float64

	if totalChecks > 0 {
		averageResponse = totalResponse / time.Duration(totalChecks)
		successRate = float64(successfulChecks) / float64(totalChecks)
		uptime = successRate * 100
	}

	if minResponse == time.Hour {
		minResponse = 0
	}

	metrics := &ProbeMetrics{
		ProbeID:          probeID,
		TotalChecks:      totalChecks,
		SuccessfulChecks: successfulChecks,
		FailedChecks:     failedChecks,
		SuccessRate:      successRate,
		AverageResponse:  averageResponse,
		MaxResponse:      maxResponse,
		MinResponse:      minResponse,
		LastCheck:        lastCheck,
		LastSuccess:      lastSuccess,
		LastFailure:      lastFailure,
		ConsecutiveFails: consecutiveFails,
		Uptime:           uptime,
	}

	c.JSON(http.StatusOK, metrics)
}

// GetProbeHistory returns historical data for a probe
func (pm *ProbeMonitor) GetProbeHistory(c *gin.Context) {
	probeID := c.Param("probe_id")
	
	// Get time range from query parameters
	hoursStr := c.DefaultQuery("hours", "24")
	hours, _ := strconv.Atoi(hoursStr)
	
	since := time.Now().Add(-time.Duration(hours) * time.Hour)

	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	results := make([]*ProbeResult, 0)
	for _, result := range pm.results {
		if result.ProbeID == probeID && result.Timestamp.After(since) {
			results = append(results, result)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"probe_id": probeID,
		"history":  results,
		"hours":    hours,
		"total":    len(results),
	})
}

// GetServiceHealth returns overall service health status
func (pm *ProbeMonitor) GetServiceHealth(c *gin.Context) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	serviceHealth := make(map[string]interface{})

	// Group probes by service (based on tags)
	for _, probe := range pm.probes {
		for _, tag := range probe.Tags {
			if _, exists := serviceHealth[tag]; !exists {
				serviceHealth[tag] = map[string]interface{}{
					"status":     "unknown",
					"probes":     0,
					"healthy":    0,
					"unhealthy":  0,
					"last_check": time.Time{},
				}
			}
			
			service := serviceHealth[tag].(map[string]interface{})
			service["probes"] = service["probes"].(int) + 1

			// Get latest result for this probe
			var latestResult *ProbeResult
			for _, result := range pm.results {
				if result.ProbeID == probe.ID {
					if latestResult == nil || result.Timestamp.After(latestResult.Timestamp) {
						latestResult = result
					}
				}
			}

			if latestResult != nil {
				if latestResult.Status == "success" {
					service["healthy"] = service["healthy"].(int) + 1
				} else {
					service["unhealthy"] = service["unhealthy"].(int) + 1
				}

				if latestResult.Timestamp.After(service["last_check"].(time.Time)) {
					service["last_check"] = latestResult.Timestamp
				}
			}

			// Determine overall service status
			if service["unhealthy"].(int) > 0 {
				service["status"] = "unhealthy"
			} else if service["healthy"].(int) > 0 {
				service["status"] = "healthy"
			}

			serviceHealth[tag] = service
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"services": serviceHealth,
		"timestamp": time.Now(),
	})
}

// GetServiceHealthDetail returns detailed health info for a specific service
func (pm *ProbeMonitor) GetServiceHealthDetail(c *gin.Context) {
	serviceID := c.Param("service_id")

	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	// Find probes for this service
	serviceProbes := make([]*ProbeConfig, 0)
	for _, probe := range pm.probes {
		for _, tag := range probe.Tags {
			if tag == serviceID {
				serviceProbes = append(serviceProbes, probe)
				break
			}
		}
	}

	if len(serviceProbes) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Service not found"})
		return
	}

	// Get results for service probes
	serviceResults := make(map[string][]*ProbeResult)
	for _, probe := range serviceProbes {
		results := make([]*ProbeResult, 0)
		for _, result := range pm.results {
			if result.ProbeID == probe.ID {
				results = append(results, result)
			}
		}
		serviceResults[probe.ID] = results
	}

	c.JSON(http.StatusOK, gin.H{
		"service_id": serviceID,
		"probes":     serviceProbes,
		"results":    serviceResults,
	})
}

// GetHealthOverview returns a high-level health overview
func (pm *ProbeMonitor) GetHealthOverview(c *gin.Context) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	totalProbes := len(pm.probes)
	enabledProbes := 0
	healthyProbes := 0
	unhealthyProbes := 0
	activeAlerts := 0

	for _, probe := range pm.probes {
		if probe.Enabled {
			enabledProbes++
		}

		// Get latest result
		var latestResult *ProbeResult
		for _, result := range pm.results {
			if result.ProbeID == probe.ID {
				if latestResult == nil || result.Timestamp.After(latestResult.Timestamp) {
					latestResult = result
				}
			}
		}

		if latestResult != nil {
			if latestResult.Status == "success" {
				healthyProbes++
			} else {
				unhealthyProbes++
			}
		}
	}

	for _, alert := range pm.alerts {
		if alert.Status == "active" {
			activeAlerts++
		}
	}

	// Determine overall health status
	overallStatus := "healthy"
	if unhealthyProbes > 0 {
		if unhealthyProbes >= healthyProbes {
			overallStatus = "critical"
		} else {
			overallStatus = "degraded"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"overall_status":  overallStatus,
		"total_probes":    totalProbes,
		"enabled_probes":  enabledProbes,
		"healthy_probes":  healthyProbes,
		"unhealthy_probes": unhealthyProbes,
		"active_alerts":   activeAlerts,
		"timestamp":       time.Now(),
	})
}

// GetActiveAlerts returns all active alerts
func (pm *ProbeMonitor) GetActiveAlerts(c *gin.Context) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	activeAlerts := make([]*Alert, 0)
	for _, alert := range pm.alerts {
		if alert.Status == "active" {
			activeAlerts = append(activeAlerts, alert)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"alerts": activeAlerts,
		"total":  len(activeAlerts),
	})
}

// TriggerFullScan triggers a full monitoring scan
func (pm *ProbeMonitor) TriggerFullScan(c *gin.Context) {
	go pm.executeProbes()

	c.JSON(http.StatusOK, gin.H{
		"message": "Full scan triggered",
		"timestamp": time.Now(),
	})
}

// CleanupOldResults removes old probe results
func (pm *ProbeMonitor) CleanupOldResults(c *gin.Context) {
	pm.performCleanup()

	c.JSON(http.StatusOK, gin.H{
		"message": "Cleanup completed",
		"timestamp": time.Now(),
	})
}

// GetMonitorMetrics returns monitoring system metrics
func (pm *ProbeMonitor) GetMonitorMetrics(c *gin.Context) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	metrics := map[string]interface{}{
		"probes": map[string]int{
			"total":   len(pm.probes),
			"enabled": 0,
			"disabled": 0,
		},
		"results": map[string]int{
			"total": len(pm.results),
		},
		"alerts": map[string]int{
			"total":    len(pm.alerts),
			"active":   0,
			"resolved": 0,
		},
		"uptime": time.Since(time.Now().Add(-time.Hour)).String(), // Placeholder
	}

	// Count enabled/disabled probes
	for _, probe := range pm.probes {
		if probe.Enabled {
			metrics["probes"].(map[string]int)["enabled"]++
		} else {
			metrics["probes"].(map[string]int)["disabled"]++
		}
	}

	// Count alert statuses
	for _, alert := range pm.alerts {
		if alert.Status == "active" {
			metrics["alerts"].(map[string]int)["active"]++
		} else {
			metrics["alerts"].(map[string]int)["resolved"]++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"metrics": metrics,
		"timestamp": time.Now(),
	})
}

// GetDetailedStatus returns detailed probe monitor status
func (pm *ProbeMonitor) GetDetailedStatus(c *gin.Context) {
	status := pm.GetStatus()
	
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	// Add more detailed information
	status["probe_details"] = make(map[string]interface{})
	for id, probe := range pm.probes {
		var latestResult *ProbeResult
		for _, result := range pm.results {
			if result.ProbeID == id {
				if latestResult == nil || result.Timestamp.After(latestResult.Timestamp) {
					latestResult = result
				}
			}
		}

		probeStatus := "unknown"
		if latestResult != nil {
			probeStatus = latestResult.Status
		}

		status["probe_details"].(map[string]interface{})[id] = map[string]interface{}{
			"name":        probe.Name,
			"type":        probe.Type,
			"enabled":     probe.Enabled,
			"status":      probeStatus,
			"last_check":  func() interface{} {
				if latestResult != nil {
					return latestResult.Timestamp
				}
				return nil
			}(),
		}
	}

	c.JSON(http.StatusOK, status)
}