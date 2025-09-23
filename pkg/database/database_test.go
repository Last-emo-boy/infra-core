package database

import (
	"testing"
	"time"

	"github.com/last-emo-boy/infra-core/pkg/config"
)

func createTestDB(t *testing.T) *DB {
	cfg := &config.Config{
		Console: config.ConsoleConfig{
			Database: config.DatabaseConfig{
				Path:    ":memory:",
				WALMode: true,
				Timeout: "30s",
			},
		},
	}

	db, err := NewDB(cfg)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	return db
}

func TestNewDB(t *testing.T) {
	cfg := &config.Config{
		Console: config.ConsoleConfig{
			Database: config.DatabaseConfig{
				Path:    ":memory:",
				WALMode: true,
				Timeout: "30s",
			},
		},
	}

	db, err := NewDB(cfg)
	if err != nil {
		t.Errorf("Failed to create database: %v", err)
	}

	if db == nil {
		t.Error("Database should not be nil")
	}

	// Clean up
	db.Close()
}

func TestInitSchema(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Schema should already be initialized in NewDB
	// Try to query a table to verify schema exists
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM users")
	if err != nil {
		t.Errorf("Failed to query users table: %v", err)
	}

	// Check other core tables exist
	tables := []string{"services", "deployments", "routes", "certificates"}
	for _, table := range tables {
		err := db.Get(&count, "SELECT COUNT(*) FROM "+table)
		if err != nil {
			t.Errorf("Failed to query %s table: %v", table, err)
		}
	}
}

func TestUserOperations(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	userRepo := db.UserRepository()

	// Create a test user
	user := &User{
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hashedpassword",
		Role:         "user",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err := userRepo.Create(user)
	if err != nil {
		t.Errorf("Failed to create user: %v", err)
	}

	if user.ID == 0 {
		t.Error("User ID should be set after creation")
	}

	// Retrieve user by username
	retrievedUser, err := userRepo.GetByUsername(user.Username)
	if err != nil {
		t.Errorf("Failed to retrieve user: %v", err)
	}

	if retrievedUser.Username != user.Username {
		t.Errorf("Expected username '%s', got '%s'", user.Username, retrievedUser.Username)
	}

	// Update user
	user.Email = "updated@example.com"
	err = userRepo.Update(user)
	if err != nil {
		t.Errorf("Failed to update user: %v", err)
	}

	// Verify update
	updatedUser, err := userRepo.GetByID(user.ID)
	if err != nil {
		t.Errorf("Failed to retrieve updated user: %v", err)
	}

	if updatedUser.Email != "updated@example.com" {
		t.Errorf("Expected email 'updated@example.com', got '%s'", updatedUser.Email)
	}

	// List users
	users, err := userRepo.List()
	if err != nil {
		t.Errorf("Failed to list users: %v", err)
	}

	if len(users) == 0 {
		t.Error("Should have at least one user")
	}
}

func TestServiceOperations(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	serviceRepo := db.ServiceRepository()

	// Create a test service
	service := &Service{
		ID:          "test-service-1",
		Name:        "test-service",
		Image:       "nginx:latest",
		Port:        8080,
		Replicas:    1,
		Status:      "running",
		Environment: map[string]string{"ENV": "test"},
		Command:     []string{"/bin/sh"},
		Args:        []string{"-c", "echo hello"},
		YAMLConfig:  "apiVersion: v1\nkind: Service",
		Version:     1,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := serviceRepo.Create(service)
	if err != nil {
		t.Errorf("Failed to create service: %v", err)
	}

	// Retrieve service
	retrievedService, err := serviceRepo.GetByID(service.ID)
	if err != nil {
		t.Errorf("Failed to retrieve service: %v", err)
	}

	if retrievedService.Name != service.Name {
		t.Errorf("Expected service name '%s', got '%s'", service.Name, retrievedService.Name)
	}

	// Update service status
	service.Status = "stopped"
	err = serviceRepo.Update(service)
	if err != nil {
		t.Errorf("Failed to update service: %v", err)
	}

	// Verify status update
	updatedService, err := serviceRepo.GetByID(service.ID)
	if err != nil {
		t.Errorf("Failed to retrieve updated service: %v", err)
	}

	if updatedService.Status != "stopped" {
		t.Errorf("Expected status 'stopped', got '%s'", updatedService.Status)
	}

	// List services
	services, err := serviceRepo.List()
	if err != nil {
		t.Errorf("Failed to list services: %v", err)
	}

	if len(services) == 0 {
		t.Error("Should have at least one service")
	}
}

func TestHealthCheck(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Check database health
	err := db.HealthCheck()
	if err != nil {
		t.Errorf("Database health check failed: %v", err)
	}
}

func TestGetStats(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// Get database statistics
	stats, err := db.GetStats()
	if err != nil {
		t.Errorf("Failed to get database stats: %v", err)
	}

	if stats == nil {
		t.Error("Database stats should not be nil")
	}

	// Check that stats is not empty
	if len(stats) == 0 {
		t.Error("Database stats should not be empty")
	}

	// Check that some expected keys exist (using actual key names from GetStats)
	expectedKeys := []string{"users_count", "services_count", "deployments_count", "database_size_bytes"}
	for _, key := range expectedKeys {
		if _, exists := stats[key]; !exists {
			t.Errorf("Expected stats key '%s' not found", key)
		}
	}

	// Check for numeric values
	for key, value := range stats {
		switch key {
		case "users_count", "services_count", "deployments_count", "database_size_bytes":
			if _, ok := value.(int); !ok {
				t.Errorf("Expected %s to be an integer", key)
			}
		case "journal_mode":
			if _, ok := value.(string); !ok {
				t.Errorf("Expected %s to be a string", key)
			}
		}
	}
}

func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}