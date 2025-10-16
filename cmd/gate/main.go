package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/last-emo-boy/infra-core/pkg/acme"
	"github.com/last-emo-boy/infra-core/pkg/config"
	"github.com/last-emo-boy/infra-core/pkg/router"
)

func main() {
	var (
		version    = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	if *version {
		fmt.Println("Infra-Core Gate v1.0.0")
		fmt.Println("Self-developed reverse proxy and HTTPS gateway")
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	fmt.Printf("Starting Infra-Core Gate v1.0.0\n")
	fmt.Printf("HTTP Port: %d\n", cfg.Gate.Ports.HTTP)
	fmt.Printf("HTTPS Port: %d\n", cfg.Gate.Ports.HTTPS)
	fmt.Printf("Data Directory: %s\n", cfg.Gate.ACME.CacheDir)

	// Create router
	r := router.NewRouter(cfg)

	// Create ACME client for HTTPS
	var acmeClient *acme.Client
	if cfg.Gate.ACME.Email != "" {
		acmeClient, err = acme.NewClient(cfg)
		if err != nil {
			log.Printf("Warning: Failed to create ACME client: %v", err)
		} else {
			fmt.Println("ACME client initialized successfully")
		}
	}

	err = r.AddRoute(&router.Route{
		ID:         "console",
		Host:       "",
		PathPrefix: "/console",
		Upstream:   fmt.Sprintf("http://127.0.0.1:%d", cfg.Console.Port),
	})
	if err != nil {
		log.Printf("Warning: Failed to add console route: %v", err)
	}

	// Add default route to hello service
	err = r.AddRoute(&router.Route{
		ID:         "default",
		Host:       "",
		PathPrefix: "/",
		Upstream:   fmt.Sprintf("http://127.0.0.1:%d", cfg.Console.Port),
	})
	if err != nil {
		log.Printf("Warning: Failed to add default route: %v", err)
	}

	// Create HTTP handler with ACME support
	var httpHandler http.Handler = r
	if acmeClient != nil {
		// Wrap router with ACME challenge handler
		httpHandler = createACMEHandler(r, acmeClient)
	}

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Gate.Ports.HTTP),
		Handler:      httpHandler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Create metrics server
	metricsHandler := createMetricsHandler(r)
	metricsServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Gate.Ports.HTTP+1000),
		Handler:      metricsHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Start servers in goroutines
	go func() {
		fmt.Printf("HTTP server listening on :%d\n", cfg.Gate.Ports.HTTP)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failed: %v", err)
		}
	}()

	go func() {
		fmt.Printf("Metrics server listening on :%d\n", cfg.Gate.Ports.HTTP+1000)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Metrics server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\nShutting down Gate...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	if err := metricsServer.Shutdown(ctx); err != nil {
		log.Printf("Metrics server shutdown error: %v", err)
	}

	fmt.Println("Gate stopped")
}

// createMetricsHandler creates an HTTP handler for metrics endpoint
func createMetricsHandler(r *router.Router) http.Handler {
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), 5*time.Second)
		defer cancel()

		if err := r.HealthCheck(ctx); err != nil {
			http.Error(w, fmt.Sprintf("Health check failed: %v", err), http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
	})

	// Metrics endpoint
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, req *http.Request) {
		metrics := r.GetMetrics()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		// Simple JSON metrics format (can be enhanced later)
		fmt.Fprintf(w, `{
			"request_count": %v,
			"error_count": %v,
			"response_times": %v,
			"timestamp": "%s"
		}`,
			formatMetricsMap(metrics.RequestCount),
			formatMetricsMap(metrics.ErrorCount),
			formatMetricsMap(metrics.ResponseTimes),
			time.Now().Format(time.RFC3339),
		)
	})

	// Routes management endpoint
	mux.HandleFunc("/routes", func(w http.ResponseWriter, req *http.Request) {
		switch req.Method {
		case http.MethodGet:
			routes := r.ListRoutes()
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			fmt.Fprintf(w, `{"routes":[`)
			for i, route := range routes {
				if i > 0 {
					fmt.Fprintf(w, ",")
				}
				fmt.Fprintf(w, `{
					"id": "%s",
					"host": "%s", 
					"path_prefix": "%s",
					"upstream": "%s",
					"created_at": "%s",
					"updated_at": "%s"
				}`,
					route.ID, route.Host, route.PathPrefix, route.Upstream,
					route.CreatedAt.Format(time.RFC3339),
					route.UpdatedAt.Format(time.RFC3339),
				)
			}
			fmt.Fprintf(w, `],"count":%d}`, len(routes))

		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	return mux
}

// formatMetricsMap formats a metrics map for JSON output
func formatMetricsMap(m map[string]int64) string {
	if len(m) == 0 {
		return "{}"
	}

	result := "{"
	first := true
	for k, v := range m {
		if !first {
			result += ","
		}
		result += fmt.Sprintf(`"%s":%d`, k, v)
		first = false
	}
	result += "}"
	return result
}

// createACMEHandler creates an HTTP handler that supports ACME challenges
func createACMEHandler(router http.Handler, acmeClient *acme.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle ACME challenges first, before routing
		if strings.HasPrefix(r.URL.Path, "/.well-known/acme-challenge/") {
			// Extract token from path
			token := r.URL.Path[len("/.well-known/acme-challenge/"):]
			if token != "" {
				// Serve ACME challenge response
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, "ACME challenge token: %s", token)
				return
			}
		}

		// Regular routing
		router.ServeHTTP(w, r)
	})
}
