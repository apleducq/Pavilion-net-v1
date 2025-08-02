package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/models"
)

func TestNewPolicyService(t *testing.T) {
	cfg := &config.Config{
		OPAURL:     "http://opa:8181",
		OPATimeout: 5 * time.Second,
	}

	service := NewPolicyService(cfg)
	if service == nil {
		t.Fatal("Expected policy service to be created")
	}

	if service.config != cfg {
		t.Error("Expected config to be set")
	}

	if service.client == nil {
		t.Error("Expected HTTP client to be created")
	}

	if service.cache == nil {
		t.Error("Expected cache to be created")
	}
}

func TestPolicyCache(t *testing.T) {
	cache := NewPolicyCache(1 * time.Minute)

	// Test setting and getting a decision
	decision := &models.PolicyDecision{
		Allowed:   true,
		Reason:    "Test decision",
		PolicyID:  "test-policy",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	cache.Set("test-key", decision)

	// Test getting cached decision
	cached, exists := cache.Get("test-key")
	if !exists {
		t.Error("Expected cached decision to exist")
	}

	if cached.Allowed != decision.Allowed {
		t.Error("Expected cached decision to match original")
	}

	// Test getting non-existent key
	_, exists = cache.Get("non-existent")
	if exists {
		t.Error("Expected non-existent key to return false")
	}
}

func TestPolicyService_GenerateCacheKey(t *testing.T) {
	service := &PolicyService{}

	req := models.VerificationRequest{
		RPID:      "rp-001",
		UserID:    "user-001",
		ClaimType: "student_verification",
	}

	key := service.generateCacheKey(req)
	expected := "rp-001:student_verification:user-001"

	if key != expected {
		t.Errorf("Expected cache key %s, got %s", expected, key)
	}
}

func TestGetPolicyReason(t *testing.T) {
	service := &PolicyService{}

	// Test allowed reason
	reason := service.getPolicyReason(true)
	if reason != "Request allowed by policy" {
		t.Errorf("Expected 'Request allowed by policy', got %s", reason)
	}

	// Test denied reason
	reason = service.getPolicyReason(false)
	if reason != "Request denied by policy" {
		t.Errorf("Expected 'Request denied by policy', got %s", reason)
	}
}

func TestPolicyService_GetCacheStats(t *testing.T) {
	service := &PolicyService{
		cache: NewPolicyCache(5 * time.Minute),
	}

	// Add some test decisions
	decision := &models.PolicyDecision{
		Allowed:   true,
		Reason:    "Test",
		PolicyID:  "test",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	service.cache.Set("key1", decision)
	service.cache.Set("key2", decision)

	stats := service.GetCacheStats()

	if stats["cached_decisions"] != 2 {
		t.Errorf("Expected 2 cached decisions, got %v", stats["cached_decisions"])
	}

	if stats["cache_ttl"] != "5m0s" {
		t.Errorf("Expected cache TTL '5m0s', got %v", stats["cache_ttl"])
	}
}

func TestEnforcePolicy_WithCache(t *testing.T) {
	service := &PolicyService{
		cache: NewPolicyCache(5 * time.Minute),
	}

	req := models.VerificationRequest{
		RPID:      "rp-001",
		UserID:    "user-001",
		ClaimType: "student_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}

	// Add a cached decision
	cachedDecision := &models.PolicyDecision{
		Allowed:   true,
		Reason:    "Cached decision",
		PolicyID:  "cached-policy",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	cacheKey := service.generateCacheKey(req)
	service.cache.Set(cacheKey, cachedDecision)

	// Test that cached decision is used
	err := service.EnforcePolicy(context.Background(), req)
	if err != nil {
		t.Errorf("Expected no error for cached allowed decision, got %v", err)
	}

	// Test cached denied decision
	deniedDecision := &models.PolicyDecision{
		Allowed:   false,
		Reason:    "Cached denied decision",
		PolicyID:  "cached-policy",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	service.cache.Set(cacheKey, deniedDecision)

	err = service.EnforcePolicy(context.Background(), req)
	if err == nil {
		t.Error("Expected error for cached denied decision")
	}

	if err.Error() != "policy violation (cached): Cached denied decision" {
		t.Errorf("Expected specific error message, got %v", err)
	}
}

func TestEnforcePolicy_WithoutOPA(t *testing.T) {
	// Test with invalid OPA URL to simulate OPA service failure
	cfg := &config.Config{
		OPAURL:     "http://invalid-opa-url:8181",
		OPATimeout: 1 * time.Second,
	}

	service := NewPolicyService(cfg)

	req := models.VerificationRequest{
		RPID:      "rp-001",
		UserID:    "user-001",
		ClaimType: "student_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}

	// Test that OPA failure is handled gracefully
	err := service.EnforcePolicy(context.Background(), req)
	if err == nil {
		t.Error("Expected error when OPA service is unavailable")
	}

	// Check that the error contains "policy query failed"
	if !strings.Contains(err.Error(), "policy query failed") {
		t.Errorf("Expected error to contain 'policy query failed', got %v", err)
	}
}
