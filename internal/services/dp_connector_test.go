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
		DPTimeout:      30 * time.Second,
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
		DPTimeout:      30 * time.Second,
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
		DPTimeout:      1 * time.Second, // Short timeout
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
		DPTimeout:      30 * time.Second,
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
		DPTimeout:      30 * time.Second,
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
		DPTimeout:      30 * time.Second,
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
		DPTimeout:      30 * time.Second,
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
		DPTimeout:      30 * time.Second,
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

func TestConnectionPool_EnhancedFeatures(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("healthy"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create connection pool
	pool := &ConnectionPool{
		clients:  make(map[string]*http.Client),
		maxIdle:  100,
		idleTime: 90 * time.Second,
	}

	// Test 1: GetConnection with enhanced timeout configuration
	t.Run("GetConnection with timeout config", func(t *testing.T) {
		client := pool.GetConnection(server.URL)
		if client == nil {
			t.Fatal("Expected client to be created")
		}

		// Verify timeout configuration
		timeoutConfig := pool.getTimeoutConfig()
		if timeoutConfig.ConnectTimeout != 30*time.Second {
			t.Errorf("Expected connect timeout 30s, got %v", timeoutConfig.ConnectTimeout)
		}
		if timeoutConfig.ReadTimeout != 30*time.Second {
			t.Errorf("Expected read timeout 30s, got %v", timeoutConfig.ReadTimeout)
		}
	})

	// Test 2: Health check functionality
	t.Run("PerformHealthCheck", func(t *testing.T) {
		// Extract host from test server URL
		host := server.URL[7:] // Remove "http://" prefix

		err := pool.PerformHealthCheck(host)
		if err != nil {
			t.Errorf("Expected health check to succeed, got error: %v", err)
		}

		// Verify health check was recorded
		healthCheck, exists := pool.healthChecks[host]
		if !exists {
			t.Fatal("Expected health check to be recorded")
		}

		if !healthCheck.IsHealthy {
			t.Error("Expected connection to be healthy")
		}

		if healthCheck.SuccessCount != 1 {
			t.Errorf("Expected success count 1, got %d", healthCheck.SuccessCount)
		}
	})

	// Test 3: Load balancing functionality
	t.Run("GetHealthyConnection with load balancing", func(t *testing.T) {
		hosts := []string{"host1.example.com", "host2.example.com", "host3.example.com"}

		client, err := pool.GetHealthyConnection(hosts)
		if err != nil {
			t.Errorf("Expected to get healthy connection, got error: %v", err)
		}

		if client == nil {
			t.Fatal("Expected client to be returned")
		}

		// Verify load balancer was initialized
		if pool.loadBalancer == nil {
			t.Fatal("Expected load balancer to be initialized")
		}

		if pool.loadBalancer.strategy != StrategyHealthCheck {
			t.Errorf("Expected strategy %s, got %s", StrategyHealthCheck, pool.loadBalancer.strategy)
		}
	})

	// Test 4: Round-robin load balancing
	t.Run("Round-robin load balancing", func(t *testing.T) {
		hosts := []string{"host1", "host2", "host3"}

		// Test multiple calls to ensure round-robin behavior
		clients := make([]*http.Client, 3)
		for i := 0; i < 3; i++ {
			client, err := pool.getConnectionRoundRobin(hosts)
			if err != nil {
				t.Errorf("Expected to get client, got error: %v", err)
			}
			clients[i] = client
		}

		// All clients should be different (from different hosts)
		if clients[0] == clients[1] || clients[1] == clients[2] || clients[0] == clients[2] {
			t.Error("Expected different clients from round-robin")
		}
	})

	// Test 5: Health check statistics
	t.Run("GetConnectionHealthStats", func(t *testing.T) {
		stats := pool.GetConnectionHealthStats()
		if len(stats) == 0 {
			t.Error("Expected health stats to be available")
		}

		// Verify stats structure
		for host, stat := range stats {
			statMap, ok := stat.(map[string]interface{})
			if !ok {
				t.Errorf("Expected stat to be map, got %T", stat)
				continue
			}

			// Check required fields
			requiredFields := []string{"is_healthy", "last_check", "response_time", "error_count", "success_count"}
			for _, field := range requiredFields {
				if _, exists := statMap[field]; !exists {
					t.Errorf("Expected field %s in health stats for host %s", field, host)
				}
			}
		}
	})

	// Test 6: Enhanced connection pool stats
	t.Run("GetConnectionPoolStats with health data", func(t *testing.T) {
		stats := pool.GetConnectionPoolStats()

		// Verify enhanced stats
		if _, exists := stats["healthy_connections"]; !exists {
			t.Error("Expected healthy_connections in stats")
		}

		if _, exists := stats["health_stats"]; !exists {
			t.Error("Expected health_stats in stats")
		}

		// Verify total connections
		if total, ok := stats["total_connections"].(int); !ok || total < 1 {
			t.Error("Expected total_connections to be positive integer")
		}
	})
}

func TestConnectionPool_TimeoutHandling(t *testing.T) {
	pool := &ConnectionPool{
		clients:  make(map[string]*http.Client),
		maxIdle:  100,
		idleTime: 90 * time.Second,
	}

	t.Run("Timeout configuration", func(t *testing.T) {
		timeoutConfig := pool.getTimeoutConfig()

		// Verify default timeout values
		expectedTimeouts := map[string]time.Duration{
			"ConnectTimeout":   30 * time.Second,
			"ReadTimeout":      30 * time.Second,
			"WriteTimeout":     30 * time.Second,
			"IdleTimeout":      90 * time.Second,
			"KeepAliveTimeout": 30 * time.Second,
		}

		if timeoutConfig.ConnectTimeout != expectedTimeouts["ConnectTimeout"] {
			t.Errorf("Expected ConnectTimeout %v, got %v", expectedTimeouts["ConnectTimeout"], timeoutConfig.ConnectTimeout)
		}
		if timeoutConfig.ReadTimeout != expectedTimeouts["ReadTimeout"] {
			t.Errorf("Expected ReadTimeout %v, got %v", expectedTimeouts["ReadTimeout"], timeoutConfig.ReadTimeout)
		}
		if timeoutConfig.WriteTimeout != expectedTimeouts["WriteTimeout"] {
			t.Errorf("Expected WriteTimeout %v, got %v", expectedTimeouts["WriteTimeout"], timeoutConfig.WriteTimeout)
		}
		if timeoutConfig.IdleTimeout != expectedTimeouts["IdleTimeout"] {
			t.Errorf("Expected IdleTimeout %v, got %v", expectedTimeouts["IdleTimeout"], timeoutConfig.IdleTimeout)
		}
		if timeoutConfig.KeepAliveTimeout != expectedTimeouts["KeepAliveTimeout"] {
			t.Errorf("Expected KeepAliveTimeout %v, got %v", expectedTimeouts["KeepAliveTimeout"], timeoutConfig.KeepAliveTimeout)
		}
	})
}

func TestConnectionPool_LoadBalancingStrategies(t *testing.T) {
	pool := &ConnectionPool{
		clients:  make(map[string]*http.Client),
		maxIdle:  100,
		idleTime: 90 * time.Second,
	}

	hosts := []string{"host1", "host2", "host3"}

	t.Run("Health check strategy", func(t *testing.T) {
		// Set up some mock health data
		pool.healthChecks = map[string]*HealthCheck{
			"host1": {IsHealthy: true, ResponseTime: 100 * time.Millisecond},
			"host2": {IsHealthy: false, ResponseTime: 200 * time.Millisecond},
			"host3": {IsHealthy: true, ResponseTime: 50 * time.Millisecond},
		}

		client, err := pool.getConnectionHealthCheck(hosts)
		if err != nil {
			t.Errorf("Expected to get healthy connection, got error: %v", err)
		}

		if client == nil {
			t.Fatal("Expected client to be returned")
		}
	})

	t.Run("Least connections strategy", func(t *testing.T) {
		client, err := pool.getConnectionLeastConnections(hosts)
		if err != nil {
			t.Errorf("Expected to get connection, got error: %v", err)
		}

		if client == nil {
			t.Fatal("Expected client to be returned")
		}
	})

	t.Run("No healthy hosts", func(t *testing.T) {
		// Set up all hosts as unhealthy
		pool.healthChecks = map[string]*HealthCheck{
			"host1": {IsHealthy: false},
			"host2": {IsHealthy: false},
			"host3": {IsHealthy: false},
		}

		_, err := pool.getConnectionHealthCheck(hosts)
		if err == nil {
			t.Error("Expected error when no healthy hosts available")
		}
	})
}

func TestDPConnectorService_EnhancedConnectionManagement(t *testing.T) {
	cfg := &config.Config{
		DPConnectorURL: "http://localhost:8080",
		DPTimeout:      30 * time.Second,
	}

	service := NewDPConnectorService(cfg)

	t.Run("Enhanced connection pool initialization", func(t *testing.T) {
		if service.pool == nil {
			t.Fatal("Expected connection pool to be initialized")
		}

		// Verify pool has enhanced features
		if service.pool.healthChecks == nil {
			t.Error("Expected health checks map to be initialized")
		}

		if service.pool.timeoutConfig == nil {
			t.Error("Expected timeout config to be initialized")
		}
	})

	t.Run("Circuit breaker integration", func(t *testing.T) {
		if service.circuitBreaker == nil {
			t.Fatal("Expected circuit breaker to be initialized")
		}

		// Test circuit breaker stats
		stats := service.circuitBreaker.GetCircuitBreakerStats()
		if len(stats) == 0 {
			t.Error("Expected circuit breaker stats")
		}
	})

	t.Run("Retry configuration", func(t *testing.T) {
		if service.retryConfig == nil {
			t.Fatal("Expected retry config to be initialized")
		}

		// Verify retry configuration
		if service.retryConfig.MaxRetries != 3 {
			t.Errorf("Expected MaxRetries 3, got %d", service.retryConfig.MaxRetries)
		}

		if service.retryConfig.BaseDelay != 1*time.Second {
			t.Errorf("Expected BaseDelay 1s, got %v", service.retryConfig.BaseDelay)
		}
	})
}

func TestAuthenticator_AuthenticationMethods(t *testing.T) {
	t.Run("API Key Authentication", func(t *testing.T) {
		config := &AuthenticationConfig{
			APIKey:     "test_api_key_123",
			AuthMethod: AuthMethodAPIKey,
		}
		auth := NewAuthenticator(config)

		req, err := http.NewRequest("GET", "https://example.com/test", nil)
		if err != nil {
			t.Fatal(err)
		}

		err = auth.AuthenticateRequest(req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Check that authentication headers are set
		if req.Header.Get("Authorization") == "" && req.Header.Get("X-API-Key") == "" {
			t.Error("Expected authentication headers to be set")
		}
	})

	t.Run("OAuth2 Authentication", func(t *testing.T) {
		config := &AuthenticationConfig{
			OAuth2: &OAuth2Config{
				ClientID:     "test_client_id",
				ClientSecret: "test_client_secret",
				TokenURL:     "https://oauth.example.com/token",
				Scopes:       []string{"read", "write"},
			},
			AuthMethod: AuthMethodOAuth2,
		}
		auth := NewAuthenticator(config)

		req, err := http.NewRequest("GET", "https://example.com/test", nil)
		if err != nil {
			t.Fatal(err)
		}

		err = auth.AuthenticateRequest(req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Check that OAuth2 token is set
		if req.Header.Get("Authorization") == "" {
			t.Error("Expected OAuth2 authorization header to be set")
		}
	})

	t.Run("mTLS Authentication", func(t *testing.T) {
		config := &AuthenticationConfig{
			MTLS: &MTLSConfig{
				CertFile: "test.crt",
				KeyFile:  "test.key",
				CAFile:   "ca.crt",
			},
			AuthMethod: AuthMethodMTLS,
		}
		auth := NewAuthenticator(config)

		req, err := http.NewRequest("GET", "https://example.com/test", nil)
		if err != nil {
			t.Fatal(err)
		}

		err = auth.AuthenticateRequest(req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Check that mTLS header is set
		if req.Header.Get("X-Client-Cert") == "" {
			t.Error("Expected mTLS client cert header to be set")
		}
	})

	t.Run("JWT Authentication", func(t *testing.T) {
		config := &AuthenticationConfig{
			JWT: &JWTConfig{
				Secret:     "test_secret",
				Issuer:     "test_issuer",
				Audience:   "test_audience",
				Expiration: 1 * time.Hour,
			},
			AuthMethod: AuthMethodJWT,
		}
		auth := NewAuthenticator(config)

		req, err := http.NewRequest("GET", "https://example.com/test", nil)
		if err != nil {
			t.Fatal(err)
		}

		err = auth.AuthenticateRequest(req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Check that JWT token is set
		if req.Header.Get("Authorization") == "" {
			t.Error("Expected JWT authorization header to be set")
		}
	})

	t.Run("No Authentication", func(t *testing.T) {
		config := &AuthenticationConfig{
			AuthMethod: AuthMethodNone,
		}
		auth := NewAuthenticator(config)

		req, err := http.NewRequest("GET", "https://example.com/test", nil)
		if err != nil {
			t.Fatal(err)
		}

		err = auth.AuthenticateRequest(req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Check that no authentication headers are set
		if req.Header.Get("Authorization") != "" || req.Header.Get("X-API-Key") != "" {
			t.Error("Expected no authentication headers to be set")
		}
	})
}

func TestAuthenticator_TokenValidation(t *testing.T) {
	t.Run("JWT Token Validation", func(t *testing.T) {
		config := &AuthenticationConfig{
			JWT: &JWTConfig{
				Secret: "test_secret",
			},
			AuthMethod: AuthMethodJWT,
		}
		auth := NewAuthenticator(config)

		// Test valid token
		err := auth.ValidateToken("mock_jwt_token_valid")
		if err != nil {
			t.Errorf("Expected no error for valid token, got %v", err)
		}

		// Test invalid token
		err = auth.ValidateToken("invalid_token")
		if err == nil {
			t.Error("Expected error for invalid token")
		}

		// Test empty token
		err = auth.ValidateToken("")
		if err == nil {
			t.Error("Expected error for empty token")
		}
	})

	t.Run("API Key Validation", func(t *testing.T) {
		config := &AuthenticationConfig{
			APIKey:     "test_api_key_123",
			AuthMethod: AuthMethodAPIKey,
		}
		auth := NewAuthenticator(config)

		// Test valid API key
		err := auth.ValidateToken("test_api_key_123")
		if err != nil {
			t.Errorf("Expected no error for valid API key, got %v", err)
		}

		// Test invalid API key
		err = auth.ValidateToken("invalid_api_key")
		if err == nil {
			t.Error("Expected error for invalid API key")
		}

		// Test empty API key
		err = auth.ValidateToken("")
		if err == nil {
			t.Error("Expected error for empty API key")
		}
	})
}

func TestAuthenticator_AuthenticationFlow(t *testing.T) {
	t.Run("Test Authentication Flow", func(t *testing.T) {
		config := &AuthenticationConfig{
			APIKey:     "test_api_key_123",
			AuthMethod: AuthMethodAPIKey,
		}
		auth := NewAuthenticator(config)

		err := auth.TestAuthenticationFlow()
		if err != nil {
			t.Errorf("Expected authentication flow test to pass, got %v", err)
		}
	})

	t.Run("Test OAuth2 Authentication Flow", func(t *testing.T) {
		config := &AuthenticationConfig{
			OAuth2: &OAuth2Config{
				ClientID:     "test_client_id",
				ClientSecret: "test_client_secret",
				TokenURL:     "https://oauth.example.com/token",
			},
			AuthMethod: AuthMethodOAuth2,
		}
		auth := NewAuthenticator(config)

		err := auth.TestAuthenticationFlow()
		if err != nil {
			t.Errorf("Expected OAuth2 authentication flow test to pass, got %v", err)
		}
	})

	t.Run("Test JWT Authentication Flow", func(t *testing.T) {
		config := &AuthenticationConfig{
			JWT: &JWTConfig{
				Secret:     "test_secret",
				Issuer:     "test_issuer",
				Audience:   "test_audience",
				Expiration: 1 * time.Hour,
			},
			AuthMethod: AuthMethodJWT,
		}
		auth := NewAuthenticator(config)

		err := auth.TestAuthenticationFlow()
		if err != nil {
			t.Errorf("Expected JWT authentication flow test to pass, got %v", err)
		}
	})

	t.Run("Test mTLS Authentication Flow", func(t *testing.T) {
		config := &AuthenticationConfig{
			MTLS: &MTLSConfig{
				CertFile: "test.crt",
				KeyFile:  "test.key",
				CAFile:   "ca.crt",
			},
			AuthMethod: AuthMethodMTLS,
		}
		auth := NewAuthenticator(config)

		err := auth.TestAuthenticationFlow()
		if err != nil {
			t.Errorf("Expected mTLS authentication flow test to pass, got %v", err)
		}
	})
}

func TestDPConnectorService_AuthenticationIntegration(t *testing.T) {
	t.Run("Service with API Key Authentication", func(t *testing.T) {
		cfg := &config.Config{
			DPConnectorURL:   "http://localhost:8080",
			DPConnectorToken: "test_api_key_123",
			DPTimeout:        30 * time.Second,
		}

		service := NewDPConnectorService(cfg)
		if service.authenticator == nil {
			t.Fatal("Expected authenticator to be initialized")
		}

		// Test that the authenticator is configured correctly
		if service.authenticator.config.AuthMethod != AuthMethodAPIKey {
			t.Errorf("Expected AuthMethodAPIKey, got %s", service.authenticator.config.AuthMethod)
		}

		if service.authenticator.config.APIKey != "test_api_key_123" {
			t.Errorf("Expected API key 'test_api_key_123', got %s", service.authenticator.config.APIKey)
		}
	})

	t.Run("Service with No Authentication", func(t *testing.T) {
		cfg := &config.Config{
			DPConnectorURL: "http://localhost:8080",
			DPTimeout:      30 * time.Second,
		}

		service := NewDPConnectorService(cfg)
		if service.authenticator == nil {
			t.Fatal("Expected authenticator to be initialized")
		}

		// Test that the authenticator defaults to no authentication method
		if service.authenticator.config.AuthMethod != AuthMethodNone {
			t.Errorf("Expected AuthMethodNone, got %s", service.authenticator.config.AuthMethod)
		}
	})
}

func TestIntegrationAdapters(t *testing.T) {
	t.Run("REST Adapter", func(t *testing.T) {
		// Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status": "healthy"}`))
			} else if r.URL.Path == "/api/v1/verify" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"result": "success", "data": "test_response"}`))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer server.Close()

		adapter := NewRESTAdapter()
		config := &AdapterConfig{
			URL:         server.URL,
			Timeout:     30 * time.Second,
			AdapterType: AdapterTypeREST,
		}

		// Test connection
		err := adapter.Connect(context.Background(), config)
		if err != nil {
			t.Errorf("Expected no error connecting to REST API, got %v", err)
		}

		// Test request
		request := map[string]interface{}{
			"test": "data",
		}
		response, err := adapter.SendRequest(context.Background(), request)
		if err != nil {
			t.Errorf("Expected no error sending REST request, got %v", err)
		}

		// Verify response
		responseMap, ok := response.(map[string]interface{})
		if !ok {
			t.Error("Expected response to be map[string]interface{}")
		}

		if responseMap["result"] != "success" {
			t.Errorf("Expected result 'success', got %v", responseMap["result"])
		}

		// Test close
		err = adapter.Close()
		if err != nil {
			t.Errorf("Expected no error closing REST adapter, got %v", err)
		}

		// Test adapter type
		if adapter.GetAdapterType() != string(AdapterTypeREST) {
			t.Errorf("Expected adapter type %s, got %s", AdapterTypeREST, adapter.GetAdapterType())
		}
	})

	t.Run("GraphQL Adapter", func(t *testing.T) {
		// Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"data": {"test": "graphql_response"}}`))
		}))
		defer server.Close()

		adapter := NewGraphQLAdapter()
		config := &AdapterConfig{
			URL:         server.URL,
			Timeout:     30 * time.Second,
			AdapterType: AdapterTypeGraphQL,
		}

		// Test connection
		err := adapter.Connect(context.Background(), config)
		if err != nil {
			t.Errorf("Expected no error connecting to GraphQL API, got %v", err)
		}

		// Test request
		query := `{ test { id name } }`
		response, err := adapter.SendRequest(context.Background(), query)
		if err != nil {
			t.Errorf("Expected no error sending GraphQL request, got %v", err)
		}

		// Verify response
		responseMap, ok := response.(map[string]interface{})
		if !ok {
			t.Error("Expected response to be map[string]interface{}")
		}

		if responseMap["data"] == nil {
			t.Error("Expected data in GraphQL response")
		}

		// Test close
		err = adapter.Close()
		if err != nil {
			t.Errorf("Expected no error closing GraphQL adapter, got %v", err)
		}

		// Test adapter type
		if adapter.GetAdapterType() != string(AdapterTypeGraphQL) {
			t.Errorf("Expected adapter type %s, got %s", AdapterTypeGraphQL, adapter.GetAdapterType())
		}
	})

	t.Run("gRPC Adapter", func(t *testing.T) {
		adapter := NewGRPCAdapter()
		config := &AdapterConfig{
			URL:         "localhost:50051",
			Timeout:     30 * time.Second,
			AdapterType: AdapterTypeGRPC,
		}

		// Test connection
		err := adapter.Connect(context.Background(), config)
		if err != nil {
			t.Errorf("Expected no error connecting to gRPC service, got %v", err)
		}

		// Test request
		request := map[string]interface{}{
			"grpc_request": "test_data",
		}
		response, err := adapter.SendRequest(context.Background(), request)
		if err != nil {
			t.Errorf("Expected no error sending gRPC request, got %v", err)
		}

		// Verify response
		responseMap, ok := response.(map[string]interface{})
		if !ok {
			t.Error("Expected response to be map[string]interface{}")
		}

		if responseMap["grpc_response"] != "mock_grpc_response" {
			t.Errorf("Expected grpc_response 'mock_grpc_response', got %v", responseMap["grpc_response"])
		}

		// Test close
		err = adapter.Close()
		if err != nil {
			t.Errorf("Expected no error closing gRPC adapter, got %v", err)
		}

		// Test adapter type
		if adapter.GetAdapterType() != string(AdapterTypeGRPC) {
			t.Errorf("Expected adapter type %s, got %s", AdapterTypeGRPC, adapter.GetAdapterType())
		}
	})

	t.Run("WebSocket Adapter", func(t *testing.T) {
		adapter := NewWebSocketAdapter()
		config := &AdapterConfig{
			URL:         "ws://localhost:8080/ws",
			Timeout:     30 * time.Second,
			AdapterType: AdapterTypeWebSocket,
		}

		// Test connection
		err := adapter.Connect(context.Background(), config)
		if err != nil {
			t.Errorf("Expected no error connecting to WebSocket, got %v", err)
		}

		// Test request
		request := map[string]interface{}{
			"websocket_request": "test_data",
		}
		response, err := adapter.SendRequest(context.Background(), request)
		if err != nil {
			t.Errorf("Expected no error sending WebSocket request, got %v", err)
		}

		// Verify response
		responseMap, ok := response.(map[string]interface{})
		if !ok {
			t.Error("Expected response to be map[string]interface{}")
		}

		if responseMap["websocket_response"] != "mock_websocket_response" {
			t.Errorf("Expected websocket_response 'mock_websocket_response', got %v", responseMap["websocket_response"])
		}

		// Test close
		err = adapter.Close()
		if err != nil {
			t.Errorf("Expected no error closing WebSocket adapter, got %v", err)
		}

		// Test adapter type
		if adapter.GetAdapterType() != string(AdapterTypeWebSocket) {
			t.Errorf("Expected adapter type %s, got %s", AdapterTypeWebSocket, adapter.GetAdapterType())
		}
	})
}

func TestAdapterFactory(t *testing.T) {
	t.Run("REST Adapter Factory", func(t *testing.T) {
		adapter := AdapterFactory(AdapterTypeREST)
		if adapter == nil {
			t.Fatal("Expected REST adapter to be created")
		}

		if adapter.GetAdapterType() != string(AdapterTypeREST) {
			t.Errorf("Expected adapter type %s, got %s", AdapterTypeREST, adapter.GetAdapterType())
		}
	})

	t.Run("GraphQL Adapter Factory", func(t *testing.T) {
		adapter := AdapterFactory(AdapterTypeGraphQL)
		if adapter == nil {
			t.Fatal("Expected GraphQL adapter to be created")
		}

		if adapter.GetAdapterType() != string(AdapterTypeGraphQL) {
			t.Errorf("Expected adapter type %s, got %s", AdapterTypeGraphQL, adapter.GetAdapterType())
		}
	})

	t.Run("gRPC Adapter Factory", func(t *testing.T) {
		adapter := AdapterFactory(AdapterTypeGRPC)
		if adapter == nil {
			t.Fatal("Expected gRPC adapter to be created")
		}

		if adapter.GetAdapterType() != string(AdapterTypeGRPC) {
			t.Errorf("Expected adapter type %s, got %s", AdapterTypeGRPC, adapter.GetAdapterType())
		}
	})

	t.Run("WebSocket Adapter Factory", func(t *testing.T) {
		adapter := AdapterFactory(AdapterTypeWebSocket)
		if adapter == nil {
			t.Fatal("Expected WebSocket adapter to be created")
		}

		if adapter.GetAdapterType() != string(AdapterTypeWebSocket) {
			t.Errorf("Expected adapter type %s, got %s", AdapterTypeWebSocket, adapter.GetAdapterType())
		}
	})

	t.Run("Default Adapter Factory", func(t *testing.T) {
		adapter := AdapterFactory("unknown")
		if adapter == nil {
			t.Fatal("Expected default adapter to be created")
		}

		if adapter.GetAdapterType() != string(AdapterTypeREST) {
			t.Errorf("Expected default adapter type %s, got %s", AdapterTypeREST, adapter.GetAdapterType())
		}
	})
}

func TestAdapterFunctionalityTests(t *testing.T) {
	t.Run("Test REST Adapter Functionality", func(t *testing.T) {
		// Create test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/health" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status": "healthy"}`))
			} else if r.URL.Path == "/api/v1/verify" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"result": "success"}`))
			}
		}))
		defer server.Close()

		adapter := NewRESTAdapter()
		config := &AdapterConfig{
			URL:         server.URL,
			Timeout:     30 * time.Second,
			AdapterType: AdapterTypeREST,
		}

		err := TestAdapterFunctionality(adapter, config)
		if err != nil {
			t.Errorf("Expected adapter functionality test to pass, got %v", err)
		}
	})

	t.Run("Test gRPC Adapter Functionality", func(t *testing.T) {
		adapter := NewGRPCAdapter()
		config := &AdapterConfig{
			URL:         "localhost:50051",
			Timeout:     30 * time.Second,
			AdapterType: AdapterTypeGRPC,
		}

		err := TestAdapterFunctionality(adapter, config)
		if err != nil {
			t.Errorf("Expected gRPC adapter functionality test to pass, got %v", err)
		}
	})

	t.Run("Test WebSocket Adapter Functionality", func(t *testing.T) {
		adapter := NewWebSocketAdapter()
		config := &AdapterConfig{
			URL:         "ws://localhost:8080/ws",
			Timeout:     30 * time.Second,
			AdapterType: AdapterTypeWebSocket,
		}

		err := TestAdapterFunctionality(adapter, config)
		if err != nil {
			t.Errorf("Expected WebSocket adapter functionality test to pass, got %v", err)
		}
	})
}
