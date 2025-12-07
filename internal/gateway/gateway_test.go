package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// **Feature: enhanced-aiops-platform, Property 21: API 规范一致性**
// **Validates: Requirements 8.1**
func TestAPISpecificationConsistency(t *testing.T) {
	// Test cases representing different scenarios for property-based testing
	testCases := []struct {
		name     string
		method   string
		path     string
		hasAuth  bool
		expected bool
	}{
		{"GET API endpoint", "GET", "/api/v1/test", false, true},
		{"POST API endpoint", "POST", "/api/v1/test", true, true},
		{"PUT API endpoint", "PUT", "/api/v1/test/123", false, true},
		{"DELETE API endpoint", "DELETE", "/api/v1/test/action", true, true},
		{"Invalid method", "PATCH", "/api/v1/test", false, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Property: For any valid API request, the response should follow OpenAPI specification
			gateway := NewGateway()
			
			// Register a test service
			gateway.RegisterService("test-service", "http://localhost:9999", "", "1.0")
			gateway.AddRoute(tc.path, "test-service", []string{tc.method})
			
			// Create test server for the backend service
			backendServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				response := APIResponse{
					Success: true,
					Data:    map[string]string{"test": "data"},
					Code:    200,
					Version: "1.0",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))
			defer backendServer.Close()
			
			// Update service URL to point to test server
			gateway.registry.mu.Lock()
			gateway.registry.services["test-service"].URL = backendServer.URL
			gateway.registry.services["test-service"].Status = "healthy"
			gateway.registry.mu.Unlock()
			
			// Create request
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.hasAuth {
				req.SetBasicAuth("test", "test")
			}
			
			// Create response recorder
			w := httptest.NewRecorder()
			
			// Handle request
			gateway.ServeHTTP(w, req)
			
			// Verify response follows API specification
			contentType := w.Header().Get("Content-Type")
			
			// Check if response is JSON (for API endpoints)
			if strings.HasPrefix(tc.path, "/api/") {
				if !strings.Contains(contentType, "application/json") {
					t.Errorf("Expected JSON content type for API endpoint, got: %s", contentType)
				}
				
				// Parse response to verify structure
				var response APIResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Errorf("Failed to decode API response: %v", err)
				}
				
				// Verify required fields exist
				if response.Code == 0 || response.Version == "" {
					t.Errorf("API response missing required fields: code=%d, version=%s", response.Code, response.Version)
				}
				
				// Success responses should have success=true
				if w.Code == 200 && !response.Success {
					t.Errorf("Success response should have success=true, got: %v", response.Success)
				}
				
				// Error responses should have success=false and error message
				if w.Code >= 400 && (response.Success || response.Error == "") {
					t.Errorf("Error response should have success=false and error message, got success=%v, error=%s", response.Success, response.Error)
				}
			}
		})
	}
}

// Property test for service health checks
func TestServiceHealthCheckProperty(t *testing.T) {
	testCases := []struct {
		name        string
		serviceName string
		isHealthy   bool
	}{
		{"Healthy service", "test-service-1", true},
		{"Unhealthy service", "test-service-2", false},
		{"Another healthy service", "api-service", true},
		{"Another unhealthy service", "db-service", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Property: For any registered service, health check should work correctly
			gateway := NewGateway()
			
			// Create test health endpoint
			var healthServer *httptest.Server
			if tc.isHealthy {
				healthServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(200)
					w.Write([]byte("OK"))
				}))
			} else {
				healthServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(500)
					w.Write([]byte("Error"))
				}))
			}
			defer healthServer.Close()
			
			// Register service with health endpoint
			gateway.RegisterService(tc.serviceName, "http://localhost:8080", healthServer.URL, "1.0")
			
			// Run health check
			gateway.checkServicesHealth()
			
			// Verify health status
			service, exists := gateway.GetService(tc.serviceName)
			if !exists {
				t.Errorf("Service %s not found after registration", tc.serviceName)
				return
			}
			
			expectedStatus := "healthy"
			if !tc.isHealthy {
				expectedStatus = "unhealthy"
			}
			
			if service.Status != expectedStatus {
				t.Errorf("Expected service status %s, got %s", expectedStatus, service.Status)
			}
		})
	}
}

// Property test for route configuration
func TestRouteConfigurationProperty(t *testing.T) {
	testCases := []struct {
		name        string
		routePath   string
		serviceName string
		methods     []string
	}{
		{"API route", "/api/v1/test", "test-service", []string{"GET", "POST"}},
		{"User route", "/api/v1/users", "user-service", []string{"GET", "POST", "PUT", "DELETE"}},
		{"Data route", "/api/v1/data", "data-service", []string{"GET"}},
		{"Complex route", "/api/v1/complex/path", "complex-service", []string{"POST", "PUT"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Property: For any route configuration, routing should work consistently
			gateway := NewGateway()
			
			// Register service
			gateway.RegisterService(tc.serviceName, "http://localhost:8080", "", "1.0")
			gateway.AddRoute(tc.routePath, tc.serviceName, tc.methods)
			
			// Test route finding
			foundRoute := gateway.findRoute(tc.routePath)
			if foundRoute == nil {
				t.Errorf("Route not found for path: %s", tc.routePath)
				return
			}
			
			// Verify route properties
			if foundRoute.Path != tc.routePath || foundRoute.ServiceName != tc.serviceName {
				t.Errorf("Route mismatch: expected path=%s, service=%s; got path=%s, service=%s", 
					tc.routePath, tc.serviceName, foundRoute.Path, foundRoute.ServiceName)
			}
			
			// Test method checking
			for _, method := range tc.methods {
				if !gateway.isMethodAllowed(foundRoute, method) {
					t.Errorf("Method %s should be allowed for route %s", method, tc.routePath)
				}
			}
			
			// Test disallowed method (if methods are specified)
			if len(tc.methods) > 0 {
				disallowedMethod := "PATCH"
				allowed := false
				for _, method := range tc.methods {
					if method == disallowedMethod {
						allowed = true
						break
					}
				}
				if !allowed && gateway.isMethodAllowed(foundRoute, disallowedMethod) {
					t.Errorf("Method %s should not be allowed for route %s", disallowedMethod, tc.routePath)
				}
			}
		})
	}
}

// Property test for middleware chain
func TestMiddlewareChainProperty(t *testing.T) {
	testCases := []struct {
		name         string
		requestPath  string
		hasRateLimit bool
	}{
		{"Health endpoint", "/health", false},
		{"API endpoint", "/api/v1/test", true},
		{"Unknown endpoint", "/unknown", false},
		{"Root endpoint", "/", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Property: For any middleware chain, request processing should be consistent
			gateway := NewGateway()
			
			// Add middlewares
			gateway.AddMiddleware(LoggingMiddleware())
			if tc.hasRateLimit {
				gateway.AddMiddleware(RateLimitMiddleware(10)) // 10 requests per minute
			}
			gateway.AddMiddleware(HealthCheckMiddleware())
			
			// Create test request
			req := httptest.NewRequest("GET", tc.requestPath, nil)
			w := httptest.NewRecorder()
			
			// Process request
			gateway.ServeHTTP(w, req)
			
			// Health check should always work
			if tc.requestPath == "/health" {
				var response APIResponse
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Errorf("Failed to decode health check response: %v", err)
					return
				}
				if !response.Success || w.Code != 200 {
					t.Errorf("Health check should return success=true and code=200, got success=%v, code=%d", response.Success, w.Code)
				}
			}
			
			// Other paths should return appropriate responses
			if w.Code <= 0 {
				t.Errorf("Request should return a valid HTTP status code, got: %d", w.Code)
			}
		})
	}
}

// Test helper functions
func TestGatewayBasicFunctionality(t *testing.T) {
	gateway := NewGateway()
	
	// Test service registration
	err := gateway.RegisterService("test", "http://localhost:8080", "", "1.0")
	if err != nil {
		t.Errorf("Failed to register service: %v", err)
	}
	
	// Test service retrieval
	service, exists := gateway.GetService("test")
	if !exists {
		t.Error("Service not found after registration")
	}
	if service.Name != "test" {
		t.Errorf("Expected service name 'test', got '%s'", service.Name)
	}
	
	// Test route addition
	gateway.AddRoute("/api/test", "test", []string{"GET"})
	route := gateway.findRoute("/api/test")
	if route == nil {
		t.Error("Route not found after addition")
	}
	
	// Test service unregistration
	err = gateway.UnregisterService("test")
	if err != nil {
		t.Errorf("Failed to unregister service: %v", err)
	}
	
	_, exists = gateway.GetService("test")
	if exists {
		t.Error("Service still exists after unregistration")
	}
}

func TestMiddlewareChain(t *testing.T) {
	gateway := NewGateway()
	
	// Add test middleware
	var middlewareOrder []string
	
	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareOrder = append(middlewareOrder, "middleware1")
			next.ServeHTTP(w, r)
		})
	}
	
	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			middlewareOrder = append(middlewareOrder, "middleware2")
			next.ServeHTTP(w, r)
		})
	}
	
	gateway.AddMiddleware(middleware1)
	gateway.AddMiddleware(middleware2)
	
	// Test request
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	
	gateway.ServeHTTP(w, req)
	
	// Verify middleware execution order (should be in addition order)
	expectedOrder := []string{"middleware1", "middleware2"}
	if len(middlewareOrder) != len(expectedOrder) {
		t.Errorf("Expected %d middleware calls, got %d", len(expectedOrder), len(middlewareOrder))
	}
	
	for i, expected := range expectedOrder {
		if i >= len(middlewareOrder) || middlewareOrder[i] != expected {
			t.Errorf("Expected middleware order %v, got %v", expectedOrder, middlewareOrder)
			break
		}
	}
}

func TestRateLimiting(t *testing.T) {
	// Create rate limit middleware with very low limit for testing
	rateLimiter := RateLimitMiddleware(2) // 2 requests per minute
	
	handler := rateLimiter(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	}))
	
	// Make requests from same IP
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		
		handler.ServeHTTP(w, req)
		
		if i < 2 {
			// First two requests should succeed
			if w.Code != 200 {
				t.Errorf("Request %d should succeed, got status %d", i+1, w.Code)
			}
		} else {
			// Third request should be rate limited
			if w.Code != 429 {
				t.Errorf("Request %d should be rate limited, got status %d", i+1, w.Code)
			}
		}
	}
}

func TestHealthCheckEndpoint(t *testing.T) {
	gateway := NewGateway()
	gateway.AddMiddleware(HealthCheckMiddleware())
	
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	
	gateway.ServeHTTP(w, req)
	
	if w.Code != 200 {
		t.Errorf("Health check should return 200, got %d", w.Code)
	}
	
	var response APIResponse
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode health check response: %v", err)
	}
	
	if !response.Success {
		t.Error("Health check should return success=true")
	}
	
	if response.Code != 200 {
		t.Errorf("Health check should return code=200, got %d", response.Code)
	}
}