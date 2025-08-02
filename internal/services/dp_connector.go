package services

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/models"
)

// DPConnectorService handles communication with Data Provider Connector
type DPConnectorService struct {
	config *config.Config
	client *http.Client
	// Connection pool for managing connections
	pool *ConnectionPool
	// Circuit breaker for handling failures
	circuitBreaker *CircuitBreaker
	// Retry configuration
	retryConfig *RetryConfig
	// Authentication handler
	authenticator *Authenticator
}

// ConnectionPool manages HTTP connections
type ConnectionPool struct {
	mu       sync.RWMutex
	clients  map[string]*http.Client
	maxIdle  int
	idleTime time.Duration
	// Connection health monitoring
	healthChecks map[string]*HealthCheck
	// Load balancing configuration
	loadBalancer *LoadBalancer
	// Connection timeout configuration
	timeoutConfig *TimeoutConfig
}

// HealthCheck tracks connection health
type HealthCheck struct {
	LastCheck    time.Time
	IsHealthy    bool
	ResponseTime time.Duration
	ErrorCount   int
	SuccessCount int
	LastError    error
}

// LoadBalancer manages connection distribution
type LoadBalancer struct {
	mu           sync.RWMutex
	connections  []string
	currentIndex int
	strategy     LoadBalancingStrategy
}

// LoadBalancingStrategy defines load balancing approach
type LoadBalancingStrategy string

const (
	StrategyRoundRobin       LoadBalancingStrategy = "round_robin"
	StrategyLeastConnections LoadBalancingStrategy = "least_connections"
	StrategyHealthCheck      LoadBalancingStrategy = "health_check"
)

// TimeoutConfig defines connection timeout settings
type TimeoutConfig struct {
	ConnectTimeout   time.Duration
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	IdleTimeout      time.Duration
	KeepAliveTimeout time.Duration
}

// CircuitBreaker implements circuit breaker pattern
type CircuitBreaker struct {
	mu              sync.RWMutex
	failureCount    int
	lastFailureTime time.Time
	state           CircuitState
	threshold       int
	timeout         time.Duration
}

// CircuitState represents the state of the circuit breaker
type CircuitState string

const (
	CircuitClosed CircuitState = "closed"
	CircuitOpen   CircuitState = "open"
	CircuitHalf   CircuitState = "half_open"
)

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxRetries        int
	BaseDelay         time.Duration
	MaxDelay          time.Duration
	BackoffMultiplier float64
}

// DPResponse represents a response from the DP Connector
type DPResponse struct {
	JobID              string                 `json:"job_id"`
	Status             string                 `json:"status"`
	VerificationResult *VerificationResult    `json:"verification_result,omitempty"`
	Error              string                 `json:"error,omitempty"`
	Timestamp          string                 `json:"timestamp"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

// VerificationResult represents the result of a verification
type VerificationResult struct {
	Verified   bool     `json:"verified"`
	Confidence float64  `json:"confidence"`
	Reason     string   `json:"reason,omitempty"`
	Evidence   []string `json:"evidence,omitempty"`
	Timestamp  string   `json:"timestamp"`
}

// NewDPConnectorService creates a new DP connector service
func NewDPConnectorService(cfg *config.Config) *DPConnectorService {
	// Create retry configuration
	retryConfig := &RetryConfig{
		MaxRetries:        3,
		BaseDelay:         1 * time.Second,
		MaxDelay:          30 * time.Second,
		BackoffMultiplier: 2.0,
	}

	// Create connection pool
	pool := &ConnectionPool{
		clients:      make(map[string]*http.Client),
		maxIdle:      100,
		idleTime:     90 * time.Second,
		healthChecks: make(map[string]*HealthCheck),
		timeoutConfig: &TimeoutConfig{
			ConnectTimeout:   30 * time.Second,
			ReadTimeout:      30 * time.Second,
			WriteTimeout:     30 * time.Second,
			IdleTimeout:      90 * time.Second,
			KeepAliveTimeout: 30 * time.Second,
		},
	}

	// Create circuit breaker
	circuitBreaker := &CircuitBreaker{
		failureCount: 0,
		state:        CircuitClosed,
		threshold:    5,
		timeout:      60 * time.Second,
	}

	// Create HTTP client with connection pooling
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
	}

	// Create authenticator with appropriate configuration
	var authConfig *AuthenticationConfig
	if cfg.DPConnectorToken != "" {
		authConfig = &AuthenticationConfig{
			APIKey:     cfg.DPConnectorToken,
			AuthMethod: AuthMethodAPIKey,
		}
	} else {
		authConfig = &AuthenticationConfig{
			AuthMethod: AuthMethodNone,
		}
	}
	authenticator := NewAuthenticator(authConfig)

	return &DPConnectorService{
		config:         cfg,
		client:         client,
		pool:           pool,
		circuitBreaker: circuitBreaker,
		retryConfig:    retryConfig,
		authenticator:  authenticator,
	}
}

// VerifyWithDP sends a verification request to the DP Connector
func (s *DPConnectorService) VerifyWithDP(ctx context.Context, req *models.PrivacyRequest) (*DPResponse, error) {
	// Check circuit breaker state
	if !s.circuitBreaker.CanExecute() {
		return nil, fmt.Errorf("circuit breaker is open, DP connector is unavailable")
	}

	// Prepare request payload
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.config.DPConnectorURL+"/verify", strings.NewReader(string(payload)))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Add authentication
	if err := s.authenticator.AuthenticateRequest(httpReq); err != nil {
		return nil, fmt.Errorf("failed to authenticate request: %w", err)
	}

	// Execute request with retry logic
	var response *DPResponse
	err = s.executeWithRetry(ctx, httpReq, func(resp *http.Response) error {
		var err error
		response, err = s.parseDPResponse(resp)
		return err
	})

	if err != nil {
		s.circuitBreaker.RecordFailure()
		return nil, fmt.Errorf("DP verification failed: %w", err)
	}

	s.circuitBreaker.RecordSuccess()
	return response, nil
}

// executeWithRetry executes a request with exponential backoff retry
func (s *DPConnectorService) executeWithRetry(ctx context.Context, req *http.Request, handler func(*http.Response) error) error {
	var lastErr error

	for attempt := 0; attempt <= s.retryConfig.MaxRetries; attempt++ {
		// Execute request
		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)

			// If this is the last attempt, return the error
			if attempt == s.retryConfig.MaxRetries {
				return lastErr
			}

			// Calculate delay for next attempt
			delay := s.calculateDelay(attempt)

			// Wait before retry
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				continue
			}
		}
		defer resp.Body.Close()

		// Check if response indicates retry is needed
		if resp.StatusCode >= 500 || resp.StatusCode == 429 {
			lastErr = fmt.Errorf("server error, status: %d", resp.StatusCode)

			if attempt == s.retryConfig.MaxRetries {
				return lastErr
			}

			delay := s.calculateDelay(attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				continue
			}
		}

		// Handle successful response
		return handler(resp)
	}

	return lastErr
}

// calculateDelay calculates the delay for exponential backoff
func (s *DPConnectorService) calculateDelay(attempt int) time.Duration {
	delay := time.Duration(float64(s.retryConfig.BaseDelay) * math.Pow(s.retryConfig.BackoffMultiplier, float64(attempt)))
	if delay > s.retryConfig.MaxDelay {
		delay = s.retryConfig.MaxDelay
	}
	return delay
}

// parseDPResponse parses the response from the DP Connector
func (s *DPConnectorService) parseDPResponse(resp *http.Response) (*DPResponse, error) {
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("DP connector returned status %d: %s", resp.StatusCode, string(body))
	}

	var dpResp DPResponse
	if err := json.NewDecoder(resp.Body).Decode(&dpResp); err != nil {
		return nil, fmt.Errorf("failed to decode DP response: %w", err)
	}

	return &dpResp, nil
}

// GetConnection returns a connection from the pool
func (p *ConnectionPool) GetConnection(host string) *http.Client {
	p.mu.RLock()
	if client, exists := p.clients[host]; exists {
		p.mu.RUnlock()
		return client
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check after acquiring write lock
	if client, exists := p.clients[host]; exists {
		return client
	}

	// Create new client for this host with enhanced timeout configuration
	timeoutConfig := p.getTimeoutConfig()
	client := &http.Client{
		Timeout: timeoutConfig.ConnectTimeout,
		Transport: &http.Transport{
			MaxIdleConns:          p.maxIdle,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       p.idleTime,
			ResponseHeaderTimeout: timeoutConfig.ReadTimeout,
			TLSHandshakeTimeout:   10 * time.Second,
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
		},
	}

	p.clients[host] = client

	// Initialize health check for this connection
	if p.healthChecks == nil {
		p.healthChecks = make(map[string]*HealthCheck)
	}
	p.healthChecks[host] = &HealthCheck{
		LastCheck: time.Now(),
		IsHealthy: true,
	}

	return client
}

// getTimeoutConfig returns the timeout configuration
func (p *ConnectionPool) getTimeoutConfig() *TimeoutConfig {
	if p.timeoutConfig == nil {
		p.timeoutConfig = &TimeoutConfig{
			ConnectTimeout:   30 * time.Second,
			ReadTimeout:      30 * time.Second,
			WriteTimeout:     30 * time.Second,
			IdleTimeout:      90 * time.Second,
			KeepAliveTimeout: 30 * time.Second,
		}
	}
	return p.timeoutConfig
}

// PerformHealthCheck performs a health check on a connection
func (p *ConnectionPool) PerformHealthCheck(host string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	client := p.GetConnection(host)
	healthCheck, exists := p.healthChecks[host]
	if !exists {
		return fmt.Errorf("no health check found for host: %s", host)
	}

	start := time.Now()

	// Perform a simple health check request
	// For MVP, we'll use HTTP for health checks to avoid HTTPS issues in test environments
	req, err := http.NewRequest("GET", fmt.Sprintf("http://%s/health", host), nil)
	if err != nil {
		healthCheck.LastError = err
		healthCheck.ErrorCount++
		healthCheck.IsHealthy = false
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	responseTime := time.Since(start)

	if err != nil {
		healthCheck.LastError = err
		healthCheck.ErrorCount++
		healthCheck.IsHealthy = false
		healthCheck.ResponseTime = responseTime
		return err
	}
	defer resp.Body.Close()

	healthCheck.LastCheck = time.Now()
	healthCheck.ResponseTime = responseTime
	healthCheck.SuccessCount++
	healthCheck.IsHealthy = resp.StatusCode >= 200 && resp.StatusCode < 300
	healthCheck.LastError = nil

	return nil
}

// GetHealthyConnection returns a healthy connection using load balancing
func (p *ConnectionPool) GetHealthyConnection(hosts []string) (*http.Client, error) {
	if p.loadBalancer == nil {
		p.loadBalancer = &LoadBalancer{
			connections:  hosts,
			currentIndex: 0,
			strategy:     StrategyHealthCheck,
		}
	}

	switch p.loadBalancer.strategy {
	case StrategyRoundRobin:
		return p.getConnectionRoundRobin(hosts)
	case StrategyHealthCheck:
		return p.getConnectionHealthCheck(hosts)
	case StrategyLeastConnections:
		return p.getConnectionLeastConnections(hosts)
	default:
		return p.getConnectionRoundRobin(hosts)
	}
}

// getConnectionRoundRobin implements round-robin load balancing
func (p *ConnectionPool) getConnectionRoundRobin(hosts []string) (*http.Client, error) {
	if len(hosts) == 0 {
		return nil, fmt.Errorf("no hosts available")
	}

	p.loadBalancer.mu.Lock()
	defer p.loadBalancer.mu.Unlock()

	host := hosts[p.loadBalancer.currentIndex%len(hosts)]
	p.loadBalancer.currentIndex++

	return p.GetConnection(host), nil
}

// getConnectionHealthCheck returns the healthiest connection
func (p *ConnectionPool) getConnectionHealthCheck(hosts []string) (*http.Client, error) {
	var healthiestHost string
	var bestResponseTime time.Duration = time.Hour // Start with a very high value

	for _, host := range hosts {
		healthCheck, exists := p.healthChecks[host]
		if !exists || !healthCheck.IsHealthy {
			continue
		}

		if healthCheck.ResponseTime < bestResponseTime {
			bestResponseTime = healthCheck.ResponseTime
			healthiestHost = host
		}
	}

	if healthiestHost == "" {
		return nil, fmt.Errorf("no healthy hosts available")
	}

	return p.GetConnection(healthiestHost), nil
}

// getConnectionLeastConnections returns connection with least active connections
func (p *ConnectionPool) getConnectionLeastConnections(hosts []string) (*http.Client, error) {
	// For simplicity, we'll use round-robin as a fallback
	// In a real implementation, you'd track active connection counts
	return p.getConnectionRoundRobin(hosts)
}

// GetConnectionHealthStats returns health statistics for all connections
func (p *ConnectionPool) GetConnectionHealthStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := make(map[string]interface{})
	for host, healthCheck := range p.healthChecks {
		stats[host] = map[string]interface{}{
			"is_healthy":    healthCheck.IsHealthy,
			"last_check":    healthCheck.LastCheck,
			"response_time": healthCheck.ResponseTime,
			"error_count":   healthCheck.ErrorCount,
			"success_count": healthCheck.SuccessCount,
			"last_error":    healthCheck.LastError,
		}
	}
	return stats
}

// CanExecute checks if the circuit breaker allows execution
func (cb *CircuitBreaker) CanExecute() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// Check if timeout has passed
		if time.Since(cb.lastFailureTime) > cb.timeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.state = CircuitHalf
			cb.mu.Unlock()
			cb.mu.RLock()
			return true
		}
		return false
	case CircuitHalf:
		return true
	default:
		return false
	}
}

// RecordFailure records a failure in the circuit breaker
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount++
	cb.lastFailureTime = time.Now()

	if cb.failureCount >= cb.threshold {
		cb.state = CircuitOpen
	}
}

// RecordSuccess records a success in the circuit breaker
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failureCount = 0
	cb.state = CircuitClosed
}

// GetCircuitBreakerStats returns circuit breaker statistics
func (cb *CircuitBreaker) GetCircuitBreakerStats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return map[string]interface{}{
		"state":         cb.state,
		"failure_count": cb.failureCount,
		"threshold":     cb.threshold,
		"timeout":       cb.timeout.String(),
		"last_failure":  cb.lastFailureTime.Format(time.RFC3339),
	}
}

// GetConnectionPoolStats returns connection pool statistics
func (p *ConnectionPool) GetConnectionPoolStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Count healthy connections
	healthyCount := 0
	for _, healthCheck := range p.healthChecks {
		if healthCheck.IsHealthy {
			healthyCount++
		}
	}

	return map[string]interface{}{
		"total_connections":   len(p.clients),
		"healthy_connections": healthyCount,
		"max_idle":            p.maxIdle,
		"idle_timeout":        p.idleTime.String(),
		"health_stats":        p.GetConnectionHealthStats(),
	}
}

// GetRetryStats returns retry configuration statistics
func (s *DPConnectorService) GetRetryStats() map[string]interface{} {
	return map[string]interface{}{
		"max_retries":        s.retryConfig.MaxRetries,
		"base_delay":         s.retryConfig.BaseDelay.String(),
		"max_delay":          s.retryConfig.MaxDelay.String(),
		"backoff_multiplier": s.retryConfig.BackoffMultiplier,
	}
}

// GetDPStats returns comprehensive DP connector statistics
func (s *DPConnectorService) GetDPStats() map[string]interface{} {
	stats := map[string]interface{}{
		"service_status":   "active",
		"dp_connector_url": s.config.DPConnectorURL,
		"timeout":          s.client.Timeout.String(),
	}

	// Add circuit breaker stats
	stats["circuit_breaker"] = s.circuitBreaker.GetCircuitBreakerStats()

	// Add connection pool stats
	stats["connection_pool"] = s.pool.GetConnectionPoolStats()

	// Add retry stats
	stats["retry_stats"] = s.GetRetryStats()

	return stats
}

// HealthCheck checks if the DP connector service is healthy
func (s *DPConnectorService) HealthCheck(ctx context.Context) error {
	// Check circuit breaker state
	if !s.circuitBreaker.CanExecute() {
		return fmt.Errorf("circuit breaker is open")
	}

	// Test connection to DP connector
	req, err := http.NewRequestWithContext(ctx, "GET", s.config.DPConnectorURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("DP connector health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("DP connector health check returned status %d", resp.StatusCode)
	}

	return nil
}

// AuthenticationConfig defines authentication settings
type AuthenticationConfig struct {
	APIKey     string
	OAuth2     *OAuth2Config
	MTLS       *MTLSConfig
	JWT        *JWTConfig
	AuthMethod AuthMethod
}

// AuthMethod defines the authentication method
type AuthMethod string

const (
	AuthMethodAPIKey AuthMethod = "api_key"
	AuthMethodOAuth2 AuthMethod = "oauth2"
	AuthMethodMTLS   AuthMethod = "mtls"
	AuthMethodJWT    AuthMethod = "jwt"
	AuthMethodNone   AuthMethod = "none"
)

// OAuth2Config defines OAuth 2.0 configuration
type OAuth2Config struct {
	ClientID     string
	ClientSecret string
	TokenURL     string
	Scopes       []string
	RedirectURL  string
}

// MTLSConfig defines mTLS configuration
type MTLSConfig struct {
	CertFile string
	KeyFile  string
	CAFile   string
}

// JWTConfig defines JWT configuration
type JWTConfig struct {
	Secret     string
	Issuer     string
	Audience   string
	Expiration time.Duration
}

// Authenticator handles authentication for DP connections
type Authenticator struct {
	config *AuthenticationConfig
	client *http.Client
}

// NewAuthenticator creates a new authenticator
func NewAuthenticator(config *AuthenticationConfig) *Authenticator {
	return &Authenticator{
		config: config,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// AuthenticateRequest adds authentication headers to the request
func (a *Authenticator) AuthenticateRequest(req *http.Request) error {
	switch a.config.AuthMethod {
	case AuthMethodAPIKey:
		return a.addAPIKeyAuth(req)
	case AuthMethodOAuth2:
		return a.addOAuth2Auth(req)
	case AuthMethodMTLS:
		return a.addMTLSAuth(req)
	case AuthMethodJWT:
		return a.addJWTAuth(req)
	case AuthMethodNone:
		return nil
	default:
		return fmt.Errorf("unsupported authentication method: %s", a.config.AuthMethod)
	}
}

// addAPIKeyAuth adds API key authentication
func (a *Authenticator) addAPIKeyAuth(req *http.Request) error {
	if a.config.APIKey == "" {
		return fmt.Errorf("API key not configured")
	}

	req.Header.Set("Authorization", "Bearer "+a.config.APIKey)
	req.Header.Set("X-API-Key", a.config.APIKey)
	return nil
}

// addOAuth2Auth adds OAuth 2.0 authentication
func (a *Authenticator) addOAuth2Auth(req *http.Request) error {
	if a.config.OAuth2 == nil {
		return fmt.Errorf("OAuth2 configuration not provided")
	}

	// Get OAuth2 token (simplified implementation)
	token, err := a.getOAuth2Token()
	if err != nil {
		return fmt.Errorf("failed to get OAuth2 token: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}

// addMTLSAuth adds mTLS authentication
func (a *Authenticator) addMTLSAuth(req *http.Request) error {
	if a.config.MTLS == nil {
		return fmt.Errorf("mTLS configuration not provided")
	}

	// mTLS is handled at the transport level
	// This method is for additional mTLS-specific headers if needed
	req.Header.Set("X-Client-Cert", "true")
	return nil
}

// addJWTAuth adds JWT authentication
func (a *Authenticator) addJWTAuth(req *http.Request) error {
	if a.config.JWT == nil {
		return fmt.Errorf("JWT configuration not provided")
	}

	// Generate JWT token
	token, err := a.generateJWTToken()
	if err != nil {
		return fmt.Errorf("failed to generate JWT token: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	return nil
}

// getOAuth2Token retrieves OAuth2 token (simplified implementation)
func (a *Authenticator) getOAuth2Token() (string, error) {
	// In a real implementation, this would:
	// 1. Check if we have a valid cached token
	// 2. If not, request a new token from the OAuth2 provider
	// 3. Cache the token for future use

	// For MVP, return a mock token
	return "mock_oauth2_token", nil
}

// generateJWTToken generates a JWT token
func (a *Authenticator) generateJWTToken() (string, error) {
	// In a real implementation, this would:
	// 1. Create JWT claims
	// 2. Sign the token with the configured secret
	// 3. Return the signed token

	// For MVP, return a mock token
	return "mock_jwt_token", nil
}

// ValidateToken validates an incoming token
func (a *Authenticator) ValidateToken(token string) error {
	switch a.config.AuthMethod {
	case AuthMethodJWT:
		return a.validateJWTToken(token)
	case AuthMethodAPIKey:
		return a.validateAPIKey(token)
	default:
		return fmt.Errorf("token validation not supported for method: %s", a.config.AuthMethod)
	}
}

// validateJWTToken validates a JWT token
func (a *Authenticator) validateJWTToken(token string) error {
	// In a real implementation, this would:
	// 1. Parse the JWT token
	// 2. Verify the signature
	// 3. Check claims (issuer, audience, expiration, etc.)

	// For MVP, simple validation
	if token == "" {
		return fmt.Errorf("empty JWT token")
	}

	if !strings.HasPrefix(token, "mock_jwt_token") {
		return fmt.Errorf("invalid JWT token format")
	}

	return nil
}

// validateAPIKey validates an API key
func (a *Authenticator) validateAPIKey(key string) error {
	if key == "" {
		return fmt.Errorf("empty API key")
	}

	if key != a.config.APIKey {
		return fmt.Errorf("invalid API key")
	}

	return nil
}

// TestAuthenticationFlow tests the authentication flow
func (a *Authenticator) TestAuthenticationFlow() error {
	// Create a test request
	req, err := http.NewRequest("GET", "https://test.example.com/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create test request: %w", err)
	}

	// Add authentication
	if err := a.AuthenticateRequest(req); err != nil {
		return fmt.Errorf("failed to authenticate request: %w", err)
	}

	// Verify authentication headers are present
	switch a.config.AuthMethod {
	case AuthMethodAPIKey:
		if req.Header.Get("Authorization") == "" && req.Header.Get("X-API-Key") == "" {
			return fmt.Errorf("API key authentication headers not set")
		}
	case AuthMethodOAuth2:
		if req.Header.Get("Authorization") == "" {
			return fmt.Errorf("OAuth2 authentication header not set")
		}
	case AuthMethodJWT:
		if req.Header.Get("Authorization") == "" {
			return fmt.Errorf("JWT authentication header not set")
		}
	case AuthMethodMTLS:
		if req.Header.Get("X-Client-Cert") == "" {
			return fmt.Errorf("mTLS authentication header not set")
		}
	}

	return nil
}

// IntegrationAdapter defines the interface for different integration adapters
type IntegrationAdapter interface {
	Connect(ctx context.Context, config *AdapterConfig) error
	SendRequest(ctx context.Context, request interface{}) (interface{}, error)
	Close() error
	GetAdapterType() string
}

// AdapterConfig defines configuration for adapters
type AdapterConfig struct {
	URL         string
	Timeout     time.Duration
	Headers     map[string]string
	AuthConfig  *AuthenticationConfig
	AdapterType AdapterType
}

// AdapterType defines the type of adapter
type AdapterType string

const (
	AdapterTypeREST      AdapterType = "rest"
	AdapterTypeGraphQL   AdapterType = "graphql"
	AdapterTypeGRPC      AdapterType = "grpc"
	AdapterTypeWebSocket AdapterType = "websocket"
)

// RESTAdapter implements REST API integration
type RESTAdapter struct {
	client  *http.Client
	config  *AdapterConfig
	baseURL string
}

// NewRESTAdapter creates a new REST adapter
func NewRESTAdapter() *RESTAdapter {
	return &RESTAdapter{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Connect establishes connection to REST API
func (r *RESTAdapter) Connect(ctx context.Context, config *AdapterConfig) error {
	r.config = config
	r.baseURL = config.URL

	// Test connection
	req, err := http.NewRequestWithContext(ctx, "GET", r.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	// Add authentication if configured
	if config.AuthConfig != nil {
		auth := NewAuthenticator(config.AuthConfig)
		if err := auth.AuthenticateRequest(req); err != nil {
			return fmt.Errorf("failed to authenticate request: %w", err)
		}
	}

	// Add custom headers
	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to REST API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("REST API health check failed with status: %d", resp.StatusCode)
	}

	return nil
}

// SendRequest sends a request to the REST API
func (r *RESTAdapter) SendRequest(ctx context.Context, request interface{}) (interface{}, error) {
	// Marshal request to JSON
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", r.baseURL+"/api/v1/verify", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add authentication if configured
	if r.config.AuthConfig != nil {
		auth := NewAuthenticator(r.config.AuthConfig)
		if err := auth.AuthenticateRequest(req); err != nil {
			return nil, fmt.Errorf("failed to authenticate request: %w", err)
		}
	}

	// Add custom headers
	for key, value := range r.config.Headers {
		req.Header.Set(key, value)
	}

	// Send request
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send REST request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode REST response: %w", err)
	}

	return response, nil
}

// Close closes the REST adapter
func (r *RESTAdapter) Close() error {
	// REST adapter doesn't need explicit cleanup
	return nil
}

// GetAdapterType returns the adapter type
func (r *RESTAdapter) GetAdapterType() string {
	return string(AdapterTypeREST)
}

// GraphQLAdapter implements GraphQL integration
type GraphQLAdapter struct {
	client *http.Client
	config *AdapterConfig
	url    string
}

// NewGraphQLAdapter creates a new GraphQL adapter
func NewGraphQLAdapter() *GraphQLAdapter {
	return &GraphQLAdapter{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Connect establishes connection to GraphQL API
func (g *GraphQLAdapter) Connect(ctx context.Context, config *AdapterConfig) error {
	g.config = config
	g.url = config.URL

	// Test connection with introspection query
	query := `{"query": "{ __schema { types { name } } }"}`
	req, err := http.NewRequestWithContext(ctx, "POST", g.url, strings.NewReader(query))
	if err != nil {
		return fmt.Errorf("failed to create GraphQL health check request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add authentication if configured
	if config.AuthConfig != nil {
		auth := NewAuthenticator(config.AuthConfig)
		if err := auth.AuthenticateRequest(req); err != nil {
			return fmt.Errorf("failed to authenticate request: %w", err)
		}
	}

	// Add custom headers
	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	resp, err := g.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to GraphQL API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GraphQL API health check failed with status: %d", resp.StatusCode)
	}

	return nil
}

// SendRequest sends a GraphQL query
func (g *GraphQLAdapter) SendRequest(ctx context.Context, request interface{}) (interface{}, error) {
	// Convert request to GraphQL query
	query, ok := request.(string)
	if !ok {
		return nil, fmt.Errorf("GraphQL adapter expects string query")
	}

	// Create GraphQL request
	graphqlReq := map[string]interface{}{
		"query": query,
	}

	requestBody, err := json.Marshal(graphqlReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GraphQL request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", g.url, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create GraphQL request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add authentication if configured
	if g.config.AuthConfig != nil {
		auth := NewAuthenticator(g.config.AuthConfig)
		if err := auth.AuthenticateRequest(req); err != nil {
			return nil, fmt.Errorf("failed to authenticate request: %w", err)
		}
	}

	// Add custom headers
	for key, value := range g.config.Headers {
		req.Header.Set(key, value)
	}

	// Send request
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send GraphQL request: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode GraphQL response: %w", err)
	}

	return response, nil
}

// Close closes the GraphQL adapter
func (g *GraphQLAdapter) Close() error {
	// GraphQL adapter doesn't need explicit cleanup
	return nil
}

// GetAdapterType returns the adapter type
func (g *GraphQLAdapter) GetAdapterType() string {
	return string(AdapterTypeGraphQL)
}

// GRPCAdapter implements gRPC integration (simplified for MVP)
type GRPCAdapter struct {
	config *AdapterConfig
	url    string
}

// NewGRPCAdapter creates a new gRPC adapter
func NewGRPCAdapter() *GRPCAdapter {
	return &GRPCAdapter{}
}

// Connect establishes connection to gRPC service
func (g *GRPCAdapter) Connect(ctx context.Context, config *AdapterConfig) error {
	g.config = config
	g.url = config.URL

	// For MVP, we'll simulate gRPC connection
	// In production, this would use the gRPC client library
	return nil
}

// SendRequest sends a gRPC request
func (g *GRPCAdapter) SendRequest(ctx context.Context, request interface{}) (interface{}, error) {
	// For MVP, simulate gRPC request
	// In production, this would use the gRPC client library

	// Simulate response
	response := map[string]interface{}{
		"grpc_response": "mock_grpc_response",
		"status":        "success",
		"timestamp":     time.Now().Format(time.RFC3339),
	}

	return response, nil
}

// Close closes the gRPC adapter
func (g *GRPCAdapter) Close() error {
	// gRPC adapter cleanup would be handled here
	return nil
}

// GetAdapterType returns the adapter type
func (g *GRPCAdapter) GetAdapterType() string {
	return string(AdapterTypeGRPC)
}

// WebSocketAdapter implements WebSocket integration
type WebSocketAdapter struct {
	conn   interface{} // For MVP, we'll use interface{} instead of websocket.Conn
	config *AdapterConfig
	url    string
}

// NewWebSocketAdapter creates a new WebSocket adapter
func NewWebSocketAdapter() *WebSocketAdapter {
	return &WebSocketAdapter{}
}

// Connect establishes WebSocket connection
func (w *WebSocketAdapter) Connect(ctx context.Context, config *AdapterConfig) error {
	w.config = config
	w.url = config.URL

	// For MVP, we'll simulate WebSocket connection
	// In production, this would use the WebSocket client library
	return nil
}

// SendRequest sends a WebSocket message
func (w *WebSocketAdapter) SendRequest(ctx context.Context, request interface{}) (interface{}, error) {
	// For MVP, simulate WebSocket request
	// In production, this would use the WebSocket client library

	// Simulate response
	response := map[string]interface{}{
		"websocket_response": "mock_websocket_response",
		"status":             "success",
		"timestamp":          time.Now().Format(time.RFC3339),
	}

	return response, nil
}

// Close closes the WebSocket adapter
func (w *WebSocketAdapter) Close() error {
	// For MVP, WebSocket cleanup is simplified
	// In production, this would properly close the WebSocket connection
	return nil
}

// GetAdapterType returns the adapter type
func (w *WebSocketAdapter) GetAdapterType() string {
	return string(AdapterTypeWebSocket)
}

// AdapterFactory creates adapters based on type
func AdapterFactory(adapterType AdapterType) IntegrationAdapter {
	switch adapterType {
	case AdapterTypeREST:
		return NewRESTAdapter()
	case AdapterTypeGraphQL:
		return NewGraphQLAdapter()
	case AdapterTypeGRPC:
		return NewGRPCAdapter()
	case AdapterTypeWebSocket:
		return NewWebSocketAdapter()
	default:
		return NewRESTAdapter() // Default to REST
	}
}

// TestAdapterFunctionality tests adapter functionality
func TestAdapterFunctionality(adapter IntegrationAdapter, config *AdapterConfig) error {
	ctx := context.Background()

	// Test connection
	if err := adapter.Connect(ctx, config); err != nil {
		return fmt.Errorf("failed to connect adapter: %w", err)
	}

	// Test request
	testRequest := map[string]interface{}{
		"test": "data",
	}

	_, err := adapter.SendRequest(ctx, testRequest)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	// Test close
	if err := adapter.Close(); err != nil {
		return fmt.Errorf("failed to close adapter: %w", err)
	}

	return nil
}
