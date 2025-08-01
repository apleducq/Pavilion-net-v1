package services

import (
	"context"
	"testing"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/models"
)

func TestNewCacheService(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
			TTL:      3600,
		},
	}
	
	service := NewCacheService(cfg)
	
	if service == nil {
		t.Fatal("Expected service to be created")
	}
	
	if service.config != cfg {
		t.Error("Expected config to be set")
	}
	
	if service.client == nil {
		t.Error("Expected Redis client to be created")
	}
	
	// Clean up
	defer service.Close()
}

func TestGenerateCacheKey(t *testing.T) {
	service := NewCacheService(&config.Config{})
	
	req := models.VerificationRequest{
		RPID:      "rp-001",
		UserID:    "user-123",
		ClaimType: "student_verification",
	}
	
	key := service.generateCacheKey(req)
	expectedKey := "verification:rp-001:user-123:student_verification"
	
	if key != expectedKey {
		t.Errorf("Expected key %s, got %s", expectedKey, key)
	}
}

func TestGetVerificationResult(t *testing.T) {
	// Note: This test requires a running Redis instance
	// In a real environment, you'd use a test Redis instance or mock
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
			TTL:      3600,
		},
	}
	
	service := NewCacheService(cfg)
	defer service.Close()
	
	req := models.VerificationRequest{
		RPID:      "rp-001",
		UserID:    "user-123",
		ClaimType: "student_verification",
	}
	
	// Test cache miss (no entry exists)
	result := service.GetVerificationResult(req)
	if result != nil {
		t.Error("Expected nil result for cache miss")
	}
}

func TestCacheVerificationResult(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
			TTL:      3600,
		},
	}
	
	service := NewCacheService(cfg)
	defer service.Close()
	
	req := models.VerificationRequest{
		RPID:      "rp-001",
		UserID:    "user-123",
		ClaimType: "student_verification",
	}
	
	response := &models.VerificationResponse{
		VerificationID:  "verif-456",
		Status:          "verified",
		Verified:        true,
		ConfidenceScore: 0.95,
		DPID:            "dp-001",
		Timestamp:       time.Now().Format(time.RFC3339),
		ExpiresAt:       time.Now().Add(90 * 24 * time.Hour).Format(time.RFC3339),
		RequestID:       "req-123",
	}
	
	// Test caching
	service.CacheVerificationResult(req, response)
	
	// Note: In a real test environment, you'd verify the cache was set
	// by checking Redis directly or using a test Redis instance
}

func TestGetCacheKey(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
			TTL:      3600,
		},
	}
	
	service := NewCacheService(cfg)
	defer service.Close()
	
	// Test getting non-existent key
	value, err := service.GetCacheKey("non-existent-key")
	if err == nil {
		t.Error("Expected error for non-existent key")
	}
	
	if value != "" {
		t.Error("Expected empty value for non-existent key")
	}
}

func TestSetCacheKey(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
			TTL:      3600,
		},
	}
	
	service := NewCacheService(cfg)
	defer service.Close()
	
	// Test setting cache key
	err := service.SetCacheKey("test-key", "test-value", time.Minute)
	if err != nil {
		t.Errorf("Expected no error setting cache key: %v", err)
	}
	
	// Test getting the key back
	value, err := service.GetCacheKey("test-key")
	if err != nil {
		t.Errorf("Expected no error getting cache key: %v", err)
	}
	
	if value != "test-value" {
		t.Errorf("Expected value 'test-value', got '%s'", value)
	}
}

func TestDeleteCacheKey(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
			TTL:      3600,
		},
	}
	
	service := NewCacheService(cfg)
	defer service.Close()
	
	// Set a key first
	err := service.SetCacheKey("delete-test-key", "delete-test-value", time.Minute)
	if err != nil {
		t.Fatalf("Failed to set cache key: %v", err)
	}
	
	// Delete the key
	err = service.DeleteCacheKey("delete-test-key")
	if err != nil {
		t.Errorf("Expected no error deleting cache key: %v", err)
	}
	
	// Verify key is deleted
	value, err := service.GetCacheKey("delete-test-key")
	if err == nil {
		t.Error("Expected error for deleted key")
	}
	
	if value != "" {
		t.Error("Expected empty value for deleted key")
	}
}

func TestGetCacheTTL(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
			TTL:      3600,
		},
	}
	
	service := NewCacheService(cfg)
	defer service.Close()
	
	// Set a key with TTL
	err := service.SetCacheKey("ttl-test-key", "ttl-test-value", time.Minute)
	if err != nil {
		t.Fatalf("Failed to set cache key: %v", err)
	}
	
	// Get TTL
	ttl, err := service.GetCacheTTL("ttl-test-key")
	if err != nil {
		t.Errorf("Expected no error getting TTL: %v", err)
	}
	
	if ttl <= 0 {
		t.Error("Expected positive TTL")
	}
	
	// Test TTL for non-existent key
	ttl, err = service.GetCacheTTL("non-existent-key")
	if err == nil {
		t.Error("Expected error for non-existent key")
	}
}

func TestFlushCache(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
			TTL:      3600,
		},
	}
	
	service := NewCacheService(cfg)
	defer service.Close()
	
	// Set some test keys
	err := service.SetCacheKey("flush-test-1", "value1", time.Minute)
	if err != nil {
		t.Fatalf("Failed to set cache key: %v", err)
	}
	
	err = service.SetCacheKey("flush-test-2", "value2", time.Minute)
	if err != nil {
		t.Fatalf("Failed to set cache key: %v", err)
	}
	
	// Flush cache
	err = service.FlushCache()
	if err != nil {
		t.Errorf("Expected no error flushing cache: %v", err)
	}
	
	// Verify keys are gone
	value1, _ := service.GetCacheKey("flush-test-1")
	value2, _ := service.GetCacheKey("flush-test-2")
	
	if value1 != "" || value2 != "" {
		t.Error("Expected keys to be deleted after flush")
	}
}

func TestGetCacheStats(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
			TTL:      3600,
		},
	}
	
	service := NewCacheService(cfg)
	defer service.Close()
	
	// Get cache stats
	stats, err := service.GetCacheStats()
	if err != nil {
		t.Errorf("Expected no error getting cache stats: %v", err)
	}
	
	if stats == nil {
		t.Fatal("Expected stats to be returned")
	}
	
	// Check for expected fields
	if stats["info"] == nil {
		t.Error("Expected info field in stats")
	}
	
	if stats["dbsize"] == nil {
		t.Error("Expected dbsize field in stats")
	}
}

func TestHealthCheck(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
			TTL:      3600,
		},
	}
	
	service := NewCacheService(cfg)
	defer service.Close()
	
	// Test health check
	err := service.HealthCheck(context.Background())
	if err != nil {
		t.Errorf("Expected health check to pass: %v", err)
	}
}

func TestCacheWithExpiredResponse(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
			TTL:      3600,
		},
	}
	
	service := NewCacheService(cfg)
	defer service.Close()
	
	req := models.VerificationRequest{
		RPID:      "rp-001",
		UserID:    "user-123",
		ClaimType: "student_verification",
	}
	
	// Create response with expired timestamp
	response := &models.VerificationResponse{
		VerificationID:  "verif-456",
		Status:          "verified",
		Verified:        true,
		ConfidenceScore: 0.95,
		DPID:            "dp-001",
		Timestamp:       time.Now().Format(time.RFC3339),
		ExpiresAt:       time.Now().Add(-1 * time.Hour).Format(time.RFC3339), // Expired
		RequestID:       "req-123",
	}
	
	// Cache the expired response
	service.CacheVerificationResult(req, response)
	
	// Try to get the result - should return nil due to expiration
	result := service.GetVerificationResult(req)
	if result != nil {
		t.Error("Expected nil result for expired response")
	}
}

func TestCacheSerialization(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
			TTL:      3600,
		},
	}
	
	service := NewCacheService(cfg)
	defer service.Close()
	
	req := models.VerificationRequest{
		RPID:      "rp-001",
		UserID:    "user-123",
		ClaimType: "student_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
		Metadata: map[string]interface{}{
			"source": "web",
		},
	}
	
	response := &models.VerificationResponse{
		VerificationID:  "verif-456",
		Status:          "verified",
		Verified:        true,
		ConfidenceScore: 0.95,
		DPID:            "dp-001",
		Timestamp:       time.Now().Format(time.RFC3339),
		ExpiresAt:       time.Now().Add(90 * 24 * time.Hour).Format(time.RFC3339),
		RequestID:       "req-123",
		ProcessingTime:  "150ms",
		Metadata: map[string]interface{}{
			"cache_hit": true,
		},
	}
	
	// Test caching with complex response
	service.CacheVerificationResult(req, response)
	
	// Note: In a real test environment, you'd verify the cache was set correctly
	// by retrieving it and comparing the values
}

func TestCacheErrorHandling(t *testing.T) {
	// Test with invalid Redis configuration
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host:     "invalid-host",
			Port:     9999,
			Password: "",
			DB:       0,
			TTL:      3600,
		},
	}
	
	service := NewCacheService(cfg)
	defer service.Close()
	
	req := models.VerificationRequest{
		RPID:      "rp-001",
		UserID:    "user-123",
		ClaimType: "student_verification",
	}
	
	response := &models.VerificationResponse{
		VerificationID:  "verif-456",
		Status:          "verified",
		Verified:        true,
		ConfidenceScore: 0.95,
		DPID:            "dp-001",
		Timestamp:       time.Now().Format(time.RFC3339),
		ExpiresAt:       time.Now().Add(90 * 24 * time.Hour).Format(time.RFC3339),
		RequestID:       "req-123",
	}
	
	// These operations should handle connection errors gracefully
	service.CacheVerificationResult(req, response)
	result := service.GetVerificationResult(req)
	
	// Should return nil for cache miss due to connection error
	if result != nil {
		t.Error("Expected nil result due to connection error")
	}
}

func TestCacheKeyGeneration(t *testing.T) {
	service := NewCacheService(&config.Config{})
	
	// Test different request combinations
	testCases := []struct {
		req      models.VerificationRequest
		expected string
	}{
		{
			req: models.VerificationRequest{
				RPID:      "rp-001",
				UserID:    "user-123",
				ClaimType: "student_verification",
			},
			expected: "verification:rp-001:user-123:student_verification",
		},
		{
			req: models.VerificationRequest{
				RPID:      "rp-002",
				UserID:    "user-456",
				ClaimType: "employee_verification",
			},
			expected: "verification:rp-002:user-456:employee_verification",
		},
		{
			req: models.VerificationRequest{
				RPID:      "rp-003",
				UserID:    "user-789",
				ClaimType: "age_verification",
			},
			expected: "verification:rp-003:user-789:age_verification",
		},
	}
	
	for i, tc := range testCases {
		key := service.generateCacheKey(tc.req)
		if key != tc.expected {
			t.Errorf("Test case %d: Expected key %s, got %s", i+1, tc.expected, key)
		}
	}
} 