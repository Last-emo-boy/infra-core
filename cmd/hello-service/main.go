package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	mux := http.NewServeMux()

	// 健康检查端点
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","service":"hello","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
	})

	// 主页端点
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
    <title>Hello Service</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .container { max-width: 600px; margin: 0 auto; }
        .header { color: #2c3e50; }
        .info { background: #f8f9fa; padding: 20px; border-radius: 5px; }
    </style>
</head>
<body>
    <div class="container">
        <h1 class="header">🚀 Infra-Core Hello Service</h1>
        <div class="info">
            <p><strong>Service:</strong> Hello Test Service</p>
            <p><strong>Time:</strong> %s</p>
            <p><strong>Remote Address:</strong> %s</p>
            <p><strong>User Agent:</strong> %s</p>
            <p><strong>Host:</strong> %s</p>
            <p><strong>Path:</strong> %s</p>
        </div>
        <p>This is a test service for the Infra-Core reverse proxy!</p>
    </div>
</body>
</html>`,
			time.Now().Format("2006-01-02 15:04:05"),
			r.RemoteAddr,
			r.UserAgent(),
			r.Host,
			r.URL.Path,
		)
	})

	// API 端点
	mux.HandleFunc("/api/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{
			"service": "hello",
			"version": "1.0.0",
			"timestamp": "%s",
			"request": {
				"method": "%s",
				"path": "%s",
				"remote_addr": "%s",
				"user_agent": "%s",
				"host": "%s"
			}
		}`,
			time.Now().Format(time.RFC3339),
			r.Method,
			r.URL.Path,
			r.RemoteAddr,
			r.UserAgent(),
			r.Host,
		)
	})

	server := &http.Server{
		Addr:         ":8081",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	fmt.Println("Hello Service starting on :8081")
	fmt.Println("Endpoints:")
	fmt.Println("  GET /         - Main page")
	fmt.Println("  GET /health   - Health check")
	fmt.Println("  GET /api/info - Service info")

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
