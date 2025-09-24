package database

import (
	"fmt"
	"strings"
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

// RegisteredServiceRepository Tests

func TestRegisteredServiceRepository_Create(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.RegisteredServiceRepository()
	
	description := "Test service description"
	callbackURL := "http://localhost:8080/callback"
	icon := "service-icon.png"
	healthURL := "http://localhost:8080/health"
	
	service := &RegisteredService{
		Name:         "test-service",
		DisplayName:  "Test Service",
		Description:  &description,
		ServiceURL:   "http://localhost:8080",
		CallbackURL:  &callbackURL,
		Icon:         &icon,
		Category:     "api",
		IsPublic:     true,
		RequiredRole: "user",
		Status:       "active",
		HealthURL:    &healthURL,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err := repo.Create(service)
	if err != nil {
		t.Fatalf("Failed to create registered service: %v", err)
	}

	// Verify the service was created with an ID
	if service.ID == "" {
		t.Error("Expected service ID to be set after creation")
	}
}

func TestRegisteredServiceRepository_GetByID(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.RegisteredServiceRepository()
	
	// Create a test service
	description := "Test service description"
	service := &RegisteredService{
		Name:         "test-service",
		DisplayName:  "Test Service",
		Description:  &description,
		ServiceURL:   "http://localhost:8080",
		Category:     "api",
		IsPublic:     true,
		RequiredRole: "user",
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	
	err := repo.Create(service)
	if err != nil {
		t.Fatalf("Failed to create test service: %v", err)
	}

	// Get the service by ID
	retrieved, err := repo.GetByID(service.ID)
	if err != nil {
		t.Fatalf("Failed to get service by ID: %v", err)
	}

	if retrieved.Name != service.Name {
		t.Errorf("Expected name %s, got %s", service.Name, retrieved.Name)
	}
	if retrieved.DisplayName != service.DisplayName {
		t.Errorf("Expected display name %s, got %s", service.DisplayName, retrieved.DisplayName)
	}
}

func TestRegisteredServiceRepository_GetByName(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.RegisteredServiceRepository()
	
	// Create a test service
	description := "Unique service description"
	service := &RegisteredService{
		Name:         "unique-service",
		DisplayName:  "Unique Service",
		Description:  &description,
		ServiceURL:   "http://localhost:8080",
		Category:     "api",
		IsPublic:     false,
		RequiredRole: "admin",
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	
	err := repo.Create(service)
	if err != nil {
		t.Fatalf("Failed to create test service: %v", err)
	}

	// Get the service by name
	retrieved, err := repo.GetByName("unique-service")
	if err != nil {
		t.Fatalf("Failed to get service by name: %v", err)
	}

	if retrieved.ID != service.ID {
		t.Errorf("Expected ID %s, got %s", service.ID, retrieved.ID)
	}
	if retrieved.RequiredRole != "admin" {
		t.Errorf("Expected required role admin, got %s", retrieved.RequiredRole)
	}
}

func TestRegisteredServiceRepository_List(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.RegisteredServiceRepository()
	
	// Create multiple test services
	desc1 := "First service"
	desc2 := "Second service"
	services := []*RegisteredService{
		{
			Name:         "service-1",
			DisplayName:  "Service One",
			Description:  &desc1,
			ServiceURL:   "http://localhost:8081",
			Category:     "api",
			IsPublic:     true,
			RequiredRole: "user",
			Status:       "active",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			Name:         "service-2",
			DisplayName:  "Service Two",
			Description:  &desc2,
			ServiceURL:   "http://localhost:8082",
			Category:     "worker",
			IsPublic:     false,
			RequiredRole: "admin",
			Status:       "inactive",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	}

	for _, service := range services {
		err := repo.Create(service)
		if err != nil {
			t.Fatalf("Failed to create test service: %v", err)
		}
	}

	// List all services
	retrieved, err := repo.List()
	if err != nil {
		t.Fatalf("Failed to list services: %v", err)
	}

	if len(retrieved) < 2 {
		t.Errorf("Expected at least 2 services, got %d", len(retrieved))
	}
}

func TestRegisteredServiceRepository_ListByCategory(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.RegisteredServiceRepository()
	
	// Create services in different categories
	desc1 := "API service"
	desc2 := "Worker service"
	services := []*RegisteredService{
		{
			Name:         "api-service-1",
			DisplayName:  "API Service 1",
			Description:  &desc1,
			ServiceURL:   "http://localhost:8081",
			Category:     "api",
			IsPublic:     true,
			RequiredRole: "user",
			Status:       "active",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			Name:         "worker-service-1",
			DisplayName:  "Worker Service 1",
			Description:  &desc2,
			ServiceURL:   "http://localhost:8082",
			Category:     "worker",
			IsPublic:     false,
			RequiredRole: "admin",
			Status:       "active",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	}

	for _, service := range services {
		err := repo.Create(service)
		if err != nil {
			t.Fatalf("Failed to create test service: %v", err)
		}
	}

	// List services by category
	apiServices, err := repo.ListByCategory("api")
	if err != nil {
		t.Fatalf("Failed to list API services: %v", err)
	}

	if len(apiServices) < 1 {
		t.Error("Expected at least 1 API service")
	}

	for _, service := range apiServices {
		if service.Category != "api" {
			t.Errorf("Expected category 'api', got '%s'", service.Category)
		}
	}
}

func TestRegisteredServiceRepository_Update(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.RegisteredServiceRepository()
	
	// Create a test service
	originalDesc := "Original description"
	service := &RegisteredService{
		Name:         "update-service",
		DisplayName:  "Update Service",
		Description:  &originalDesc,
		ServiceURL:   "http://localhost:8080",
		Category:     "api",
		IsPublic:     true,
		RequiredRole: "user",
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	
	err := repo.Create(service)
	if err != nil {
		t.Fatalf("Failed to create test service: %v", err)
	}

	// Update the service
	updatedDesc := "Updated description"
	service.Description = &updatedDesc
	service.DisplayName = "Updated Service Name"
	service.Status = "maintenance"
	service.IsPublic = false
	service.UpdatedAt = time.Now()

	err = repo.Update(service)
	if err != nil {
		t.Fatalf("Failed to update service: %v", err)
	}

	// Verify the update
	retrieved, err := repo.GetByID(service.ID)
	if err != nil {
		t.Fatalf("Failed to get updated service: %v", err)
	}

	if retrieved.Description == nil || *retrieved.Description != "Updated description" {
		t.Errorf("Expected updated description, got %v", retrieved.Description)
	}
	if retrieved.DisplayName != "Updated Service Name" {
		t.Errorf("Expected updated display name, got %s", retrieved.DisplayName)
	}
	if retrieved.Status != "maintenance" {
		t.Errorf("Expected status 'maintenance', got %s", retrieved.Status)
	}
	if retrieved.IsPublic != false {
		t.Errorf("Expected IsPublic false, got %t", retrieved.IsPublic)
	}
}

func TestRegisteredServiceRepository_UpdateHealthStatus(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.RegisteredServiceRepository()
	
	// Create a test service
	description := "Health test service"
	service := &RegisteredService{
		Name:         "health-service",
		DisplayName:  "Health Service",
		Description:  &description,
		ServiceURL:   "http://localhost:8080",
		Category:     "api",
		IsPublic:     true,
		RequiredRole: "user",
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	
	err := repo.Create(service)
	if err != nil {
		t.Fatalf("Failed to create test service: %v", err)
	}

	// Update health status to healthy
	err = repo.UpdateHealthStatus(service.ID, true)
	if err != nil {
		t.Fatalf("Failed to update health status: %v", err)
	}

	// Verify the health status update
	retrieved, err := repo.GetByID(service.ID)
	if err != nil {
		t.Fatalf("Failed to get updated service: %v", err)
	}

	if retrieved.LastHealthy == nil {
		t.Error("Expected LastHealthy to be set")
	}

	// Update health status to unhealthy
	err = repo.UpdateHealthStatus(service.ID, false)
	if err != nil {
		t.Fatalf("Failed to update health status to unhealthy: %v", err)
	}
}

func TestRegisteredServiceRepository_Delete(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.RegisteredServiceRepository()
	
	// Create a test service
	description := "Service to be deleted"
	service := &RegisteredService{
		Name:         "delete-service",
		DisplayName:  "Delete Service",
		Description:  &description,
		ServiceURL:   "http://localhost:8080",
		Category:     "api",
		IsPublic:     true,
		RequiredRole: "user",
		Status:       "active",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	
	err := repo.Create(service)
	if err != nil {
		t.Fatalf("Failed to create test service: %v", err)
	}

	// Delete the service
	err = repo.Delete(service.ID)
	if err != nil {
		t.Fatalf("Failed to delete service: %v", err)
	}

	// Verify the service is deleted
	_, err = repo.GetByID(service.ID)
	if err == nil {
		t.Error("Expected error when getting deleted service, but got nil")
	}
}

// SSOSessionRepository Tests

func TestSSOSessionRepository_Create(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.SSOSessionRepository()
	
	session := &SSOSession{
		UserID:    1,
		TokenHash: "hashed_token_123",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IPAddress: "192.168.1.1",
		UserAgent: "Mozilla/5.0",
		IsActive:  true,
		LastUsed:  time.Now(),
		CreatedAt: time.Now(),
	}

	err := repo.Create(session)
	if err != nil {
		t.Fatalf("Failed to create SSO session: %v", err)
	}

	// Verify the session was created with an ID
	if session.ID == "" {
		t.Error("Expected session ID to be set after creation")
	}
}

func TestSSOSessionRepository_GetByTokenHash(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.SSOSessionRepository()
	
	// Create a test session
	session := &SSOSession{
		UserID:    2,
		TokenHash: "unique_token_hash_456",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IPAddress: "192.168.1.2",
		UserAgent: "Chrome/91.0",
		IsActive:  true,
		LastUsed:  time.Now(),
		CreatedAt: time.Now(),
	}
	
	err := repo.Create(session)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Get the session by token hash
	retrieved, err := repo.GetByTokenHash("unique_token_hash_456")
	if err != nil {
		t.Fatalf("Failed to get session by token hash: %v", err)
	}

	if retrieved.UserID != session.UserID {
		t.Errorf("Expected UserID %d, got %d", session.UserID, retrieved.UserID)
	}
	if retrieved.IPAddress != session.IPAddress {
		t.Errorf("Expected IP address %s, got %s", session.IPAddress, retrieved.IPAddress)
	}
}

func TestSSOSessionRepository_UpdateLastUsed(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.SSOSessionRepository()
	
	// Create a test session
	session := &SSOSession{
		UserID:    3,
		TokenHash: "update_test_token",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IPAddress: "192.168.1.3",
		UserAgent: "Safari/14.0",
		IsActive:  true,
		LastUsed:  time.Now().Add(-1 * time.Hour), // 1 hour ago
		CreatedAt: time.Now().Add(-2 * time.Hour), // 2 hours ago
	}
	
	err := repo.Create(session)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Update last used
	err = repo.UpdateLastUsed(session.ID)
	if err != nil {
		t.Fatalf("Failed to update last used: %v", err)
	}

	// Verify the update
	retrieved, err := repo.GetByTokenHash("update_test_token")
	if err != nil {
		t.Fatalf("Failed to get updated session: %v", err)
	}

	// LastUsed should be more recent than the original
	if !retrieved.LastUsed.After(session.LastUsed) {
		t.Error("Expected LastUsed to be updated to a more recent time")
	}
}

func TestSSOSessionRepository_Invalidate(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.SSOSessionRepository()
	
	// Create a test session
	session := &SSOSession{
		UserID:    4,
		TokenHash: "invalidate_test_token",
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IPAddress: "192.168.1.4",
		UserAgent: "Firefox/89.0",
		IsActive:  true,
		LastUsed:  time.Now(),
		CreatedAt: time.Now(),
	}
	
	err := repo.Create(session)
	if err != nil {
		t.Fatalf("Failed to create test session: %v", err)
	}

	// Invalidate the session
	err = repo.Invalidate(session.ID)
	if err != nil {
		t.Fatalf("Failed to invalidate session: %v", err)
	}

	// Verify the session is invalidated by checking its IsActive status
	// Note: The session record still exists but IsActive should be false
	retrieved, err := repo.GetByTokenHash("invalidate_test_token")
	if err != nil {
		// This is expected if the implementation filters out inactive sessions
		// Let's check if this is the case
		t.Logf("Session not found after invalidation (expected behavior): %v", err)
		return
	}

	if retrieved.IsActive {
		t.Error("Expected session to be inactive after invalidation")
	}
}

func TestSSOSessionRepository_InvalidateUserSessions(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.SSOSessionRepository()
	
	// Create multiple sessions for the same user
	userID := 5
	sessions := []*SSOSession{
		{
			UserID:    userID,
			TokenHash: "user_session_1",
			ExpiresAt: time.Now().Add(24 * time.Hour),
			IPAddress: "192.168.1.5",
			UserAgent: "Chrome/91.0",
			IsActive:  true,
			LastUsed:  time.Now(),
			CreatedAt: time.Now(),
		},
		{
			UserID:    userID,
			TokenHash: "user_session_2",
			ExpiresAt: time.Now().Add(24 * time.Hour),
			IPAddress: "192.168.1.6",
			UserAgent: "Firefox/89.0",
			IsActive:  true,
			LastUsed:  time.Now(),
			CreatedAt: time.Now(),
		},
	}

	for _, session := range sessions {
		err := repo.Create(session)
		if err != nil {
			t.Fatalf("Failed to create test session: %v", err)
		}
	}

	// Invalidate all user sessions
	err := repo.InvalidateUserSessions(userID)
	if err != nil {
		t.Fatalf("Failed to invalidate user sessions: %v", err)
	}

	// Verify all sessions are invalidated
	// Note: Sessions may not be retrievable if implementation filters inactive sessions
	for _, tokenHash := range []string{"user_session_1", "user_session_2"} {
		retrieved, err := repo.GetByTokenHash(tokenHash)
		if err != nil {
			// This is expected if the implementation filters out inactive sessions
			t.Logf("Session %s not found after invalidation (expected behavior): %v", tokenHash, err)
			continue
		}

		if retrieved.IsActive {
			t.Errorf("Expected session %s to be inactive after user invalidation", tokenHash)
		}
	}
}

func TestSSOSessionRepository_CleanupExpiredSessions(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.SSOSessionRepository()
	
	// Create expired and active sessions
	sessions := []*SSOSession{
		{
			UserID:    6,
			TokenHash: "expired_session_1",
			ExpiresAt: time.Now().Add(-1 * time.Hour), // Expired
			IPAddress: "192.168.1.7",
			UserAgent: "Chrome/91.0",
			IsActive:  true,
			LastUsed:  time.Now().Add(-2 * time.Hour),
			CreatedAt: time.Now().Add(-3 * time.Hour),
		},
		{
			UserID:    7,
			TokenHash: "active_session_1",
			ExpiresAt: time.Now().Add(24 * time.Hour), // Active
			IPAddress: "192.168.1.8",
			UserAgent: "Firefox/89.0",
			IsActive:  true,
			LastUsed:  time.Now(),
			CreatedAt: time.Now(),
		},
	}

	for _, session := range sessions {
		err := repo.Create(session)
		if err != nil {
			t.Fatalf("Failed to create test session: %v", err)
		}
	}

	// Cleanup expired sessions
	err := repo.CleanupExpiredSessions()
	if err != nil {
		t.Fatalf("Failed to cleanup expired sessions: %v", err)
	}

	// Verify expired session is cleaned up (should error when trying to get it)
	_, err = repo.GetByTokenHash("expired_session_1")
	if err == nil {
		t.Error("Expected error when getting expired session after cleanup, but got nil")
	}

	// Verify active session still exists
	_, err = repo.GetByTokenHash("active_session_1")
	if err != nil {
		t.Errorf("Expected active session to still exist after cleanup, but got error: %v", err)
	}
}

// UserServicePermissionRepository Tests

func TestUserServicePermissionRepository_Grant(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.UserServicePermissionRepository()
	
	// Grant permission with no expiration
	userID := 1
	serviceID := "test-service-123"
	grantedBy := 2
	
	err := repo.Grant(userID, serviceID, grantedBy, nil)
	if err != nil {
		t.Fatalf("Failed to grant permission: %v", err)
	}

	// Verify permission was granted
	hasPermission, err := repo.CheckPermission(userID, serviceID)
	if err != nil {
		t.Fatalf("Failed to check permission: %v", err)
	}

	if !hasPermission {
		t.Error("Expected user to have permission after granting")
	}

	// Grant permission with expiration
	expiresAt := time.Now().Add(24 * time.Hour)
	err = repo.Grant(userID, "service-with-expiry", grantedBy, &expiresAt)
	if err != nil {
		t.Fatalf("Failed to grant permission with expiry: %v", err)
	}
}

func TestUserServicePermissionRepository_Revoke(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.UserServicePermissionRepository()
	
	// Grant permission first
	userID := 2
	serviceID := "test-service-456"
	grantedBy := 3
	
	err := repo.Grant(userID, serviceID, grantedBy, nil)
	if err != nil {
		t.Fatalf("Failed to grant permission: %v", err)
	}

	// Verify permission exists
	hasPermission, err := repo.CheckPermission(userID, serviceID)
	if err != nil {
		t.Fatalf("Failed to check permission: %v", err)
	}
	if !hasPermission {
		t.Fatal("Expected permission to exist before revoking")
	}

	// Revoke permission
	err = repo.Revoke(userID, serviceID)
	if err != nil {
		t.Fatalf("Failed to revoke permission: %v", err)
	}

	// Verify permission was revoked
	hasPermission, err = repo.CheckPermission(userID, serviceID)
	if err != nil {
		t.Fatalf("Failed to check permission after revoke: %v", err)
	}

	if hasPermission {
		t.Error("Expected user to not have permission after revoking")
	}
}

func TestUserServicePermissionRepository_CheckPermission(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.UserServicePermissionRepository()
	
	userID := 3
	serviceID := "check-permission-service"
	grantedBy := 4

	// Check permission before granting (should be false)
	hasPermission, err := repo.CheckPermission(userID, serviceID)
	if err != nil {
		t.Fatalf("Failed to check permission: %v", err)
	}
	if hasPermission {
		t.Error("Expected user to not have permission initially")
	}

	// Grant permission
	err = repo.Grant(userID, serviceID, grantedBy, nil)
	if err != nil {
		t.Fatalf("Failed to grant permission: %v", err)
	}

	// Check permission after granting (should be true)
	hasPermission, err = repo.CheckPermission(userID, serviceID)
	if err != nil {
		t.Fatalf("Failed to check permission after granting: %v", err)
	}
	if !hasPermission {
		t.Error("Expected user to have permission after granting")
	}
}

func TestUserServicePermissionRepository_ListUserServices(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	// First create some registered services
	regRepo := db.RegisteredServiceRepository()
	userPermRepo := db.UserServicePermissionRepository()
	
	userID := 4
	grantedBy := 5
	
	// Create registered services first
	desc1 := "Service 1 description"
	desc2 := "Service 2 description"
	services := []*RegisteredService{
		{
			Name:         "permission-service-1",
			DisplayName:  "Permission Service 1",
			Description:  &desc1,
			ServiceURL:   "http://localhost:8081",
			Category:     "api",
			IsPublic:     true,
			RequiredRole: "user",
			Status:       "active",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			Name:         "permission-service-2",
			DisplayName:  "Permission Service 2",
			Description:  &desc2,
			ServiceURL:   "http://localhost:8082",
			Category:     "worker",
			IsPublic:     false,
			RequiredRole: "admin",
			Status:       "active",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	}

	for _, service := range services {
		err := regRepo.Create(service)
		if err != nil {
			t.Fatalf("Failed to create registered service: %v", err)
		}
		
		// Grant permission for this service
		err = userPermRepo.Grant(userID, service.ID, grantedBy, nil)
		if err != nil {
			t.Fatalf("Failed to grant permission for %s: %v", service.ID, err)
		}
	}

	// List user services
	userServices, err := userPermRepo.ListUserServices(userID)
	if err != nil {
		t.Fatalf("Failed to list user services: %v", err)
	}

	if len(userServices) < 2 {
		t.Errorf("Expected at least 2 services, got %d", len(userServices))
	}

	// Verify the returned services contain our test services
	serviceMap := make(map[string]bool)
	for _, service := range userServices {
		serviceMap[service.ID] = true
	}

	for _, service := range services {
		if !serviceMap[service.ID] {
			t.Errorf("Expected service %s to be in user's services list", service.ID)
		}
	}
}

// ServiceHealthCheckRepository tests
func TestServiceHealthCheckRepository_Record(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.ServiceHealthCheckRepository()
	
	check := &ServiceHealthCheck{
		ServiceID:     "health-service-1",
		IsHealthy:     true,
		ResponseTime:  150,
		ErrorMessage:  nil,
		CheckedAt:     time.Now(),
	}

	err := repo.Record(check)
	if err != nil {
		t.Fatalf("Failed to record health check: %v", err)
	}

	// Verify the check has an ID after recording
	if check.ID == 0 {
		t.Error("Expected health check to have an ID after recording")
	}
}

func TestServiceHealthCheckRepository_GetLatest(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.ServiceHealthCheckRepository()
	
	serviceID := "health-service-2"
	
	// Record multiple checks with different timestamps
	oldCheck := &ServiceHealthCheck{
		ServiceID:     serviceID,
		IsHealthy:     false,
		ResponseTime:  5000,
		CheckedAt:     time.Now().Add(-time.Hour),
	}
	
	recentCheck := &ServiceHealthCheck{
		ServiceID:     serviceID,
		IsHealthy:     true,
		ResponseTime:  100,
		CheckedAt:     time.Now(),
	}

	err := repo.Record(oldCheck)
	if err != nil {
		t.Fatalf("Failed to record old health check: %v", err)
	}

	err = repo.Record(recentCheck)
	if err != nil {
		t.Fatalf("Failed to record recent health check: %v", err)
	}

	// Get latest check
	latestCheck, err := repo.GetLatest(serviceID)
	if err != nil {
		t.Fatalf("Failed to get latest health check: %v", err)
	}

	if latestCheck == nil {
		t.Fatal("Expected to get a health check result")
	}

	if !latestCheck.IsHealthy {
		t.Errorf("Expected latest check to be healthy, got %t", latestCheck.IsHealthy)
	}

	if latestCheck.ResponseTime != 100 {
		t.Errorf("Expected latest check response time to be 100, got %d", latestCheck.ResponseTime)
	}
}

func TestServiceHealthCheckRepository_GetHistory(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.ServiceHealthCheckRepository()
	
	serviceID := "health-service-3"
	
	// Record multiple checks over time
	checkTimes := []time.Time{
		time.Now().Add(-3 * time.Hour),
		time.Now().Add(-2 * time.Hour),
		time.Now().Add(-1 * time.Hour),
		time.Now(),
	}

	for i, checkTime := range checkTimes {
		check := &ServiceHealthCheck{
			ServiceID:     serviceID,
			IsHealthy:     i%2 == 0, // alternate healthy/unhealthy
			ResponseTime:  100 + i*50,
			CheckedAt:     checkTime,
		}
		err := repo.Record(check)
		if err != nil {
			t.Fatalf("Failed to record health check %d: %v", i, err)
		}
	}

	// Get checks history (last 3)
	history, err := repo.GetHistory(serviceID, 3)
	if err != nil {
		t.Fatalf("Failed to get checks history: %v", err)
	}

	if len(history) != 3 {
		t.Errorf("Expected 3 checks in history, got %d", len(history))
	}

	// Verify they are in descending order (most recent first)
	for i := 1; i < len(history); i++ {
		if history[i-1].CheckedAt.Before(history[i].CheckedAt) {
			t.Error("Expected checks to be in descending order by CheckedAt")
			break
		}
	}
}

// RouteRepository tests
func TestRouteRepository_Create(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.RouteRepository()
	
	route := &Route{
		Host:              "api.example.com",
		PathPrefix:        "/api/v1/users",
		UpstreamServiceID: nil,
		UpstreamURL:       stringPtr("http://localhost:8080"),
		TLSCertID:         nil,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	err := repo.Create(route)
	if err != nil {
		t.Fatalf("Failed to create route: %v", err)
	}

	// Verify the route has an ID after creating
	if route.ID == "" {
		t.Error("Expected route to have an ID after creating")
	}
}

func TestRouteRepository_GetByID(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.RouteRepository()
	
	// Create a test route
	route := &Route{
		Host:       "test.example.com",
		PathPrefix: "/api/test",
		UpstreamURL: stringPtr("http://localhost:8081"),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	
	err := repo.Create(route)
	if err != nil {
		t.Fatalf("Failed to create test route: %v", err)
	}

	// Get the route by ID
	retrieved, err := repo.GetByID(route.ID)
	if err != nil {
		t.Fatalf("Failed to get route by ID: %v", err)
	}

	if retrieved.Host != route.Host {
		t.Errorf("Expected host %s, got %s", route.Host, retrieved.Host)
	}
	if retrieved.PathPrefix != route.PathPrefix {
		t.Errorf("Expected path prefix %s, got %s", route.PathPrefix, retrieved.PathPrefix)
	}
}

func TestRouteRepository_List(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.RouteRepository()
	
	// Create multiple routes
	routes := []*Route{
		{
			Host:       "api1.example.com",
			PathPrefix: "/api/v1",
			UpstreamURL: stringPtr("http://localhost:8081"),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
		{
			Host:       "api2.example.com",
			PathPrefix: "/api/v2",
			UpstreamURL: stringPtr("http://localhost:8082"),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		},
	}

	for _, route := range routes {
		err := repo.Create(route)
		if err != nil {
			t.Fatalf("Failed to create route: %v", err)
		}
	}

	// List all routes
	retrievedRoutes, err := repo.List()
	if err != nil {
		t.Fatalf("Failed to list routes: %v", err)
	}

	if len(retrievedRoutes) < 2 {
		t.Errorf("Expected at least 2 routes, got %d", len(retrievedRoutes))
	}
}

func TestRouteRepository_Update(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.RouteRepository()
	
	// Create a route first
	route := &Route{
		Host:       "update.example.com",
		PathPrefix: "/api/v1",
		UpstreamURL: stringPtr("http://localhost:8080"),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := repo.Create(route)
	if err != nil {
		t.Fatalf("Failed to create route: %v", err)
	}

	// Update the route
	route.PathPrefix = "/api/v2"
	route.UpstreamURL = stringPtr("http://localhost:8090")
	route.UpdatedAt = time.Now()

	err = repo.Update(route)
	if err != nil {
		t.Fatalf("Failed to update route: %v", err)
	}

	// Verify the route was updated
	updated, err := repo.GetByID(route.ID)
	if err != nil {
		t.Fatalf("Failed to get updated route: %v", err)
	}

	if updated.PathPrefix != "/api/v2" {
		t.Errorf("Expected path prefix to be updated to '/api/v2', got '%s'", updated.PathPrefix)
	}

	if updated.UpstreamURL == nil || *updated.UpstreamURL != "http://localhost:8090" {
		t.Errorf("Expected upstream URL to be updated")
	}
}

func TestRouteRepository_Delete(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.RouteRepository()
	
	// Create a route first
	route := &Route{
		Host:       "delete.example.com",
		PathPrefix: "/api/temp",
		UpstreamURL: stringPtr("http://localhost:8080"),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	err := repo.Create(route)
	if err != nil {
		t.Fatalf("Failed to create route: %v", err)
	}

	// Verify route exists
	_, err = repo.GetByID(route.ID)
	if err != nil {
		t.Fatalf("Failed to get route before delete: %v", err)
	}

	// Delete the route
	err = repo.Delete(route.ID)
	if err != nil {
		t.Fatalf("Failed to delete route: %v", err)
	}

	// Verify route was deleted
	_, err = repo.GetByID(route.ID)
	if err == nil {
		t.Error("Expected error when getting deleted route, but got nil")
	}
}

// MetricRepository tests
func TestMetricRepository_Insert(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.MetricRepository()
	
	labelsJSON := `{"endpoint": "/api/users", "method": "GET"}`
	metric := &Metric{
		Timestamp:   time.Now(),
		ScopeType:   "service",
		ScopeID:     "metric-service-1",
		MetricName:  "request_count",
		MetricValue: 125.5,
		Labels:      &labelsJSON,
		CreatedAt:   time.Now(),
	}

	err := repo.Insert(metric)
	if err != nil {
		t.Fatalf("Failed to insert metric: %v", err)
	}

	// Note: The current implementation doesn't return the generated ID
	// We can verify insertion by querying for the metric
	from := metric.Timestamp.Add(-1 * time.Minute)
	to := metric.Timestamp.Add(1 * time.Minute)
	
	metrics, err := repo.Query(metric.ScopeType, metric.ScopeID, metric.MetricName, from, to, 10)
	if err != nil {
		t.Fatalf("Failed to query inserted metric: %v", err)
	}

	if len(metrics) == 0 {
		t.Error("Expected to find the inserted metric")
	}

	// Verify the metric data
	found := false
	for _, m := range metrics {
		if m.MetricValue == 125.5 && m.ScopeID == "metric-service-1" {
			found = true
			break
		}
	}

	if !found {
		t.Error("Could not find the inserted metric with expected values")
	}
}

func TestMetricRepository_Query(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.MetricRepository()
	
	scopeType := "service"
	scopeID := "metric-service-2"
	metricName := "response_time"
	
	// Insert multiple metrics over time
	baseTime := time.Now().Add(-time.Hour)
	
	for i := 0; i < 5; i++ {
		labelsJSON := fmt.Sprintf(`{"endpoint": "/api/endpoint%d"}`, i)
		metric := &Metric{
			Timestamp:   baseTime.Add(time.Duration(i) * 10 * time.Minute),
			ScopeType:   scopeType,
			ScopeID:     scopeID,
			MetricName:  metricName,
			MetricValue: float64(100 + i*20),
			Labels:      &labelsJSON,
			CreatedAt:   time.Now(),
		}
		
		err := repo.Insert(metric)
		if err != nil {
			t.Fatalf("Failed to insert metric %d: %v", i, err)
		}
	}

	// Query metrics for the last 30 minutes
	from := time.Now().Add(-30 * time.Minute)
	to := time.Now()
	
	metrics, err := repo.Query(scopeType, scopeID, metricName, from, to, 10)
	if err != nil {
		t.Fatalf("Failed to query metrics: %v", err)
	}

	if len(metrics) == 0 {
		t.Error("Expected to get at least some metrics")
	}

	// Verify all metrics match the criteria
	for _, metric := range metrics {
		if metric.ScopeType != scopeType {
			t.Errorf("Expected scope type %s, got %s", scopeType, metric.ScopeType)
		}
		if metric.ScopeID != scopeID {
			t.Errorf("Expected scope ID %s, got %s", scopeID, metric.ScopeID)
		}
		if metric.MetricName != metricName {
			t.Errorf("Expected metric name %s, got %s", metricName, metric.MetricName)
		}
	}
}

func TestMetricRepository_GetByService(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.MetricRepository()
	
	serviceID := "metric-service-3"
	
	// Insert metrics for the service
	for i := 0; i < 3; i++ {
		labelsJSON := fmt.Sprintf(`{"host": "host-%d"}`, i)
		metric := &Metric{
			Timestamp:   time.Now().Add(time.Duration(-i) * time.Hour),
			ScopeType:   "service",
			ScopeID:     serviceID,
			MetricName:  "cpu_usage",
			MetricValue: float64(10 + i*5),
			Labels:      &labelsJSON,
			CreatedAt:   time.Now(),
		}
		
		err := repo.Insert(metric)
		if err != nil {
			t.Fatalf("Failed to insert metric %d: %v", i, err)
		}
	}

	// Get metrics by service
	metrics, err := repo.GetByService(serviceID, 10)
	if err != nil {
		t.Fatalf("Failed to get metrics by service: %v", err)
	}

	if len(metrics) < 3 {
		t.Errorf("Expected at least 3 metrics, got %d", len(metrics))
	}

	// Verify all metrics belong to the service
	for _, metric := range metrics {
		if metric.ScopeID != serviceID {
			t.Errorf("Expected scope ID %s, got %s", serviceID, metric.ScopeID)
		}
	}
}

func TestMetricRepository_DeleteOld(t *testing.T) {
	db := createTestDB(t)
	defer db.Close()

	repo := db.MetricRepository()
	
	// Insert old and recent metrics
	oldTime := time.Now().Add(-30 * 24 * time.Hour) // 30 days ago
	recentTime := time.Now().Add(-1 * time.Hour)    // 1 hour ago
	
	oldMetric := &Metric{
		Timestamp:   oldTime,
		ScopeType:   "service",
		ScopeID:     "metric-service-4",
		MetricName:  "memory_usage",
		MetricValue: 50.0,
		Labels:      stringPtr(`{"type": "old"}`),
		CreatedAt:   oldTime,
	}
	
	recentMetric := &Metric{
		Timestamp:   recentTime,
		ScopeType:   "service",
		ScopeID:     "metric-service-4",
		MetricName:  "memory_usage",
		MetricValue: 75.0,
		Labels:      stringPtr(`{"type": "recent"}`),
		CreatedAt:   recentTime,
	}

	err := repo.Insert(oldMetric)
	if err != nil {
		t.Fatalf("Failed to insert old metric: %v", err)
	}

	err = repo.Insert(recentMetric)
	if err != nil {
		t.Fatalf("Failed to insert recent metric: %v", err)
	}

	// Delete metrics older than 7 days
	err = repo.DeleteOld(7)
	if err != nil {
		t.Fatalf("Failed to delete old metrics: %v", err)
	}

	// Verify recent metric still exists by querying
	from := recentTime.Add(-1 * time.Hour)
	to := time.Now()
	metrics, err := repo.Query("service", "metric-service-4", "memory_usage", from, to, 10)
	if err != nil {
		t.Fatalf("Failed to query metrics after deletion: %v", err)
	}

	found := false
	for _, metric := range metrics {
		if metric.Labels != nil && strings.Contains(*metric.Labels, "recent") {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected recent metric to still exist after deleting old metrics")
	}
}

// Helper function for string pointers
func stringPtr(s string) *string {
	return &s
}