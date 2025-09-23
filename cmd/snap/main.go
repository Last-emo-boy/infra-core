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
	"github.com/last-emo-boy/infra-core/pkg/snap"
)

func main() {
	log.Printf("📦 Starting InfraCore Snap Service...")

	// Load environment
	environment := os.Getenv("ENVIRONMENT")
	if environment == "" {
		environment = "development"
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	log.Printf("📋 Environment: %s", environment)

	// Connect to database
	db, err := database.NewDB(cfg)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Ensure snap directories exist
	snapDir := cfg.Snap.RepoDir
	tempDir := cfg.Snap.TempDir
	if err := os.MkdirAll(snapDir, 0755); err != nil {
		log.Fatalf("❌ Failed to create snap directory %s: %v", snapDir, err)
	}
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		log.Fatalf("❌ Failed to create temp directory %s: %v", tempDir, err)
	}

	// Initialize snap manager
	snapManager, err := snap.NewSnapManager(db.DB, cfg.Snap)
	if err != nil {
		log.Fatalf("❌ Failed to initialize snap manager: %v", err)
	}

	log.Printf("📦 Starting snap manager engine...")
	snapManager.Start(context.Background())
	log.Printf("✅ Snap manager started")

	// Setup HTTP router
	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "infra-core-snap",
			"timestamp": time.Now().Unix(),
		})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		// Snap plans
		api.POST("/plans", snapManager.CreatePlan)
		api.GET("/plans", snapManager.ListPlans)
		api.GET("/plans/:id", snapManager.GetPlan)
		api.PUT("/plans/:id", snapManager.UpdatePlan)
		api.DELETE("/plans/:id", snapManager.DeletePlan)
		api.POST("/plans/:id/enable", snapManager.EnablePlan)
		api.POST("/plans/:id/disable", snapManager.DisablePlan)

		// Snapshots
		api.POST("/snapshots", snapManager.CreateSnapshot)
		api.GET("/snapshots", snapManager.ListSnapshots)
		api.GET("/snapshots/:id", snapManager.GetSnapshot)
		api.DELETE("/snapshots/:id", snapManager.DeleteSnapshot)
		api.GET("/snapshots/:id/status", snapManager.GetSnapshotStatus)
		api.POST("/snapshots/:id/verify", snapManager.VerifySnapshot)

		// Restore operations
		api.POST("/restore", snapManager.RestoreSnapshot)
		api.GET("/restore/:id/status", snapManager.GetRestoreStatus)
		api.POST("/restore/:id/cancel", snapManager.CancelRestore)

		// Management
		api.GET("/stats", snapManager.GetStats)
		api.POST("/cleanup", snapManager.CleanupOrphans)
		api.POST("/scrub", snapManager.TriggerScrub)
		api.GET("/scrub/status", snapManager.GetScrubStatus)
	}

	// Start HTTP server
	port := cfg.Snap.Port
	if port == 0 {
		port = 8086 // Default port for snap service
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("🚀 Snap Service API server starting on port %d", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-quit
	log.Printf("📦 Shutting down Snap Service...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop snap manager
	snapManager.Stop()

	// Shutdown HTTP server
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("❌ Server forced to shutdown: %v", err)
	}

	log.Printf("✅ Snap Service stopped gracefully")
}