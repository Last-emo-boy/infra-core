package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {
	// Give the server time to start
	fmt.Println("Testing Console API endpoints...")

	// Test health check
	resp, err := http.Get("http://localhost:8082/api/v1/health")
	if err != nil {
		fmt.Printf("❌ Health check failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ Failed to read response: %v\n", err)
		return
	}

	fmt.Printf("✅ Health check response (%d): %s\n", resp.StatusCode, string(body))

	// Test root endpoint
	resp2, err := http.Get("http://localhost:8082/")
	if err != nil {
		fmt.Printf("❌ Root endpoint failed: %v\n", err)
		return
	}
	defer resp2.Body.Close()

	body2, err := io.ReadAll(resp2.Body)
	if err != nil {
		fmt.Printf("❌ Failed to read root response: %v\n", err)
		return
	}

	fmt.Printf("✅ Root endpoint response (%d): %s\n", resp2.StatusCode, string(body2))
}
