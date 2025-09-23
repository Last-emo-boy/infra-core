package main

import (
	"fmt"
	"log"
	"time"

	"github.com/last-emo-boy/infra-core/pkg/config"
	"github.com/last-emo-boy/infra-core/pkg/database"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	fmt.Println("Testing Infra-Core Database System")
	fmt.Printf("Database Path: %s\n", cfg.Console.Database.Path)

	// Initialize database
	db, err := database.NewDB(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	fmt.Println("✅ Database initialized successfully")

	// Test health check
	if err := db.HealthCheck(); err != nil {
		log.Fatalf("Database health check failed: %v", err)
	}
	fmt.Println("✅ Database health check passed")

	// Get database statistics
	stats, err := db.GetStats()
	if err != nil {
		log.Fatalf("Failed to get database stats: %v", err)
	}

	fmt.Println("\n📊 Database Statistics:")
	for key, value := range stats {
		fmt.Printf("  %s: %v\n", key, value)
	}

	// Test user repository
	userRepo := database.NewUserRepository(db)

	// Create a test user
	user := &database.User{
		Username:     "admin",
		Email:        "admin@infra-core.local",
		PasswordHash: "$2a$10$example_hash", // In real app, use bcrypt
		Role:         "admin",
	}

	fmt.Println("\n👤 Testing User Repository:")
	if err := userRepo.Create(user); err != nil {
		log.Printf("Failed to create user (might already exist): %v", err)
	} else {
		fmt.Printf("✅ Created user: %s (ID: %d)\n", user.Username, user.ID)
	}

	// Test service repository
	serviceRepo := database.NewServiceRepository(db)

	service := &database.Service{
		Name:       "hello-service",
		YAMLConfig: "name: hello-service\nimage: nginx:alpine\nports:\n  - internal: 8080",
		Version:    1,
		Status:     "stopped",
	}

	fmt.Println("\n🚀 Testing Service Repository:")
	if err := serviceRepo.Create(service); err != nil {
		log.Printf("Failed to create service (might already exist): %v", err)
	} else {
		fmt.Printf("✅ Created service: %s (ID: %s)\n", service.Name, service.ID)
	}

	// List services
	services, err := serviceRepo.List()
	if err != nil {
		log.Printf("Failed to list services: %v", err)
	} else {
		fmt.Printf("📋 Found %d services in database\n", len(services))
		for _, svc := range services {
			fmt.Printf("  - %s (%s) - Status: %s\n", svc.Name, svc.ID, svc.Status)
		}
	}

	// Test route repository
	routeRepo := database.NewRouteRepository(db)

	route := &database.Route{
		Host:              "hello.local",
		PathPrefix:        "/",
		UpstreamServiceID: &service.ID,
	}

	fmt.Println("\n🌐 Testing Route Repository:")
	if err := routeRepo.Create(route); err != nil {
		log.Printf("Failed to create route (might already exist): %v", err)
	} else {
		fmt.Printf("✅ Created route: %s%s -> %s\n", route.Host, route.PathPrefix, *route.UpstreamServiceID)
	}

	// Test metric repository
	metricRepo := database.NewMetricRepository(db)

	metric := &database.Metric{
		Timestamp:   time.Now(),
		ScopeType:   "service",
		ScopeID:     service.ID,
		MetricName:  "cpu_usage",
		MetricValue: 45.5,
	}

	fmt.Println("\n📈 Testing Metric Repository:")
	if err := metricRepo.Insert(metric); err != nil {
		log.Printf("Failed to insert metric: %v", err)
	} else {
		fmt.Printf("✅ Inserted metric: %s = %.2f for %s/%s\n",
			metric.MetricName, metric.MetricValue, metric.ScopeType, metric.ScopeID)
	}

	// Get updated stats
	fmt.Println("\n📊 Updated Database Statistics:")
	stats, err = db.GetStats()
	if err != nil {
		log.Fatalf("Failed to get updated database stats: %v", err)
	}

	for key, value := range stats {
		fmt.Printf("  %s: %v\n", key, value)
	}

	fmt.Println("\n🎉 Database system test completed successfully!")
}
