package main

import (
	"testing"

	"github.com/last-emo-boy/infra-core/pkg/config"
	"github.com/last-emo-boy/infra-core/pkg/database"
)

func TestIntegration(t *testing.T) {
	// Test configuration loading
	cfg := &config.Config{
		Console: config.ConsoleConfig{
			Database: config.DatabaseConfig{
				Path:    ":memory:",
				WALMode: true,
				Timeout: "30s",
			},
		},
	}

	// Test database creation
	db, err := database.NewDB(cfg)
	if err != nil {
		t.Errorf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Test database health
	err = db.HealthCheck()
	if err != nil {
		t.Errorf("Database health check failed: %v", err)
	}

	// Test database stats
	stats, err := db.GetStats()
	if err != nil {
		t.Errorf("Failed to get database stats: %v", err)
	}

	if stats == nil {
		t.Error("Database stats should not be nil")
	}

	t.Logf("Integration test passed - Config loaded, DB created, health check OK, stats retrieved")
}