package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/last-emo-boy/infra-core/pkg/api/handlers"
	"github.com/last-emo-boy/infra-core/pkg/api/middleware"
	"github.com/last-emo-boy/infra-core/pkg/auth"
	"github.com/last-emo-boy/infra-core/pkg/config"
	"github.com/last-emo-boy/infra-core/pkg/database"
	"github.com/last-emo-boy/infra-core/pkg/services"
)

func main() {
	log.Println("🚀 Starting Console API Server...")

	// Load configuration
	environment := os.Getenv("INFRA_CORE_ENV")
	if environment == "" {
		environment = "development"
	}

	cfg, err := config.Load()
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
	ssoHandler := handlers.NewSSOHandler(authService, db)

	// Setup Gin router
	if environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Global middleware
	r.Use(middleware.LoggingMiddleware())
	r.Use(middleware.RecoveryMiddleware())
	r.Use(middleware.CORSMiddleware())

	// Static UI support
	uiDistDir := "/app/ui/dist"
	uiIndexFile := filepath.Join(uiDistDir, "index.html")
	uiAvailable := false

	if info, err := os.Stat(uiIndexFile); err == nil && !info.IsDir() {
		uiAvailable = true
		log.Printf("🖥️  UI assets detected at %s", uiDistDir)

		// Serve static assets (JS/CSS/etc.)
		assetsDir := filepath.Join(uiDistDir, "assets")
		if stat, err := os.Stat(assetsDir); err == nil && stat.IsDir() {
			r.Static("/assets", assetsDir)
		}

		// Serve index for root requests
		r.GET("/", func(c *gin.Context) {
			c.File(uiIndexFile)
		})
	} else {
		// Root health check (legacy JSON response)
		r.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"service":     "infra-core-console",
				"version":     "1.0.0",
				"status":      "healthy",
				"environment": environment,
				"time":        time.Now().UTC().Format(time.RFC3339),
			})
		})
	}

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

	// SPA fallback for non-API routes when UI is present
	if uiAvailable {
		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			if strings.HasPrefix(path, "/api/") {
				c.JSON(http.StatusNotFound, gin.H{
					"error":   "endpoint_not_found",
					"path":    path,
					"message": "The requested API route was not found",
				})
				return
			}

			c.File(uiIndexFile)
		})
	} else {
		r.NoRoute(func(c *gin.Context) {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "endpoint_not_found",
				"path":    c.Request.URL.Path,
				"message": "The requested route was not found",
			})
		})
	}

	// Protected routes
	protected := api.Group("/")
	protected.Use(middleware.AuthMiddleware(authService, db))
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

		// SSO management
		sso := protected.Group("/sso")
		{
			// Service registration and management
			sso.GET("/services", ssoHandler.ListServices)
			sso.GET("/user/services", ssoHandler.ListUserServices)
			sso.GET("/services/:id", ssoHandler.GetService)
			sso.GET("/services/:id/health", ssoHandler.GetServiceHealth)
			sso.GET("/services/:id/health/history", ssoHandler.GetServiceHealthHistory)

			// SSO authentication
			sso.POST("/login", ssoHandler.InitiateSSO)
			sso.GET("/validate", ssoHandler.ValidateSSO)

			// Admin-only SSO management
			adminSSO := sso.Group("/")
			adminSSO.Use(middleware.RequireRole(authService, "admin"))
			{
				adminSSO.POST("/services", ssoHandler.RegisterService)
				adminSSO.GET("/services/:id/permissions", ssoHandler.ListServicePermissions)
				adminSSO.PUT("/services/:id", ssoHandler.UpdateService)
				adminSSO.DELETE("/services/:id", ssoHandler.DeleteService)
				adminSSO.POST("/permissions/:user_id/:service_id/grant", ssoHandler.GrantServiceAccess)
				adminSSO.POST("/permissions/:user_id/:service_id/revoke", ssoHandler.RevokeServiceAccess)
			}
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

	// Start health checker service
	healthChecker := services.NewHealthChecker(db)
	healthChecker.Start()
	log.Printf("🏥 Health checker service started")

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
