package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/last-emo-boy/infra-core/pkg/config"
	"github.com/last-emo-boy/infra-core/pkg/database"
	"github.com/last-emo-boy/infra-core/pkg/orchestrator"
)

func main() {
	log.Println("🎭 Starting InfraCore Orchestrator...")

	// Load environment
	environment := os.Getenv("ENVIRONMENT")
	if environment == "" {
		environment = "development"
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}

	log.Printf("📋 Environment: %s", environment)

	// Initialize database
	db, err := database.NewDB(cfg)
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create orchestrator
	orch := orchestrator.New(db, cfg)

	// Start orchestrator
	if err := orch.Start(); err != nil {
		log.Fatalf("❌ Failed to start orchestrator: %v", err)
	}

	// Set up Gin router
	if environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		status := orch.GetStatus()
		c.JSON(http.StatusOK, gin.H{
			"status":      "healthy",
			"orchestrator": status,
			"timestamp":   time.Now().Unix(),
		})
	})

	// API routes
	api := r.Group("/api/v1")
	{
		// Service orchestration
		services := api.Group("/services")
		{
			services.POST("/deploy", orch.DeployService)
			services.POST("/:id/start", orch.StartService)
			services.POST("/:id/stop", orch.StopService)
			services.POST("/:id/restart", orch.RestartService)
			services.DELETE("/:id", orch.RemoveService)
			services.GET("/:id/status", orch.GetServiceStatus)
			services.GET("/:id/logs", orch.GetServiceLogs)
		}

		// Deployment management
		deployments := api.Group("/deployments")
		{
			deployments.GET("/", orch.ListDeployments)
			deployments.GET("/:id", orch.GetDeployment)
			deployments.POST("/:id/rollback", orch.RollbackDeployment)
			deployments.DELETE("/:id", orch.DeleteDeployment)
		}

		// Cluster management
		cluster := api.Group("/cluster")
		{
			cluster.GET("/nodes", orch.ListNodes)
			cluster.GET("/resources", orch.GetClusterResources)
			cluster.GET("/events", orch.GetClusterEvents)
		}

		// Orchestrator control
		control := api.Group("/control")
		{
			control.POST("/sync", orch.SyncServices)
			control.POST("/cleanup", orch.CleanupResources)
			control.GET("/metrics", orch.GetMetrics)
		}
	}

	// Create HTTP server
	port := cfg.Orchestrator.Port
	if port == 0 {
		port = 8084 // Default orchestrator port
	}

	server := &http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		Handler:        r,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB
	}

	// Start server in goroutine
	go func() {
		log.Printf("🚀 Orchestrator API server starting on port %d", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down orchestrator...")

	// Shutdown server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("❌ Server forced to shutdown: %v", err)
	}

	// Stop orchestrator
	orch.Stop()

	log.Println("✅ Orchestrator shutdown complete")
}