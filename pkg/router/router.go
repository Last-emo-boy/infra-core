package router

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/last-emo-boy/infra-core/pkg/config"
)

// Route represents a routing rule
type Route struct {
	ID         string    `json:"id"`
	Host       string    `json:"host"`
	PathPrefix string    `json:"path_prefix"`
	Upstream   string    `json:"upstream"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Router handles HTTP request routing
type Router struct {
	routes  map[string]*Route
	proxies map[string]*httputil.ReverseProxy
	mu      sync.RWMutex
	config  *config.Config
	metrics *Metrics
}

// Metrics holds routing metrics
type Metrics struct {
	RequestCount  map[string]int64 `json:"request_count"`
	ErrorCount    map[string]int64 `json:"error_count"`
	ResponseTimes map[string]int64 `json:"response_times"`
	mu            sync.RWMutex
}

// NewRouter creates a new router instance
func NewRouter(cfg *config.Config) *Router {
	return &Router{
		routes:  make(map[string]*Route),
		proxies: make(map[string]*httputil.ReverseProxy),
		config:  cfg,
		metrics: &Metrics{
			RequestCount:  make(map[string]int64),
			ErrorCount:    make(map[string]int64),
			ResponseTimes: make(map[string]int64),
		},
	}
}

// AddRoute adds a new route
func (r *Router) AddRoute(route *Route) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate upstream URL
	upstream, err := url.Parse(route.Upstream)
	if err != nil {
		return fmt.Errorf("invalid upstream URL: %w", err)
	}

	// Create reverse proxy
	proxy := httputil.NewSingleHostReverseProxy(upstream)

	// Customize proxy behavior
	proxy.Director = func(req *http.Request) {
		req.URL.Scheme = upstream.Scheme
		req.URL.Host = upstream.Host
		req.Host = upstream.Host

		// Add forwarded headers
		req.Header.Set("X-Forwarded-Proto", req.URL.Scheme)
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Real-IP", req.RemoteAddr)
	}

	// Error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, req *http.Request, err error) {
		r.recordError(route.ID)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	// Store route and proxy
	route.UpdatedAt = time.Now()
	if route.CreatedAt.IsZero() {
		route.CreatedAt = time.Now()
	}

	r.routes[route.ID] = route
	r.proxies[route.ID] = proxy

	return nil
}

// RemoveRoute removes a route
func (r *Router) RemoveRoute(routeID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.routes[routeID]; !exists {
		return fmt.Errorf("route not found: %s", routeID)
	}

	delete(r.routes, routeID)
	delete(r.proxies, routeID)

	return nil
}

// GetRoute gets a route by ID
func (r *Router) GetRoute(routeID string) (*Route, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	route, exists := r.routes[routeID]
	if !exists {
		return nil, fmt.Errorf("route not found: %s", routeID)
	}

	return route, nil
}

// ListRoutes returns all routes
func (r *Router) ListRoutes() []*Route {
	r.mu.RLock()
	defer r.mu.RUnlock()

	routes := make([]*Route, 0, len(r.routes))
	for _, route := range r.routes {
		routes = append(routes, route)
	}

	return routes
}

// ServeHTTP implements http.Handler interface
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	start := time.Now()

	// Find matching route
	route := r.findRoute(req)
	if route == nil {
		r.recordError("no-route")
		http.NotFound(w, req)
		return
	}

	// Get proxy for this route
	r.mu.RLock()
	proxy, exists := r.proxies[route.ID]
	r.mu.RUnlock()

	if !exists {
		r.recordError(route.ID)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Record metrics
	r.recordRequest(route.ID, time.Since(start))

	// Proxy the request
	proxy.ServeHTTP(w, req)
}

// findRoute finds the best matching route for a request
func (r *Router) findRoute(req *http.Request) *Route {
	r.mu.RLock()
	defer r.mu.RUnlock()

	host := req.Host
	path := req.URL.Path

	// Remove port from host if present
	if colonIndex := strings.Index(host, ":"); colonIndex != -1 {
		host = host[:colonIndex]
	}

	var bestMatch *Route
	var bestScore int

	for _, route := range r.routes {
		score := 0

		// Check host match
		if route.Host != "" && route.Host == host {
			score += 100
		} else if route.Host != "" {
			continue // Host doesn't match, skip this route
		}

		// Check path prefix match
		if route.PathPrefix != "" {
			if strings.HasPrefix(path, route.PathPrefix) {
				score += len(route.PathPrefix)
			} else {
				continue // Path doesn't match, skip this route
			}
		}

		// Select best match (highest score)
		if bestMatch == nil || score > bestScore {
			bestMatch = route
			bestScore = score
		}
	}

	return bestMatch
}

// recordRequest records request metrics
func (r *Router) recordRequest(routeID string, duration time.Duration) {
	r.metrics.mu.Lock()
	defer r.metrics.mu.Unlock()

	r.metrics.RequestCount[routeID]++
	r.metrics.ResponseTimes[routeID] += duration.Nanoseconds()
}

// recordError records error metrics
func (r *Router) recordError(routeID string) {
	r.metrics.mu.Lock()
	defer r.metrics.mu.Unlock()

	r.metrics.ErrorCount[routeID]++
}

// GetMetrics returns current metrics
func (r *Router) GetMetrics() *Metrics {
	r.metrics.mu.RLock()
	defer r.metrics.mu.RUnlock()

	// Create a copy of metrics
	metrics := &Metrics{
		RequestCount:  make(map[string]int64),
		ErrorCount:    make(map[string]int64),
		ResponseTimes: make(map[string]int64),
	}

	for k, v := range r.metrics.RequestCount {
		metrics.RequestCount[k] = v
	}
	for k, v := range r.metrics.ErrorCount {
		metrics.ErrorCount[k] = v
	}
	for k, v := range r.metrics.ResponseTimes {
		metrics.ResponseTimes[k] = v
	}

	return metrics
}

// HealthCheck checks if the router is healthy
func (r *Router) HealthCheck(ctx context.Context) error {
	r.mu.RLock()
	routeCount := len(r.routes)
	r.mu.RUnlock()

	if routeCount == 0 {
		return fmt.Errorf("no routes configured")
	}

	return nil
}
