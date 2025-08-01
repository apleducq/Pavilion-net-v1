package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/models"
)

func TestNewDPConnectorService(t *testing.T) {
	cfg := &config.Config{
		DPConnectorURL: "http://localhost:8081",
		Timeout:        30 * time.Second,
	}

	service := NewDPConnectorService(cfg)
	if service == nil {
		t.Fatal("Expected service to be created")
	}

	if service.config != cfg {
		t.Error("Expected config to be set")
	}

	if service.client == nil {
		t.Error("Expected HTTP client to be created")
	}

	if service.pool == nil {
		t.Error("Expected connection pool to be created")
	}

	if service.circuitBreaker == nil {
		t.Error("Expected circuit breaker to be created")
	}
}

func TestDPConnectorService_VerifyWithDP_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Verify content type
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected application/json content type, got %s", r.Header.Get("Content-Type"))
		}

		// Return success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"job_id": "job_123456",
			"status": "completed",
			"verification_result": {
				"verified": true,
				"confidence": 0.95,
				"reason": "Student ID found in database",
				"evidence": ["student_id_match", "enrollment_active"],
				"timestamp": "2025-08-02T07:00:00Z"
			},
			"timestamp": "2025-08-02T07:00:00Z"
		}`))
	}))
	defer server.Close()

	// Create service with test server URL
	cfg := &config.Config{
		DPConnectorURL: server.URL,
		Timeout:        30 * time.Second,
	}
	service := NewDPConnectorService(cfg)

	// Create test request
	req := &models.PrivacyRequest{
		RPID:      "rp_123",
		UserHash:  "hash_abc123",
		ClaimType: "student_verification",
		HashedIdentifiers: map[string]string{
			"student_id": "hash_student_123",
		},
		BloomFilters: map[string]string{
			"student_id": "bloom_filter_data",
		},
	}

	// Test verification
	ctx := context.Background()
	response, err := service.VerifyWithDP(ctx, req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if response == nil {
		t.Fatal("Expected response to be returned")
	}

	if response.JobID != "job_123456" {
		t.Errorf("Expected job ID 'job_123456', got %s", response.JobID)
	}

	if response.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", response.Status)
	}

	if response.VerificationResult == nil {
		t.Fatal("Expected verification result to be present")
	}

	if !response.VerificationResult.Verified {
		t.Error("Expected verification to be true")
	}

	if response.VerificationResult.Confidence != 0.95 {
		t.Errorf("Expected confidence 0.95, got %f", response.VerificationResult.Confidence)
	}
}

func TestDPConnectorService_VerifyWithDP_Timeout(t *testing.T) {
	// Create test server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Delay longer than timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create service with short timeout
	cfg := &config.Config{
		DPConnectorURL: server.URL,
		Timeout:        1 * time.Second, // Short timeout
	}
	service := NewDPConnectorService(cfg)

	req := &models.PrivacyRequest{
		RPID:      "rp_123",
		UserHash:  "hash_abc123",
		ClaimType: "student_verification",
	}

	ctx := context.Background()
	_, err := service.VerifyWithDP(ctx, req)

	if err == nil {
		t.Error("Expected timeout error")
	}
}

func TestDPConnectorService_VerifyWithDP_ServerError(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "Internal server error"}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		DPConnectorURL: server.URL,
		Timeout:        30 * time.Second,
	}
	service := NewDPConnectorService(cfg)

	req := &models.PrivacyRequest{
		RPID:      "rp_123",
		UserHash:  "hash_abc123",
		ClaimType: "student_verification",
	}

	ctx := context.Background()
	_, err := service.VerifyWithDP(ctx, req)

	if err == nil {
		t.Error("Expected error for server error response")
	}
}

func TestDPConnectorService_VerifyWithDP_MalformedResponse(t *testing.T) {
	// Create test server that returns malformed JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"invalid": json`)) // Malformed JSON
	}))
	defer server.Close()

	cfg := &config.Config{
		DPConnectorURL: server.URL,
		Timeout:        30 * time.Second,
	}
	service := NewDPConnectorService(cfg)

	req := &models.PrivacyRequest{
		RPID:      "rp_123",
		UserHash:  "hash_abc123",
		ClaimType: "student_verification",
	}

	ctx := context.Background()
	_, err := service.VerifyWithDP(ctx, req)

	if err == nil {
		t.Error("Expected error for malformed JSON response")
	}
}

func TestCircuitBreaker_CanExecute(t *testing.T) {
	cb := &CircuitBreaker{
		state:     CircuitClosed,
		threshold: 3,
		timeout:   30 * time.Second,
	}

	// Should be able to execute when closed
	if !cb.CanExecute() {
		t.Error("Expected to be able to execute when circuit is closed")
	}

	// Open the circuit
	cb.state = CircuitOpen
	cb.lastFailureTime = time.Now()

	// Should not be able to execute when open
	if cb.CanExecute() {
		t.Error("Expected to not be able to execute when circuit is open")
	}

	// Wait for timeout and check half-open
	cb.lastFailureTime = time.Now().Add(-31 * time.Second)
	if !cb.CanExecute() {
		t.Error("Expected to be able to execute when circuit is half-open")
	}
}

func TestCircuitBreaker_RecordFailure(t *testing.T) {
	cb := &CircuitBreaker{
		state:        CircuitClosed,
		failureCount: 0,
		threshold:    3,
		timeout:      30 * time.Second,
	}

	// Record failures
	cb.RecordFailure()
	if cb.failureCount != 1 {
		t.Errorf("Expected failure count 1, got %d", cb.failureCount)
	}

	cb.RecordFailure()
	cb.RecordFailure()

	// Should open circuit after threshold
	if cb.state != CircuitOpen {
		t.Error("Expected circuit to be open after threshold failures")
	}
}

func TestCircuitBreaker_RecordSuccess(t *testing.T) {
	cb := &CircuitBreaker{
		state:        CircuitHalf,
		failureCount: 3,
		threshold:    3,
		timeout:      30 * time.Second,
	}

	cb.RecordSuccess()

	// Should close circuit on success
	if cb.state != CircuitClosed {
		t.Error("Expected circuit to be closed after success")
	}

	if cb.failureCount != 0 {
		t.Error("Expected failure count to be reset after success")
	}
}

func TestConnectionPool_GetConnection(t *testing.T) {
	pool := &ConnectionPool{
		clients:  make(map[string]*http.Client),
		maxIdle:  100,
		idleTime: 90 * time.Second,
	}

	host := "localhost:8081"
	client := pool.GetConnection(host)

	if client == nil {
		t.Fatal("Expected HTTP client to be created")
	}

	// Should return same client for same host
	client2 := pool.GetConnection(host)
	if client != client2 {
		t.Error("Expected same client for same host")
	}
}

func TestDPConnectorService_HealthCheck(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &config.Config{
		DPConnectorURL: server.URL,
		Timeout:        30 * time.Second,
	}
	service := NewDPConnectorService(cfg)

	ctx := context.Background()
	err := service.HealthCheck(ctx)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestDPConnectorService_HealthCheck_Failure(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cfg := &config.Config{
		DPConnectorURL: server.URL,
		Timeout:        30 * time.Second,
	}
	service := NewDPConnectorService(cfg)

	ctx := context.Background()
	err := service.HealthCheck(ctx)

	if err == nil {
		t.Error("Expected error for unhealthy service")
	}
}

func TestDPConnectorService_GetDPStats(t *testing.T) {
	cfg := &config.Config{
		DPConnectorURL: "http://localhost:8081",
		Timeout:        30 * time.Second,
	}
	service := NewDPConnectorService(cfg)

	stats := service.GetDPStats()

	if stats == nil {
		t.Fatal("Expected stats to be returned")
	}

	// Check that stats contain expected fields
	expectedFields := []string{"circuit_breaker", "connection_pool", "retry_stats"}
	for _, field := range expectedFields {
		if _, exists := stats[field]; !exists {
			t.Errorf("Expected stats to contain field: %s", field)
		}
	}
} 