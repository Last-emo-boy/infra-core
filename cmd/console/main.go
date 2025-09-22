package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/last-emo-boy/infra-core/pkg/api/handlers"
	"github.com/last-emo-boy/infra-core/pkg/api/middleware"
	"github.com/last-emo-boy/infra-core/pkg/auth"
	"github.com/last-emo-boy/infra-core/pkg/config"
	"github.com/last-emo-boy/infra-core/pkg/database"
)

func main() {
	log.Println("🚀 Starting Console API Server...")

	// Load configuration
	environment := os.Getenv("INFRA_CORE_ENV")
	if environment == "" {
		environment = "development"
	}

	cfg, err := config.Load(environment)
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}

	log.Printf("📋 Environment: %s", environment)
	log.Printf("🌐 Server will start on %s:%d", cfg.Console.Host, cfg.Console.Port)

	// Initialize database
	db, err := database.NewDB(cfg)
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Initialize auth service
	authService, err := auth.NewAuth(&cfg.Console)
	if err != nil {
		log.Fatalf("❌ Failed to initialize auth service: %v", err)
	}

	// Create handlers
	userHandler := handlers.NewUserHandler(authService, db)
	serviceHandler := handlers.NewServiceHandler(db)
	systemHandler := handlers.NewSystemHandler(db)

	// Setup Gin router
	if environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Global middleware
	r.Use(middleware.LoggingMiddleware())
	r.Use(middleware.RecoveryMiddleware())
	r.Use(middleware.CORSMiddleware())

	// Root health check
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":     "infra-core-console",
			"version":     "1.0.0",
			"status":      "healthy",
			"environment": environment,
			"time":        time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Public routes
	api := r.Group("/api/v1")
	{
		// Authentication endpoints
		auth := api.Group("/auth")
		{
			auth.POST("/register", userHandler.Register)
			auth.POST("/login", userHandler.Login)
		}

		// Health check endpoint
		api.GET("/health", systemHandler.HealthCheck)
	}

	// Protected routes
	protected := api.Group("/")
	protected.Use(middleware.AuthMiddleware(authService))
	{
		// User management
		users := protected.Group("/users")
		{
			users.GET("/profile", userHandler.GetProfile)
			users.PUT("/:id", userHandler.UpdateUser)
		}

		// Admin-only user management
		adminUsers := users.Group("/")
		adminUsers.Use(middleware.RequireRole(authService, "admin"))
		{
			adminUsers.GET("/", userHandler.ListUsers)
			adminUsers.DELETE("/:id", userHandler.DeleteUser)
		}

		// Service management
		services := protected.Group("/services")
		{
			services.POST("/", serviceHandler.CreateService)
			services.GET("/", serviceHandler.ListServices)
			services.GET("/:id", serviceHandler.GetService)
			services.PUT("/:id", serviceHandler.UpdateService)
			services.DELETE("/:id", serviceHandler.DeleteService)
			services.POST("/:id/start", serviceHandler.StartService)
			services.POST("/:id/stop", serviceHandler.StopService)
			services.GET("/:id/logs", serviceHandler.GetServiceLogs)
		}

		// System information
		system := protected.Group("/system")
		{
			system.GET("/info", systemHandler.GetSystemInfo)
			system.GET("/metrics", systemHandler.GetMetrics)
			system.GET("/dashboard", systemHandler.GetDashboardData)
		}

		// Admin-only system management
		adminSystem := system.Group("/")
		adminSystem.Use(middleware.RequireRole(authService, "admin"))
		{
			adminSystem.GET("/audit", systemHandler.GetAuditLogs)
		}
	}

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Console.Host, cfg.Console.Port)
	server := &http.Server{
		Addr:           addr,
		Handler:        r,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	log.Printf("🚀 Console API server starting on %s", addr)
	log.Printf("📝 Environment: %s", environment)
	if len(cfg.Console.Auth.JWT.Secret) > 8 {
		log.Printf("🔑 JWT Secret: %s...", cfg.Console.Auth.JWT.Secret[:8])
	} else {
		log.Printf("🔑 JWT Secret: Generated automatically")
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("❌ Failed to start server: %v", err)
	}
}
