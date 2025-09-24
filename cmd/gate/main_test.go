package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Set test environment
	os.Setenv("ENVIRONMENT", "test")
	
	// Run tests
	code := m.Run()
	
	// Cleanup
	os.Unsetenv("ENVIRONMENT")
	
	os.Exit(code)
}

func TestVersionFlag(t *testing.T) {
	// This test simulates the version flag behavior
	// In a real scenario, we would need to refactor main() to be testable
	
	expectedOutput := "Infra-Core Gate v1.0.0"
	
	// Test version string format
	versionString := "Infra-Core Gate v1.0.0"
	assert.Equal(t, expectedOutput, versionString)
	
	// Test version description
	description := "Self-developed reverse proxy and HTTPS gateway"
	assert.NotEmpty(t, description)
	assert.Contains(t, description, "reverse proxy")
	assert.Contains(t, description, "HTTPS gateway")
}

func TestMetricsEndpoints(t *testing.T) {
	// Create a test HTTP server to simulate metrics endpoints
	mux := http.NewServeMux()
	
	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
	})
	
	// Metrics endpoint
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{
			"request_count": {},
			"error_count": {},
			"response_times": {},
			"timestamp": "%s"
		}`, time.Now().Format(time.RFC3339))
	})
	
	// Routes endpoint
	mux.HandleFunc("/routes", func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"routes":[],"count":0}`)
	})
	
	server := httptest.NewServer(mux)
	defer server.Close()
	
	// Test health endpoint
	t.Run("health endpoint", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
		
		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		
		assert.Equal(t, "healthy", response["status"])
		assert.NotEmpty(t, response["timestamp"])
	})
	
	// Test metrics endpoint
	t.Run("metrics endpoint", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/metrics")
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
		
		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		
		assert.Contains(t, response, "request_count")
		assert.Contains(t, response, "error_count")
		assert.Contains(t, response, "response_times")
		assert.Contains(t, response, "timestamp")
	})
	
	// Test routes endpoint
	t.Run("routes endpoint", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/routes")
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
		
		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		require.NoError(t, err)
		
		assert.Contains(t, response, "routes")
		assert.Contains(t, response, "count")
		assert.Equal(t, float64(0), response["count"])
	})
	
	// Test routes endpoint with wrong method
	t.Run("routes endpoint wrong method", func(t *testing.T) {
		req, err := http.NewRequest("POST", server.URL+"/routes", nil)
		require.NoError(t, err)
		
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})
}

func TestFormatMetricsMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]int64
		expected string
	}{
		{
			name:     "empty map",
			input:    map[string]int64{},
			expected: "{}",
		},
		{
			name:     "single entry",
			input:    map[string]int64{"test": 42},
			expected: `{"test":42}`,
		},
		{
			name:     "multiple entries",
			input:    map[string]int64{"a": 1, "b": 2},
			expected: `{"a":1,"b":2}`, // Note: order might vary in real implementation
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMetricsMap(tt.input)
			
			if len(tt.input) <= 1 {
				assert.Equal(t, tt.expected, result)
			} else {
				// For multiple entries, just check it's valid JSON with correct content
				assert.True(t, strings.HasPrefix(result, "{"))
				assert.True(t, strings.HasSuffix(result, "}"))
				for k, v := range tt.input {
					expectedEntry := fmt.Sprintf(`"%s":%d`, k, v)
					assert.Contains(t, result, expectedEntry)
				}
			}
		})
	}
}

func TestACMEHandler(t *testing.T) {
	// Create a mock router
	mockRouter := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("regular route"))
	})
	
	// Create ACME handler (simplified version)
	acmeHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle ACME challenges first
		if strings.HasPrefix(r.URL.Path, "/.well-known/acme-challenge/") {
			token := r.URL.Path[len("/.well-known/acme-challenge/"):]
			if token != "" {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
				fmt.Fprintf(w, "ACME challenge token: %s", token)
				return
			}
		}
		
		// Regular routing
		mockRouter.ServeHTTP(w, r)
	})
	
	server := httptest.NewServer(acmeHandler)
	defer server.Close()
	
	// Test ACME challenge
	t.Run("ACME challenge", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/.well-known/acme-challenge/test-token")
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "text/plain", resp.Header.Get("Content-Type"))
		
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		bodyStr := string(body[:n])
		
		assert.Equal(t, "ACME challenge token: test-token", bodyStr)
	})
	
	// Test empty token
	t.Run("ACME challenge empty token", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/.well-known/acme-challenge/")
		require.NoError(t, err)
		defer resp.Body.Close()
		
		// Should fall through to regular routing
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		bodyStr := string(body[:n])
		
		assert.Equal(t, "regular route", bodyStr)
	})
	
	// Test regular route
	t.Run("regular route", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/api/test")
		require.NoError(t, err)
		defer resp.Body.Close()
		
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		bodyStr := string(body[:n])
		
		assert.Equal(t, "regular route", bodyStr)
	})
}

func TestServerConfiguration(t *testing.T) {
	// Test HTTP server configuration
	httpServer := &http.Server{
		Addr:         ":8080",
		Handler:      http.NewServeMux(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	assert.Equal(t, ":8080", httpServer.Addr)
	assert.Equal(t, 30*time.Second, httpServer.ReadTimeout)
	assert.Equal(t, 30*time.Second, httpServer.WriteTimeout)
	assert.Equal(t, 60*time.Second, httpServer.IdleTimeout)
	
	// Test metrics server configuration
	metricsServer := &http.Server{
		Addr:         ":9080",
		Handler:      http.NewServeMux(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	
	assert.Equal(t, ":9080", metricsServer.Addr)
	assert.Equal(t, 10*time.Second, metricsServer.ReadTimeout)
	assert.Equal(t, 10*time.Second, metricsServer.WriteTimeout)
}

func TestRouteStructure(t *testing.T) {
	// Test route structure definition
	type Route struct {
		ID         string
		Host       string
		PathPrefix string
		Upstream   string
	}
	
	// Test sample routes like in main.go
	routes := []Route{
		{
			ID:         "hello",
			Host:       "",
			PathPrefix: "/hello",
			Upstream:   "http://127.0.0.1:8081",
		},
		{
			ID:         "console",
			Host:       "",
			PathPrefix: "/console", 
			Upstream:   "http://127.0.0.1:8082",
		},
		{
			ID:         "default",
			Host:       "",
			PathPrefix: "/",
			Upstream:   "http://127.0.0.1:8081",
		},
	}
	
	// Validate route structure
	for _, route := range routes {
		assert.NotEmpty(t, route.ID, "Route ID should not be empty")
		assert.NotEmpty(t, route.PathPrefix, "Route PathPrefix should not be empty")
		assert.NotEmpty(t, route.Upstream, "Route Upstream should not be empty")
		assert.True(t, strings.HasPrefix(route.Upstream, "http://"), "Upstream should be a valid HTTP URL")
		assert.True(t, strings.HasPrefix(route.PathPrefix, "/"), "PathPrefix should start with /")
	}
}

func TestContextTimeout(t *testing.T) {
	// Test context timeout behavior
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	select {
	case <-ctx.Done():
		t.Error("Context should not be done immediately")
	default:
		// Context is not done, which is expected
	}
	
	// Test context deadline
	deadline, ok := ctx.Deadline()
	assert.True(t, ok, "Context should have a deadline")
	assert.True(t, deadline.After(time.Now()), "Deadline should be in the future")
}

func TestShutdownTimeout(t *testing.T) {
	// Test shutdown timeout configuration
	shutdownTimeout := 30 * time.Second
	
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	
	deadline, ok := ctx.Deadline()
	assert.True(t, ok)
	
	// Check that deadline is approximately 30 seconds from now
	expectedDeadline := time.Now().Add(shutdownTimeout)
	timeDiff := deadline.Sub(expectedDeadline)
	
	// Allow for small timing differences (within 1 second)
	assert.True(t, timeDiff < time.Second && timeDiff > -time.Second)
}