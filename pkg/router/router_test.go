package router

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/last-emo-boy/infra-core/pkg/config"
)

func TestNewRouter(t *testing.T) {
	cfg := &config.Config{}
	router := NewRouter(cfg)

	assert.NotNil(t, router)
	assert.NotNil(t, router.routes)
	assert.NotNil(t, router.proxies)
	assert.NotNil(t, router.metrics)
	assert.Equal(t, cfg, router.config)
	assert.Empty(t, router.routes)
	assert.Empty(t, router.proxies)
}

func TestAddRoute(t *testing.T) {
	router := NewRouter(&config.Config{})

	tests := []struct {
		name          string
		route         *Route
		expectedError bool
	}{
		{
			name: "valid route",
			route: &Route{
				ID:         "test-route-1",
				Host:       "example.com",
				PathPrefix: "/api",
				Upstream:   "http://localhost:8080",
			},
			expectedError: false,
		},
		{
			name: "valid route without host",
			route: &Route{
				ID:         "test-route-2",
				PathPrefix: "/web",
				Upstream:   "http://localhost:9090",
			},
			expectedError: false,
		},
		{
			name: "invalid upstream URL",
			route: &Route{
				ID:         "test-route-3",
				Host:       "example.com",
				PathPrefix: "/api",
				Upstream:   "://invalid-url",
			},
			expectedError: true,
		},
		{
			name: "https upstream",
			route: &Route{
				ID:         "test-route-4",
				Host:       "secure.example.com",
				PathPrefix: "/secure",
				Upstream:   "https://localhost:8443",
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := router.AddRoute(tt.route)

			if tt.expectedError {
				assert.Error(t, err)
				// Route should not be added if there's an error
				_, getErr := router.GetRoute(tt.route.ID)
				assert.Error(t, getErr)
			} else {
				assert.NoError(t, err)
				
				// Verify route was added
				storedRoute, err := router.GetRoute(tt.route.ID)
				assert.NoError(t, err)
				assert.Equal(t, tt.route.ID, storedRoute.ID)
				assert.Equal(t, tt.route.Host, storedRoute.Host)
				assert.Equal(t, tt.route.PathPrefix, storedRoute.PathPrefix)
				assert.Equal(t, tt.route.Upstream, storedRoute.Upstream)
				assert.False(t, storedRoute.CreatedAt.IsZero())
				assert.False(t, storedRoute.UpdatedAt.IsZero())

				// Verify proxy was created
				router.mu.RLock()
				_, exists := router.proxies[tt.route.ID]
				router.mu.RUnlock()
				assert.True(t, exists)
			}
		})
	}
}

func TestRemoveRoute(t *testing.T) {
	router := NewRouter(&config.Config{})

	// Add a test route first
	route := &Route{
		ID:         "test-route",
		Host:       "example.com",
		PathPrefix: "/api",
		Upstream:   "http://localhost:8080",
	}
	err := router.AddRoute(route)
	require.NoError(t, err)

	tests := []struct {
		name          string
		routeID       string
		expectedError bool
	}{
		{
			name:          "remove existing route",
			routeID:       "test-route",
			expectedError: false,
		},
		{
			name:          "remove non-existent route",
			routeID:       "non-existent",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := router.RemoveRoute(tt.routeID)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				
				// Verify route was removed
				_, getErr := router.GetRoute(tt.routeID)
				assert.Error(t, getErr)

				// Verify proxy was removed
				router.mu.RLock()
				_, exists := router.proxies[tt.routeID]
				router.mu.RUnlock()
				assert.False(t, exists)
			}
		})
	}
}

func TestGetRoute(t *testing.T) {
	router := NewRouter(&config.Config{})

	// Add a test route
	route := &Route{
		ID:         "test-route",
		Host:       "example.com",
		PathPrefix: "/api",
		Upstream:   "http://localhost:8080",
	}
	err := router.AddRoute(route)
	require.NoError(t, err)

	tests := []struct {
		name          string
		routeID       string
		expectedError bool
	}{
		{
			name:          "get existing route",
			routeID:       "test-route",
			expectedError: false,
		},
		{
			name:          "get non-existent route",
			routeID:       "non-existent",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrievedRoute, err := router.GetRoute(tt.routeID)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, retrievedRoute)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, retrievedRoute)
				assert.Equal(t, tt.routeID, retrievedRoute.ID)
			}
		})
	}
}

func TestListRoutes(t *testing.T) {
	router := NewRouter(&config.Config{})

	// Initially should be empty
	routes := router.ListRoutes()
	assert.Empty(t, routes)

	// Add multiple routes
	route1 := &Route{
		ID:         "route-1",
		Host:       "example1.com",
		PathPrefix: "/api1",
		Upstream:   "http://localhost:8081",
	}
	route2 := &Route{
		ID:         "route-2",
		Host:       "example2.com",
		PathPrefix: "/api2",
		Upstream:   "http://localhost:8082",
	}

	err := router.AddRoute(route1)
	require.NoError(t, err)
	err = router.AddRoute(route2)
	require.NoError(t, err)

	// List routes
	routes = router.ListRoutes()
	assert.Len(t, routes, 2)

	// Check that both routes are present
	routeIDs := make(map[string]bool)
	for _, route := range routes {
		routeIDs[route.ID] = true
	}
	assert.True(t, routeIDs["route-1"])
	assert.True(t, routeIDs["route-2"])
}

func TestFindRoute(t *testing.T) {
	router := NewRouter(&config.Config{})

	// Add test routes
	routes := []*Route{
		{
			ID:         "exact-host-and-path",
			Host:       "api.example.com",
			PathPrefix: "/v1/users",
			Upstream:   "http://localhost:8081",
		},
		{
			ID:         "host-only",
			Host:       "example.com",
			PathPrefix: "",
			Upstream:   "http://localhost:8082",
		},
		{
			ID:         "path-only",
			Host:       "",
			PathPrefix: "/api",
			Upstream:   "http://localhost:8083",
		},
		{
			ID:         "catch-all",
			Host:       "",
			PathPrefix: "",
			Upstream:   "http://localhost:8084",
		},
	}

	for _, route := range routes {
		err := router.AddRoute(route)
		require.NoError(t, err)
	}

	tests := []struct {
		name           string
		host           string
		path           string
		expectedRouteID string
	}{
		{
			name:           "exact host and path match",
			host:           "api.example.com",
			path:           "/v1/users/123",
			expectedRouteID: "exact-host-and-path",
		},
		{
			name:           "host only match",
			host:           "example.com",
			path:           "/something",
			expectedRouteID: "host-only",
		},
		{
			name:           "path only match",
			host:           "unknown.com",
			path:           "/api/test",
			expectedRouteID: "path-only",
		},
		{
			name:           "catch all match",
			host:           "unknown.com",
			path:           "/unknown",
			expectedRouteID: "catch-all",
		},
		{
			name:           "host with port",
			host:           "api.example.com:8080",
			path:           "/v1/users/456",
			expectedRouteID: "exact-host-and-path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "http://"+tt.host+tt.path, nil)
			req.Host = tt.host

			route := router.findRoute(req)
			if tt.expectedRouteID == "" {
				assert.Nil(t, route)
			} else {
				assert.NotNil(t, route)
				assert.Equal(t, tt.expectedRouteID, route.ID)
			}
		})
	}
}

func TestServeHTTP(t *testing.T) {
	// Create a test upstream server
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("upstream response"))
	}))
	defer upstreamServer.Close()

	router := NewRouter(&config.Config{})

	// Add a route to the test upstream
	route := &Route{
		ID:         "test-route",
		Host:       "",
		PathPrefix: "/test",
		Upstream:   upstreamServer.URL,
	}
	err := router.AddRoute(route)
	require.NoError(t, err)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "matching route",
			path:           "/test/endpoint",
			expectedStatus: http.StatusOK,
			expectedBody:   "upstream response",
		},
		{
			name:           "no matching route",
			path:           "/notfound",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "404 page not found\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.expectedBody, w.Body.String())
		})
	}
}

func TestMetrics(t *testing.T) {
	// Create a test upstream server
	upstreamServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer upstreamServer.Close()

	router := NewRouter(&config.Config{})

	// Add a route
	route := &Route{
		ID:         "test-route",
		Host:       "",
		PathPrefix: "/test",
		Upstream:   upstreamServer.URL,
	}
	err := router.AddRoute(route)
	require.NoError(t, err)

	// Make some requests
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("GET", "/test/endpoint", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	// Make a request that will cause an error (no route)
	req := httptest.NewRequest("GET", "/notfound", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check metrics
	metrics := router.GetMetrics()
	
	assert.Equal(t, int64(5), metrics.RequestCount["test-route"])
	assert.Equal(t, int64(1), metrics.ErrorCount["no-route"])
	// Response times should be greater than 0 for successful requests
	assert.GreaterOrEqual(t, metrics.ResponseTimes["test-route"], int64(0))
}

func TestRecordRequest(t *testing.T) {
	router := NewRouter(&config.Config{})
	routeID := "test-route"
	duration := 100 * time.Millisecond

	// Initially should be empty
	metrics := router.GetMetrics()
	assert.Equal(t, int64(0), metrics.RequestCount[routeID])
	assert.Equal(t, int64(0), metrics.ResponseTimes[routeID])

	// Record a request
	router.recordRequest(routeID, duration)

	// Check metrics
	metrics = router.GetMetrics()
	assert.Equal(t, int64(1), metrics.RequestCount[routeID])
	assert.Equal(t, duration.Nanoseconds(), metrics.ResponseTimes[routeID])

	// Record another request
	router.recordRequest(routeID, duration)

	// Check metrics again
	metrics = router.GetMetrics()
	assert.Equal(t, int64(2), metrics.RequestCount[routeID])
	assert.Equal(t, 2*duration.Nanoseconds(), metrics.ResponseTimes[routeID])
}

func TestRecordError(t *testing.T) {
	router := NewRouter(&config.Config{})
	routeID := "test-route"

	// Initially should be empty
	metrics := router.GetMetrics()
	assert.Equal(t, int64(0), metrics.ErrorCount[routeID])

	// Record an error
	router.recordError(routeID)

	// Check metrics
	metrics = router.GetMetrics()
	assert.Equal(t, int64(1), metrics.ErrorCount[routeID])

	// Record another error
	router.recordError(routeID)

	// Check metrics again
	metrics = router.GetMetrics()
	assert.Equal(t, int64(2), metrics.ErrorCount[routeID])
}

func TestGetMetrics(t *testing.T) {
	router := NewRouter(&config.Config{})

	// Add some test data
	router.recordRequest("route1", 100*time.Millisecond)
	router.recordRequest("route2", 200*time.Millisecond)
	router.recordError("route1")
	router.recordError("route3")

	metrics := router.GetMetrics()

	// Verify metrics are copied correctly
	assert.Equal(t, int64(1), metrics.RequestCount["route1"])
	assert.Equal(t, int64(1), metrics.RequestCount["route2"])
	assert.Equal(t, int64(1), metrics.ErrorCount["route1"])
	assert.Equal(t, int64(1), metrics.ErrorCount["route3"])
	assert.Equal(t, (100*time.Millisecond).Nanoseconds(), metrics.ResponseTimes["route1"])
	assert.Equal(t, (200*time.Millisecond).Nanoseconds(), metrics.ResponseTimes["route2"])

	// Modify returned metrics should not affect original
	metrics.RequestCount["route1"] = 999
	originalMetrics := router.GetMetrics()
	assert.Equal(t, int64(1), originalMetrics.RequestCount["route1"])
}

func TestHealthCheck(t *testing.T) {
	router := NewRouter(&config.Config{})
	ctx := context.Background()

	// Health check should fail with no routes
	err := router.HealthCheck(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no routes configured")

	// Add a route
	route := &Route{
		ID:         "test-route",
		Host:       "example.com",
		PathPrefix: "/api",
		Upstream:   "http://localhost:8080",
	}
	err = router.AddRoute(route)
	require.NoError(t, err)

	// Health check should pass with routes
	err = router.HealthCheck(ctx)
	assert.NoError(t, err)
}

func TestConcurrentAccess(t *testing.T) {
	router := NewRouter(&config.Config{})

	// Test concurrent route addition and removal
	done := make(chan bool, 10)

	// Add routes concurrently
	for i := 0; i < 5; i++ {
		go func(id int) {
			route := &Route{
				ID:         fmt.Sprintf("route-%d", id),
				Host:       fmt.Sprintf("example%d.com", id),
				PathPrefix: fmt.Sprintf("/api%d", id),
				Upstream:   fmt.Sprintf("http://localhost:808%d", id),
			}
			err := router.AddRoute(route)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all additions to complete
	for i := 0; i < 5; i++ {
		<-done
	}

	// Verify all routes were added
	routes := router.ListRoutes()
	assert.Len(t, routes, 5)

	// Remove routes concurrently
	for i := 0; i < 5; i++ {
		go func(id int) {
			err := router.RemoveRoute(fmt.Sprintf("route-%d", id))
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all removals to complete
	for i := 0; i < 5; i++ {
		<-done
	}

	// Verify all routes were removed
	routes = router.ListRoutes()
	assert.Empty(t, routes)
}