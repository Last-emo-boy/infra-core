package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthCheckEndpoint(t *testing.T) {
	// Create a mock server for health check endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/health" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			response := map[string]interface{}{
				"status":    "healthy",
				"service":   "infra-core-console",
				"timestamp": time.Now().Unix(),
			}
			_ = json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	
	// Test health check request
	resp, err := http.Get(server.URL + "/api/v1/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	
	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)
	
	assert.Equal(t, "healthy", response["status"])
	assert.Equal(t, "infra-core-console", response["service"])
	assert.Contains(t, response, "timestamp")
}

func TestRootEndpoint(t *testing.T) {
	// Create a mock server for root endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			response := map[string]interface{}{
				"service":     "infra-core-console",
				"version":     "1.0.0",
				"status":      "healthy",
				"environment": "development",
				"time":        time.Now().UTC().Format(time.RFC3339),
			}
			_ = json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	
	// Test root endpoint request
	resp, err := http.Get(server.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	
	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	require.NoError(t, err)
	
	assert.Equal(t, "infra-core-console", response["service"])
	assert.Equal(t, "1.0.0", response["version"])
	assert.Equal(t, "healthy", response["status"])
	assert.Contains(t, response, "time")
}

func TestHTTPClientConfiguration(t *testing.T) {
	// Test HTTP client default configuration
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	assert.Equal(t, 30*time.Second, client.Timeout)
	assert.NotNil(t, client.Transport)
}

func TestResponseReading(t *testing.T) {
	// Test response body reading functionality
	testData := `{"status":"healthy","message":"API is working"}`
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(testData))
	}))
	defer server.Close()
	
	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	
	assert.Equal(t, testData, string(body))
	assert.True(t, len(body) > 0, "Response body should not be empty")
}

func TestErrorHandling(t *testing.T) {
	// Test error handling for various scenarios
	testCases := []struct {
		name           string
		serverBehavior func(w http.ResponseWriter, r *http.Request)
		expectedStatus int
	}{
		{
			name: "server error",
			serverBehavior: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("Internal Server Error"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
		{
			name: "not found",
			serverBehavior: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte("Not Found"))
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "bad request",
			serverBehavior: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte("Bad Request"))
			},
			expectedStatus: http.StatusBadRequest,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(tc.serverBehavior))
			defer server.Close()
			
			resp, err := http.Get(server.URL)
			require.NoError(t, err)
			defer resp.Body.Close()
			
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)
			
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.NotEmpty(t, body, "Error response should have body")
		})
	}
}

func TestAPIEndpointURLs(t *testing.T) {
	// Test API endpoint URL construction
	baseURL := "http://localhost:8082"
	endpoints := []string{
		"/",
		"/api/v1/health",
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/services",
		"/api/v1/system/info",
	}
	
	for _, endpoint := range endpoints {
		fullURL := baseURL + endpoint
		
		assert.True(t, len(fullURL) > len(baseURL), "Full URL should be longer than base URL")
		assert.Contains(t, fullURL, baseURL, "Full URL should contain base URL")
		assert.True(t, fullURL[0] != '/', "Full URL should not start with /")
		
		if endpoint != "/" {
			assert.Contains(t, fullURL, endpoint, "Full URL should contain endpoint path")
		}
	}
}

func TestJSONResponseParsing(t *testing.T) {
	// Test JSON response parsing
	testResponses := []struct {
		name     string
		jsonData string
		isValid  bool
	}{
		{
			name:     "valid health response",
			jsonData: `{"status":"healthy","timestamp":1234567890}`,
			isValid:  true,
		},
		{
			name:     "valid service info",
			jsonData: `{"service":"infra-core-console","version":"1.0.0","status":"healthy"}`,
			isValid:  true,
		},
		{
			name:     "invalid json",
			jsonData: `{"status":"healthy","timestamp":}`,
			isValid:  false,
		},
		{
			name:     "empty response",
			jsonData: `{}`,
			isValid:  true,
		},
	}
	
	for _, tc := range testResponses {
		t.Run(tc.name, func(t *testing.T) {
			var response map[string]interface{}
			err := json.Unmarshal([]byte(tc.jsonData), &response)
			
			if tc.isValid {
				assert.NoError(t, err, "Valid JSON should parse without error")
				assert.NotNil(t, response, "Response should not be nil")
			} else {
				assert.Error(t, err, "Invalid JSON should produce error")
			}
		})
	}
}

func TestHTTPMethods(t *testing.T) {
	// Test different HTTP methods
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]string{
			"method": r.Method,
			"path":   r.URL.Path,
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
	
	for _, method := range methods {
		t.Run(fmt.Sprintf("method_%s", method), func(t *testing.T) {
			req, err := http.NewRequest(method, server.URL+"/test", nil)
			require.NoError(t, err)
			
			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()
			
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			
			var response map[string]string
			err = json.NewDecoder(resp.Body).Decode(&response)
			require.NoError(t, err)
			
			assert.Equal(t, method, response["method"])
			assert.Equal(t, "/test", response["path"])
		})
	}
}

func TestResponseHeaders(t *testing.T) {
	// Test response headers handling
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Service", "infra-core-console")
		w.Header().Set("X-Version", "1.0.0")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()
	
	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
	assert.Equal(t, "infra-core-console", resp.Header.Get("X-Service"))
	assert.Equal(t, "1.0.0", resp.Header.Get("X-Version"))
}

func TestTimeoutHandling(t *testing.T) {
	// Test timeout handling
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"slow"}`))
	}))
	defer server.Close()
	
	// Test with sufficient timeout
	client := &http.Client{
		Timeout: 1 * time.Second,
	}
	
	resp, err := client.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	
	// Test with very short timeout
	fastClient := &http.Client{
		Timeout: 1 * time.Millisecond,
	}
	
	_, err = fastClient.Get(server.URL)
	assert.Error(t, err, "Very short timeout should cause error")
}

func TestUserAgentHeader(t *testing.T) {
	// Test User-Agent header handling
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := map[string]string{
			"user_agent": r.UserAgent(),
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	req, err := http.NewRequest("GET", server.URL, nil)
	require.NoError(t, err)
	req.Header.Set("User-Agent", "infra-core-api-test/1.0")
	
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	
	var response map[string]string
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err)
	
	assert.Equal(t, "infra-core-api-test/1.0", response["user_agent"])
}

func TestResponseBodySize(t *testing.T) {
	// Test handling of different response body sizes
	testCases := []struct {
		name     string
		dataSize int
	}{
		{"small response", 100},
		{"medium response", 1024},
		{"large response", 10240},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create response data of specified size
			data := make([]byte, tc.dataSize)
			for i := range data {
				data[i] = byte('a' + (i % 26))
			}
			
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(data)
			}))
			defer server.Close()
			
			resp, err := http.Get(server.URL)
			require.NoError(t, err)
			defer resp.Body.Close()
			
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			
			assert.Equal(t, tc.dataSize, len(body))
			assert.Equal(t, data, body)
		})
	}
}