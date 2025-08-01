package services

import (
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
}

// ConnectionPool manages HTTP connections
type ConnectionPool struct {
	mu       sync.RWMutex
	clients  map[string]*http.Client
	maxIdle  int
	idleTime time.Duration
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
	MaxRetries      int
	BaseDelay       time.Duration
	MaxDelay        time.Duration
	BackoffMultiplier float64
}

// DPResponse represents a response from the DP Connector
type DPResponse struct {
	JobID           string                 `json:"job_id"`
	Status          string                 `json:"status"`
	VerificationResult *VerificationResult `json:"verification_result,omitempty"`
	Error           string                 `json:"error,omitempty"`
	Timestamp       string                 `json:"timestamp"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// VerificationResult represents the result of a verification
type VerificationResult struct {
	Verified    bool    `json:"verified"`
	Confidence  float64 `json:"confidence"`
	Reason      string  `json:"reason,omitempty"`
	Evidence    []string `json:"evidence,omitempty"`
	Timestamp   string  `json:"timestamp"`
}

// NewDPConnectorService creates a new DP connector service
func NewDPConnectorService(cfg *config.Config) *DPConnectorService {
	// Create retry configuration
	retryConfig := &RetryConfig{
		MaxRetries:       3,
		BaseDelay:        1 * time.Second,
		MaxDelay:         30 * time.Second,
		BackoffMultiplier: 2.0,
	}

	// Create connection pool
	pool := &ConnectionPool{
		clients:  make(map[string]*http.Client),
		maxIdle:  100,
		idleTime: 90 * time.Second,
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

	return &DPConnectorService{
		config:        cfg,
		client:        client,
		pool:          pool,
		circuitBreaker: circuitBreaker,
		retryConfig:   retryConfig,
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
	httpReq.Header.Set("Authorization", "Bearer "+s.config.DPConnectorToken)

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

	// Create new client for this host
	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        p.maxIdle,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     p.idleTime,
		},
	}

	p.clients[host] = client
	return client
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
		"state":           cb.state,
		"failure_count":   cb.failureCount,
		"threshold":       cb.threshold,
		"timeout":         cb.timeout.String(),
		"last_failure":    cb.lastFailureTime.Format(time.RFC3339),
	}
}

// GetConnectionPoolStats returns connection pool statistics
func (p *ConnectionPool) GetConnectionPoolStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]interface{}{
		"total_connections": len(p.clients),
		"max_idle":         p.maxIdle,
		"idle_timeout":     p.idleTime.String(),
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
		"service_status": "active",
		"dp_connector_url": s.config.DPConnectorURL,
		"timeout":          s.client.Timeout.String(),
	}

	// Add circuit breaker stats
	for key, value := range s.circuitBreaker.GetCircuitBreakerStats() {
		stats["circuit_breaker_"+key] = value
	}

	// Add connection pool stats
	for key, value := range s.pool.GetConnectionPoolStats() {
		stats["pool_"+key] = value
	}

	// Add retry stats
	for key, value := range s.GetRetryStats() {
		stats["retry_"+key] = value
	}

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