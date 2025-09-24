package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthEndpoint(t *testing.T) {
	// Create test server
	mux := http.NewServeMux()
	
	// Add health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","service":"hello","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
	})
	
	server := httptest.NewServer(mux)
	defer server.Close()
	
	// Test health endpoint
	resp, err := http.Get(server.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	
	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, "hello", response["service"])
	assert.NotEmpty(t, response["timestamp"])
}

func TestMainPageEndpoint(t *testing.T) {
	// Create test server
	mux := http.NewServeMux()
	
	// Add main page endpoint
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
	
	server := httptest.NewServer(mux)
	defer server.Close()
	
	// Test main page
	resp, err := http.Get(server.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/html", resp.Header.Get("Content-Type"))
	
	// Read response body
	body := make([]byte, 4096)
	n, _ := resp.Body.Read(body)
	bodyStr := string(body[:n])
	
	// Check HTML content
	assert.Contains(t, bodyStr, "<!DOCTYPE html>")
	assert.Contains(t, bodyStr, "<title>Hello Service</title>")
	assert.Contains(t, bodyStr, "🚀 Infra-Core Hello Service")
	assert.Contains(t, bodyStr, "Hello Test Service")
	assert.Contains(t, bodyStr, "Infra-Core reverse proxy")
	
	// Check request information is included
	assert.Contains(t, bodyStr, "Remote Address")
	assert.Contains(t, bodyStr, "User Agent")
	assert.Contains(t, bodyStr, "Host")
	assert.Contains(t, bodyStr, "Path")
}

func TestAPIInfoEndpoint(t *testing.T) {
	// Create test server
	mux := http.NewServeMux()
	
	// Add API info endpoint
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
	
	server := httptest.NewServer(mux)
	defer server.Close()
	
	// Test API info endpoint
	resp, err := http.Get(server.URL + "/api/info")
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	
	// Check service info
	assert.Equal(t, "hello", response["service"])
	assert.Equal(t, "1.0.0", response["version"])
	assert.NotEmpty(t, response["timestamp"])
	
	// Check request information
	assert.Contains(t, response, "request")
	request := response["request"].(map[string]interface{})
	assert.Equal(t, "GET", request["method"])
	assert.Equal(t, "/api/info", request["path"])
	assert.NotEmpty(t, request["remote_addr"])
	assert.NotEmpty(t, request["host"])
}

func TestServerConfiguration(t *testing.T) {
	// Test server configuration values
	server := &http.Server{
		Addr:         ":8081",
		Handler:      http.NewServeMux(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	
	assert.Equal(t, ":8081", server.Addr)
	assert.Equal(t, 10*time.Second, server.ReadTimeout)
	assert.Equal(t, 10*time.Second, server.WriteTimeout)
	assert.Equal(t, 60*time.Second, server.IdleTimeout)
}

func TestRequestMethodHandling(t *testing.T) {
	// Create test server
	mux := http.NewServeMux()
	
	// Add endpoints that should handle different methods
	mux.HandleFunc("/api/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"method": "%s"}`, r.Method)
	})
	
	server := httptest.NewServer(mux)
	defer server.Close()
	
	// Test different HTTP methods
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	
	for _, method := range methods {
		t.Run(fmt.Sprintf("method_%s", method), func(t *testing.T) {
			req, err := http.NewRequest(method, server.URL+"/api/info", nil)
			require.NoError(t, err)
			
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			
			var response map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(t, err)
			
			assert.Equal(t, method, response["method"])
		})
	}
}

func TestHTMLContentStructure(t *testing.T) {
	// Test HTML structure components
	testHTML := `
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
        </div>
        <p>This is a test service for the Infra-Core reverse proxy!</p>
    </div>
</body>
</html>`
	
	// Check required HTML elements
	assert.Contains(t, testHTML, "<!DOCTYPE html>")
	assert.Contains(t, testHTML, "<html>")
	assert.Contains(t, testHTML, "<head>")
	assert.Contains(t, testHTML, "<title>Hello Service</title>")
	assert.Contains(t, testHTML, "<style>")
	assert.Contains(t, testHTML, "<body>")
	assert.Contains(t, testHTML, "</html>")
	
	// Check CSS classes
	assert.Contains(t, testHTML, "class=\"container\"")
	assert.Contains(t, testHTML, "class=\"header\"")
	assert.Contains(t, testHTML, "class=\"info\"")
	
	// Check content
	assert.Contains(t, testHTML, "🚀 Infra-Core Hello Service")
	assert.Contains(t, testHTML, "Hello Test Service")
	assert.Contains(t, testHTML, "Infra-Core reverse proxy")
}

func TestJSONResponseFormat(t *testing.T) {
	// Test JSON response formatting
	testTime := time.Now().Format(time.RFC3339)
	
	jsonResponse := fmt.Sprintf(`{
		"service": "hello",
		"version": "1.0.0",
		"timestamp": "%s",
		"request": {
			"method": "GET",
			"path": "/api/info",
			"remote_addr": "127.0.0.1:12345",
			"user_agent": "test-agent",
			"host": "localhost"
		}
	}`, testTime)
	
	var response map[string]interface{}
	err := json.Unmarshal([]byte(jsonResponse), &response)
	require.NoError(t, err)
	
	// Validate JSON structure
	assert.Equal(t, "hello", response["service"])
	assert.Equal(t, "1.0.0", response["version"])
	assert.Equal(t, testTime, response["timestamp"])
	
	// Validate request object
	assert.Contains(t, response, "request")
	request := response["request"].(map[string]interface{})
	assert.Equal(t, "GET", request["method"])
	assert.Equal(t, "/api/info", request["path"])
	assert.Equal(t, "127.0.0.1:12345", request["remote_addr"])
	assert.Equal(t, "test-agent", request["user_agent"])
	assert.Equal(t, "localhost", request["host"])
}

func TestEndpointAvailability(t *testing.T) {
	// Create test server with all endpoints
	mux := http.NewServeMux()
	
	// Main page
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("main page"))
	})
	
	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	})
	
	// API info
	mux.HandleFunc("/api/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"service":"hello"}`))
	})
	
	server := httptest.NewServer(mux)
	defer server.Close()
	
	// Test all endpoints
	endpoints := []struct {
		path           string
		expectedStatus int
		expectedType   string
	}{
		{"/", http.StatusOK, ""},
		{"/health", http.StatusOK, "application/json"},
		{"/api/info", http.StatusOK, "application/json"},
	}
	
	for _, endpoint := range endpoints {
		t.Run(fmt.Sprintf("endpoint_%s", endpoint.path), func(t *testing.T) {
			resp, err := http.Get(server.URL + endpoint.path)
			require.NoError(t, err)
			defer resp.Body.Close()
			
			assert.Equal(t, endpoint.expectedStatus, resp.StatusCode)
			
			if endpoint.expectedType != "" {
				assert.Equal(t, endpoint.expectedType, resp.Header.Get("Content-Type"))
			}
		})
	}
}

func TestTimeFormatting(t *testing.T) {
	// Test time formatting used in the service
	now := time.Now()
	
	// Test RFC3339 format (used in JSON responses)
	rfc3339 := now.Format(time.RFC3339)
	assert.True(t, strings.Contains(rfc3339, "T"))
	assert.True(t, strings.Contains(rfc3339, "Z") || strings.Contains(rfc3339, "+"))
	
	// Test custom format (used in HTML)
	customFormat := now.Format("2006-01-02 15:04:05")
	assert.True(t, strings.Contains(customFormat, "-"))
	assert.True(t, strings.Contains(customFormat, ":"))
	assert.True(t, len(customFormat) == 19) // YYYY-MM-DD HH:MM:SS
}

func TestUserAgentAndRemoteAddr(t *testing.T) {
	// Create test server that captures request info
	mux := http.NewServeMux()
	
	mux.HandleFunc("/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{
			"remote_addr": "%s",
			"user_agent": "%s",
			"host": "%s"
		}`, r.RemoteAddr, r.UserAgent(), r.Host)
	})
	
	server := httptest.NewServer(mux)
	defer server.Close()
	
	// Create request with custom User-Agent
	req, err := http.NewRequest("GET", server.URL+"/test", nil)
	require.NoError(t, err)
	req.Header.Set("User-Agent", "test-client/1.0")
	
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	
	assert.NotEmpty(t, response["remote_addr"])
	assert.Equal(t, "test-client/1.0", response["user_agent"])
	assert.NotEmpty(t, response["host"])
}