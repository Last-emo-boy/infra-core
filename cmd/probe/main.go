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
	"github.com/last-emo-boy/infra-core/pkg/probe"
)

func main() {
	log.Println("🔍 Starting InfraCore Probe Monitor...")

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

	// Create probe monitor
	probeMonitor := probe.New(db, cfg)

	// Start probe monitor
	if err := probeMonitor.Start(); err != nil {
		log.Fatalf("❌ Failed to start probe monitor: %v", err)
	}

	// Set up Gin router
	if environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.Default()

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		status := probeMonitor.GetStatus()
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"probe":     status,
			"timestamp": time.Now().Unix(),
		})
	})

	// API routes
	api := r.Group("/api/v1")
	{
		// Probe configuration
		probes := api.Group("/probes")
		{
			probes.POST("/", probeMonitor.CreateProbe)
			probes.GET("/", probeMonitor.ListProbes)
			probes.GET("/:id", probeMonitor.GetProbe)
			probes.PUT("/:id", probeMonitor.UpdateProbe)
			probes.DELETE("/:id", probeMonitor.DeleteProbe)
			probes.POST("/:id/enable", probeMonitor.EnableProbe)
			probes.POST("/:id/disable", probeMonitor.DisableProbe)
		}

		// Probe results and metrics
		results := api.Group("/results")
		{
			results.GET("/", probeMonitor.ListResults)
			results.GET("/:probe_id", probeMonitor.GetProbeResults)
			results.GET("/:probe_id/latest", probeMonitor.GetLatestResult)
			results.GET("/:probe_id/metrics", probeMonitor.GetProbeMetrics)
			results.GET("/:probe_id/history", probeMonitor.GetProbeHistory)
		}

		// Health monitoring
		health := api.Group("/health")
		{
			health.GET("/services", probeMonitor.GetServiceHealth)
			health.GET("/services/:service_id", probeMonitor.GetServiceHealthDetail)
			health.GET("/overview", probeMonitor.GetHealthOverview)
			health.GET("/alerts", probeMonitor.GetActiveAlerts)
		}

		// Monitoring control
		control := api.Group("/control")
		{
			control.POST("/scan", probeMonitor.TriggerFullScan)
			control.POST("/cleanup", probeMonitor.CleanupOldResults)
			control.GET("/metrics", probeMonitor.GetMonitorMetrics)
			control.GET("/status", probeMonitor.GetDetailedStatus)
		}
	}

	// Create HTTP server
	port := cfg.Probe.Port
	if port == 0 {
		port = 8085 // Default probe port
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
		log.Printf("🚀 Probe Monitor API server starting on port %d", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("❌ Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down probe monitor...")

	// Shutdown server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("❌ Server forced to shutdown: %v", err)
	}

	// Stop probe monitor
	probeMonitor.Stop()

	log.Println("✅ Probe monitor shutdown complete")
}