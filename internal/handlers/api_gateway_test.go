package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pavilion-trust/core-broker/internal/config"
)

func TestNewAPIGatewayHandler(t *testing.T) {
	cfg := &config.Config{
		CoreBrokerURL: "http://localhost:8080",
	}

	handler := NewAPIGatewayHandler(cfg)

	if handler == nil {
		t.Fatal("Expected handler to be created")
	}

	if handler.config != cfg {
		t.Error("Expected config to be set correctly")
	}

	if handler.proxy == nil {
		t.Error("Expected proxy to be created")
	}
}

func TestAPIGatewayHandler_HandleAPIRequest(t *testing.T) {
	// Create a test server to simulate Core Broker
	coreBrokerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	}))
	defer coreBrokerServer.Close()

	cfg := &config.Config{
		CoreBrokerURL: coreBrokerServer.URL,
	}

	handler := NewAPIGatewayHandler(cfg)

	// Create test request
	req := httptest.NewRequest("POST", "/api/v1/verify", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Handle the request
	handler.HandleAPIRequest(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	expectedBody := `{"status": "success"}`
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, w.Body.String())
	}
}

func TestAPIGatewayHandler_HandleAPIRequest_WithRequestID(t *testing.T) {
	// Create a test server to simulate Core Broker
	coreBrokerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if request ID is forwarded
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			t.Error("Expected request ID to be forwarded")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "success"}`))
	}))
	defer coreBrokerServer.Close()

	cfg := &config.Config{
		CoreBrokerURL: coreBrokerServer.URL,
	}

	handler := NewAPIGatewayHandler(cfg)

	// Create test request with request ID
	req := httptest.NewRequest("POST", "/api/v1/verify", nil)
	req.Header.Set("X-Request-ID", "test-request-id")
	req.Header.Set("Authorization", "Bearer test-token")

	w := httptest.NewRecorder()

	// Handle the request
	handler.HandleAPIRequest(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestAPIGatewayHandler_HandleHealth(t *testing.T) {
	// Create a test server to simulate Core Broker health endpoint
	coreBrokerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "healthy"}`))
	}))
	defer coreBrokerServer.Close()

	cfg := &config.Config{
		CoreBrokerURL: coreBrokerServer.URL,
	}

	handler := NewAPIGatewayHandler(cfg)

	// Create test request
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	// Handle the request
	handler.HandleHealth(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	expectedBody := `{"status": "healthy"}`
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, w.Body.String())
	}

	// Check content type
	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}

func TestAPIGatewayHandler_HandleHealth_CoreBrokerUnreachable(t *testing.T) {
	cfg := &config.Config{
		CoreBrokerURL: "http://localhost:9999", // Unreachable port
	}

	handler := NewAPIGatewayHandler(cfg)

	// Create test request
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	// Handle the request
	handler.HandleHealth(w, req)

	// Check response
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}

	expectedBody := "Core Broker unreachable\n"
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, w.Body.String())
	}
}

func TestNewAPIGatewayHandler_InvalidURL(t *testing.T) {
	cfg := &config.Config{
		CoreBrokerURL: "://invalid-url", // Invalid URL
	}

	// This should panic with an invalid URL
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic with invalid URL")
		}
	}()

	NewAPIGatewayHandler(cfg)
}
