package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pavilion-trust/core-broker/internal/config"
)

func TestNewAPIGateway(t *testing.T) {
	cfg := &config.Config{
		APIGatewayPort: "8443",
		CoreBrokerURL:  "http://localhost:8080",
		TLSCertFile:    "testdata/server.crt",
		TLSKeyFile:     "testdata/server.key",
	}

	server := NewAPIGateway(cfg)

	if server == nil {
		t.Fatal("Expected server to be created")
	}

	if server.config != cfg {
		t.Error("Expected config to be set correctly")
	}

	if server.Addr != ":8443" {
		t.Errorf("Expected address :8443, got %s", server.Addr)
	}
}

func TestAPIGateway_Routing(t *testing.T) {
	// Create a test Core Broker server
	coreBrokerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	}))
	defer coreBrokerServer.Close()

	cfg := &config.Config{
		APIGatewayPort: "8443",
		CoreBrokerURL:  coreBrokerServer.URL,
		TLSCertFile:    "testdata/server.crt",
		TLSKeyFile:     "testdata/server.key",
	}

	server := NewAPIGateway(cfg)

	// Test API request routing
	req := httptest.NewRequest("POST", "/api/v1/verify", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	w := httptest.NewRecorder()

	server.Handler.ServeHTTP(w, req)

	// Should be routed to Core Broker
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	expectedBody := `{"status": "success"}`
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, w.Body.String())
	}
}

func TestAPIGateway_HealthCheck(t *testing.T) {
	// Create a test Core Broker server
	coreBrokerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy"}`))
	}))
	defer coreBrokerServer.Close()

	cfg := &config.Config{
		APIGatewayPort: "8443",
		CoreBrokerURL:  coreBrokerServer.URL,
		TLSCertFile:    "testdata/server.crt",
		TLSKeyFile:     "testdata/server.key",
	}

	server := NewAPIGateway(cfg)

	// Test health check endpoint
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	expectedBody := `{"status": "healthy"}`
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, w.Body.String())
	}
}

func TestAPIGateway_SecurityHeaders(t *testing.T) {
	cfg := &config.Config{
		APIGatewayPort: "8443",
		CoreBrokerURL:  "http://localhost:8080",
		TLSCertFile:    "testdata/server.crt",
		TLSKeyFile:     "testdata/server.key",
	}

	server := NewAPIGateway(cfg)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.Handler.ServeHTTP(w, req)

	// Check security headers are present
	expectedHeaders := map[string]string{
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"X-XSS-Protection":          "1; mode=block",
		"Content-Security-Policy":   "default-src 'self'",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
	}

	for header, expectedValue := range expectedHeaders {
		actualValue := w.Header().Get(header)
		if actualValue != expectedValue {
			t.Errorf("Expected header %s to be %s, got %s", header, expectedValue, actualValue)
		}
	}
}

func TestAPIGateway_CORSHeaders(t *testing.T) {
	cfg := &config.Config{
		APIGatewayPort: "8443",
		CoreBrokerURL:  "http://localhost:8080",
		TLSCertFile:    "testdata/server.crt",
		TLSKeyFile:     "testdata/server.key",
	}

	server := NewAPIGateway(cfg)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.Handler.ServeHTTP(w, req)

	// Check CORS headers are present
	expectedHeaders := map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Authorization, X-Request-ID",
		"Access-Control-Max-Age":       "86400",
	}

	for header, expectedValue := range expectedHeaders {
		actualValue := w.Header().Get(header)
		if actualValue != expectedValue {
			t.Errorf("Expected header %s to be %s, got %s", header, expectedValue, actualValue)
		}
	}
}

func TestAPIGateway_RequestID(t *testing.T) {
	cfg := &config.Config{
		APIGatewayPort: "8443",
		CoreBrokerURL:  "http://localhost:8080",
		TLSCertFile:    "testdata/server.crt",
		TLSKeyFile:     "testdata/server.key",
	}

	server := NewAPIGateway(cfg)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.Handler.ServeHTTP(w, req)

	// Check that request ID is added to response
	requestID := w.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("Expected X-Request-ID header to be present")
	}
}

func TestAPIGateway_Shutdown(t *testing.T) {
	cfg := &config.Config{
		APIGatewayPort: "8443",
		CoreBrokerURL:  "http://localhost:8080",
		TLSCertFile:    "testdata/server.crt",
		TLSKeyFile:     "testdata/server.key",
	}

	server := NewAPIGateway(cfg)

	// Test graceful shutdown
	ctx := context.Background()
	err := server.Shutdown(ctx)
	if err != nil {
		t.Errorf("Expected no error on shutdown, got %v", err)
	}
}
