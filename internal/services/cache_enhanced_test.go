package services

import (
	"context"
	"testing"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/models"
)

func TestCacheService_EnhancedVerificationCaching(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
	}
	
	service := NewCacheService(cfg)
	
	// Test data
	req := models.VerificationRequest{
		UserID:    "user123",
		RPID:      "rp001",
		ClaimType: "student_verification",
		Identifiers: map[string]string{
			"student_id": "STU123456",
		},
	}
	
	response := &models.VerificationResponse{
		VerificationID: "verif_123",
		Verified:       true,
		ConfidenceScore: 0.95,
		Status:         "completed",
		DPID:           "dp-001",
		ExpiresAt:      time.Now().Add(90 * 24 * time.Hour).Format(time.RFC3339),
	}
	
	// Test caching with 90-day TTL
	t.Run("CacheVerificationResult_90DayTTL", func(t *testing.T) {
		service.CacheVerificationResult(req, response)
		
		// Verify cache hit
		cached := service.GetVerificationResult(req)
		if cached == nil {
			t.Error("Expected cached response, got nil")
		}
		
		if cached.VerificationID != response.VerificationID {
			t.Errorf("Expected verification ID %s, got %s", response.VerificationID, cached.VerificationID)
		}
	})
	
	// Test cache metrics
	t.Run("CacheMetrics", func(t *testing.T) {
		metrics := service.GetCacheMetrics()
		
		if metrics["hit_count"] == nil {
			t.Error("Expected hit count in metrics")
		}
		
		if metrics["miss_count"] == nil {
			t.Error("Expected miss count in metrics")
		}
		
		if metrics["hit_rate"] == nil {
			t.Error("Expected hit rate in metrics")
		}
	})
	
	// Test cache invalidation
	t.Run("InvalidateVerificationCache", func(t *testing.T) {
		err := service.InvalidateVerificationCache(req)
		if err != nil {
			t.Errorf("Failed to invalidate cache: %v", err)
		}
		
		// Verify cache miss after invalidation
		cached := service.GetVerificationResult(req)
		if cached != nil {
			t.Error("Expected nil after cache invalidation")
		}
	})
	
	// Test cache invalidation by pattern
	t.Run("InvalidateCacheByPattern", func(t *testing.T) {
		// Cache multiple results
		req1 := req
		req1.UserID = "user1"
		req2 := req
		req2.UserID = "user2"
		
		service.CacheVerificationResult(req1, response)
		service.CacheVerificationResult(req2, response)
		
		// Invalidate by pattern
		pattern := "verification:rp001:*"
		err := service.InvalidateCacheByPattern(pattern)
		if err != nil {
			t.Errorf("Failed to invalidate cache by pattern: %v", err)
		}
		
		// Verify both are invalidated
		cached1 := service.GetVerificationResult(req1)
		cached2 := service.GetVerificationResult(req2)
		
		if cached1 != nil || cached2 != nil {
			t.Error("Expected both cached results to be invalidated")
		}
	})
}

func TestCacheService_ErrorHandling(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host: "invalid-host",
			Port: 9999,
		},
	}
	
	service := NewCacheService(cfg)
	
	req := models.VerificationRequest{
		UserID:    "user123",
		RPID:      "rp001",
		ClaimType: "student_verification",
	}
	
	response := &models.VerificationResponse{
		VerificationID: "verif_123",
		Verified:       true,
	}
	
	t.Run("CacheErrorHandling", func(t *testing.T) {
		// This should fail due to invalid Redis connection
		service.CacheVerificationResult(req, response)
		
		// Should return nil due to connection error
		cached := service.GetVerificationResult(req)
		if cached != nil {
			t.Error("Expected nil due to connection error")
		}
	})
}

func TestCacheService_ExpiredResponse(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
	}
	
	service := NewCacheService(cfg)
	
	req := models.VerificationRequest{
		UserID:    "user123",
		RPID:      "rp001",
		ClaimType: "student_verification",
	}
	
	// Create response with expired timestamp
	response := &models.VerificationResponse{
		VerificationID: "verif_123",
		Verified:       true,
		ExpiresAt:      time.Now().Add(-1 * time.Hour).Format(time.RFC3339), // Expired
	}
	
	t.Run("ExpiredResponseHandling", func(t *testing.T) {
		service.CacheVerificationResult(req, response)
		
		// Should return nil for expired response
		cached := service.GetVerificationResult(req)
		if cached != nil {
			t.Error("Expected nil for expired response")
		}
	})
}

func TestCacheService_HealthCheck(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
	}
	
	service := NewCacheService(cfg)
	
	t.Run("HealthCheck", func(t *testing.T) {
		ctx := context.Background()
		err := service.HealthCheck(ctx)
		if err != nil {
			t.Errorf("Health check failed: %v", err)
		}
	})
}

func TestCacheService_Close(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
	}
	
	service := NewCacheService(cfg)
	
	t.Run("Close", func(t *testing.T) {
		err := service.Close()
		if err != nil {
			t.Errorf("Failed to close service: %v", err)
		}
	})
} 