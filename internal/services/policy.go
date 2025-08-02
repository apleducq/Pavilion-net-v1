package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/models"
)

// PolicyService handles policy enforcement using OPA
type PolicyService struct {
	config *config.Config
	client *http.Client
	cache  *PolicyCache
}

// PolicyCache provides caching for policy decisions
type PolicyCache struct {
	decisions map[string]*models.PolicyDecision
	mu        sync.RWMutex
	ttl       time.Duration
}

// NewPolicyCache creates a new policy cache
func NewPolicyCache(ttl time.Duration) *PolicyCache {
	return &PolicyCache{
		decisions: make(map[string]*models.PolicyDecision),
		ttl:       ttl,
	}
}

// Get retrieves a cached policy decision
func (c *PolicyCache) Get(key string) (*models.PolicyDecision, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	decision, exists := c.decisions[key]
	if !exists {
		return nil, false
	}

	// Check if decision has expired
	timestamp, err := time.Parse(time.RFC3339, decision.Timestamp)
	if err != nil {
		// If we can't parse the timestamp, consider it expired
		delete(c.decisions, key)
		return nil, false
	}

	if time.Since(timestamp) > c.ttl {
		delete(c.decisions, key)
		return nil, false
	}

	return decision, true
}

// Set stores a policy decision in cache
func (c *PolicyCache) Set(key string, decision *models.PolicyDecision) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.decisions[key] = decision
}

// NewPolicyService creates a new policy service
func NewPolicyService(cfg *config.Config) *PolicyService {
	return &PolicyService{
		config: cfg,
		client: &http.Client{
			Timeout: cfg.OPATimeout,
		},
		cache: NewPolicyCache(5 * time.Minute), // Cache policy decisions for 5 minutes
	}
}

// EnforcePolicy checks if the request is allowed based on policies
func (s *PolicyService) EnforcePolicy(ctx context.Context, req models.VerificationRequest) error {
	// Generate cache key based on request parameters
	cacheKey := s.generateCacheKey(req)

	// Check cache first
	if cached, exists := s.cache.Get(cacheKey); exists {
		if !cached.Allowed {
			return fmt.Errorf("policy violation (cached): %s", cached.Reason)
		}
		return nil
	}

	// Create policy query
	query := map[string]interface{}{
		"input": map[string]interface{}{
			"rp_id":      req.RPID,
			"claim_type": req.ClaimType,
			"user_id":    req.UserID,
			"timestamp":  time.Now().Format(time.RFC3339),
		},
	}

	// Send query to OPA
	decision, err := s.queryOPA(ctx, query)
	if err != nil {
		// Log the error but don't cache failures
		return fmt.Errorf("policy query failed: %w", err)
	}

	// Cache the decision
	s.cache.Set(cacheKey, decision)

	// Check if request is allowed
	if !decision.Allowed {
		return fmt.Errorf("policy violation: %s", decision.Reason)
	}

	return nil
}

// queryOPA sends a query to the OPA service
func (s *PolicyService) queryOPA(ctx context.Context, query map[string]interface{}) (*models.PolicyDecision, error) {
	// Prepare the request body
	requestBody, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal policy query: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", s.config.OPAURL+"/v1/data/pavilion/allow", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create OPA request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Send request to OPA
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OPA request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OPA service returned status %d", resp.StatusCode)
	}

	// Parse OPA response
	var opaResponse struct {
		Result bool `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&opaResponse); err != nil {
		return nil, fmt.Errorf("failed to parse OPA response: %w", err)
	}

	// Create policy decision
	decision := &models.PolicyDecision{
		Allowed:   opaResponse.Result,
		Reason:    s.getPolicyReason(opaResponse.Result),
		PolicyID:  "opa-policy-001",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	return decision, nil
}

// generateCacheKey creates a cache key for the request
func (s *PolicyService) generateCacheKey(req models.VerificationRequest) string {
	return fmt.Sprintf("%s:%s:%s", req.RPID, req.ClaimType, req.UserID)
}

// getPolicyReason returns a human-readable reason for the policy decision
func (s *PolicyService) getPolicyReason(allowed bool) string {
	if allowed {
		return "Request allowed by policy"
	}
	return "Request denied by policy"
}

// HealthCheck checks if the policy service is healthy
func (s *PolicyService) HealthCheck(ctx context.Context) error {
	// Create a simple health check query
	query := map[string]interface{}{
		"input": map[string]interface{}{
			"health_check": true,
		},
	}

	// Try to query OPA
	_, err := s.queryOPA(ctx, query)
	if err != nil {
		return fmt.Errorf("OPA health check failed: %w", err)
	}

	return nil
}

// GetCacheStats returns cache statistics for monitoring
func (s *PolicyService) GetCacheStats() map[string]interface{} {
	s.cache.mu.RLock()
	defer s.cache.mu.RUnlock()

	return map[string]interface{}{
		"cached_decisions": len(s.cache.decisions),
		"cache_ttl":        s.cache.ttl.String(),
	}
}
