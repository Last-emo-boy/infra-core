package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/last-emo-boy/infra-core/pkg/config"
	"github.com/last-emo-boy/infra-core/pkg/database"
)

func TestGetServiceSummary(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{
		Gate: config.GateConfig{
			Host:  "127.0.0.1",
			Ports: config.PortsConfig{HTTP: 8080, HTTPS: 8443},
			Logs:  config.LogConfig{Level: "info", Console: true},
			ACME:  config.ACMEConfig{Enabled: false},
		},
		Console: config.ConsoleConfig{
			Host: "127.0.0.1",
			Port: 8082,
			Logs: config.LogConfig{Level: "info", Console: true},
			Database: config.DatabaseConfig{
				Path:    ":memory:",
				WALMode: false,
			},
			Auth: config.AuthConfig{
				JWT:     config.JWTConfig{Secret: "test-secret", ExpiresHours: 24},
				Session: config.SessionConfig{TimeoutMinutes: 30},
			},
		},
		Orchestrator: config.OrchestratorConfig{Port: 9090},
		Probe: config.ProbeMonitorConfig{
			Port:                8085,
			CheckInterval:       "30s",
			AlertInterval:       "5m",
			CleanupInterval:     "24h",
			ResultRetention:     "7d",
			AlertRetention:      "30d",
			EnableNotifications: false,
			MaxConcurrentProbes: 10,
		},
		Snap: config.SnapConfig{
			Port:          8086,
			RepoDir:       "./tmp",
			TempDir:       "./tmp",
			MaxParallel:   4,
			RateLimit:     "50MB/s",
			ScrubInterval: "24h",
		},
	}
	cfg.Snap.DefaultRetention.Daily = 7
	cfg.Snap.DefaultRetention.Weekly = 4
	cfg.Snap.DefaultRetention.Monthly = 12

	db, err := database.NewDB(cfg)
	require.NoError(t, err)
	t.Cleanup(func() {
		_ = db.Close()
	})

	handler := NewServiceHandler(db)
	repo := db.ServiceRepository()

	services := []*database.Service{
		{
			Name:     "alpha",
			Image:    "nginx:latest",
			Port:     8080,
			Replicas: 1,
			Status:   "running",
		},
		{
			Name:     "beta",
			Image:    "redis:alpine",
			Port:     6379,
			Replicas: 1,
			Status:   "stopped",
		},
		{
			Name:     "gamma",
			Image:    "busybox",
			Port:     9000,
			Replicas: 2,
			Status:   "error",
		},
	}

	for _, svc := range services {
		require.NoError(t, repo.Create(svc))
		time.Sleep(5 * time.Millisecond)
	}

	// Perform an extra update to ensure a deterministic most recent service
	updated, err := repo.GetByName("gamma")
	require.NoError(t, err)
	updated.Status = "error"
	require.NoError(t, repo.Update(updated))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/services/summary", nil)

	handler.GetServiceSummary(c)

	require.Equal(t, http.StatusOK, w.Code)

	var response ServiceSummaryResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, 3, response.Counts.Total)
	assert.Equal(t, 1, response.Counts.Running)
	assert.Equal(t, 1, response.Counts.Stopped)
	assert.Equal(t, 1, response.Counts.Error)
	assert.LessOrEqual(t, len(response.Recent), 5)
	require.NotNil(t, response.LastUpdated)
	assert.WithinDuration(t, time.Now().UTC(), *response.LastUpdated, time.Minute)
	require.NotEmpty(t, response.Recent)
	assert.Equal(t, updated.ID, response.Recent[0].ID)
	assert.WithinDuration(t, time.Now().UTC(), response.GeneratedAt, time.Minute)
}
