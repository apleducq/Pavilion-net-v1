package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/models"
)

// CacheService handles Redis caching operations
type CacheService struct {
	config *config.Config
	client *redis.Client
	// Cache metrics
	hitCount   int64
	missCount  int64
	errorCount int64
}

// CacheConfig represents Redis cache configuration
type CacheConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`
	TTL      int    `json:"ttl"` // Time to live in seconds
}

// NewCacheService creates a new cache service
func NewCacheService(cfg *config.Config) *CacheService {
	// Initialize Redis client
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: 10,
	})

	return &CacheService{
		config: cfg,
		client: client,
	}
}

// GetVerificationResult retrieves a cached verification result
func (s *CacheService) GetVerificationResult(req models.VerificationRequest) *models.VerificationResponse {
	ctx := context.Background()
	key := s.generateCacheKey(req)

	// Get from Redis
	result, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			// Key not found - cache miss
			s.missCount++
			return nil
		}
		// Log error but don't fail the request
		fmt.Printf("CACHE ERROR: Failed to get from cache: %v\n", err)
		s.errorCount++
		return nil
	}

	// Deserialize response
	var response models.VerificationResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		fmt.Printf("CACHE ERROR: Failed to deserialize cached response: %v\n", err)
		s.errorCount++
		return nil
	}

	// Check if expired
	if response.ExpiresAt != "" {
		expiresAt, err := time.Parse(time.RFC3339, response.ExpiresAt)
		if err == nil && time.Now().After(expiresAt) {
			// Remove expired entry
			s.client.Del(ctx, key)
			s.missCount++
			return nil
		}
	}

	// Cache hit
	s.hitCount++
	return &response
}

// CacheVerificationResult stores a verification result in cache with 90-day TTL
func (s *CacheService) CacheVerificationResult(req models.VerificationRequest, response *models.VerificationResponse) {
	ctx := context.Background()
	key := s.generateCacheKey(req)

	// Serialize response
	data, err := json.Marshal(response)
	if err != nil {
		fmt.Printf("CACHE ERROR: Failed to serialize response: %v\n", err)
		s.errorCount++
		return
	}

	// Set in Redis with 90-day TTL (T-020 requirement)
	ttl := 90 * 24 * time.Hour // 90 days
	err = s.client.Set(ctx, key, data, ttl).Err()
	if err != nil {
		fmt.Printf("CACHE ERROR: Failed to cache response: %v\n", err)
		s.errorCount++
		return
	}

	fmt.Printf("CACHE: Cached verification result for RP %s, User %s, Claim %s (TTL: 90 days)\n", 
		req.RPID, req.UserID, req.ClaimType)
}

// generateCacheKey creates a cache key for a verification request
func (s *CacheService) generateCacheKey(req models.VerificationRequest) string {
	return fmt.Sprintf("verification:%s:%s:%s", req.RPID, req.UserID, req.ClaimType)
}

// GetCacheKey retrieves a value by key
func (s *CacheService) GetCacheKey(key string) (string, error) {
	ctx := context.Background()
	result, err := s.client.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return result, nil
}

// SetCacheKey sets a value with TTL
func (s *CacheService) SetCacheKey(key string, value string, ttl time.Duration) error {
	ctx := context.Background()
	return s.client.Set(ctx, key, value, ttl).Err()
}

// DeleteCacheKey deletes a key from cache
func (s *CacheService) DeleteCacheKey(key string) error {
	ctx := context.Background()
	return s.client.Del(ctx, key).Err()
}

// GetCacheTTL gets the remaining TTL for a key
func (s *CacheService) GetCacheTTL(key string) (time.Duration, error) {
	ctx := context.Background()
	return s.client.TTL(ctx, key).Result()
}

// FlushCache clears all cache entries
func (s *CacheService) FlushCache() error {
	ctx := context.Background()
	return s.client.FlushDB(ctx).Err()
}

// GetCacheStats gets cache statistics
func (s *CacheService) GetCacheStats() (map[string]interface{}, error) {
	ctx := context.Background()
	info, err := s.client.Info(ctx).Result()
	if err != nil {
		return nil, err
	}

	stats := make(map[string]interface{})
	stats["info"] = info
	stats["dbsize"], _ = s.client.DBSize(ctx).Result()
	
	// Add cache hit/miss metrics (T-020)
	stats["hit_count"] = s.hitCount
	stats["miss_count"] = s.missCount
	stats["error_count"] = s.errorCount
	
	// Calculate hit rate
	totalRequests := s.hitCount + s.missCount
	if totalRequests > 0 {
		stats["hit_rate"] = float64(s.hitCount) / float64(totalRequests)
	} else {
		stats["hit_rate"] = 0.0
	}
	
	return stats, nil
}

// InvalidateVerificationCache invalidates cached verification results
func (s *CacheService) InvalidateVerificationCache(req models.VerificationRequest) error {
	ctx := context.Background()
	key := s.generateCacheKey(req)
	
	err := s.client.Del(ctx, key).Err()
	if err != nil {
		fmt.Printf("CACHE ERROR: Failed to invalidate cache for key %s: %v\n", key, err)
		s.errorCount++
		return err
	}
	
	fmt.Printf("CACHE: Invalidated verification result for RP %s, User %s, Claim %s\n", 
		req.RPID, req.UserID, req.ClaimType)
	return nil
}

// InvalidateCacheByPattern invalidates cache entries matching a pattern
func (s *CacheService) InvalidateCacheByPattern(pattern string) error {
	ctx := context.Background()
	
	// Scan for keys matching pattern
	iter := s.client.Scan(ctx, 0, pattern, 0).Iterator()
	var keys []string
	
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	
	if err := iter.Err(); err != nil {
		return err
	}
	
	// Delete all matching keys
	if len(keys) > 0 {
		err := s.client.Del(ctx, keys...).Err()
		if err != nil {
			fmt.Printf("CACHE ERROR: Failed to invalidate cache by pattern %s: %v\n", pattern, err)
			s.errorCount++
			return err
		}
		
		fmt.Printf("CACHE: Invalidated %d cache entries matching pattern %s\n", len(keys), pattern)
	}
	
	return nil
}

// GetCacheMetrics returns cache performance metrics
func (s *CacheService) GetCacheMetrics() map[string]interface{} {
	totalRequests := s.hitCount + s.missCount
	hitRate := 0.0
	if totalRequests > 0 {
		hitRate = float64(s.hitCount) / float64(totalRequests)
	}
	
	return map[string]interface{}{
		"hit_count":     s.hitCount,
		"miss_count":    s.missCount,
		"error_count":   s.errorCount,
		"total_requests": totalRequests,
		"hit_rate":      hitRate,
	}
}

// HealthCheck checks if the cache service is healthy
func (s *CacheService) HealthCheck(ctx context.Context) error {
	// Test Redis connection
	_, err := s.client.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("Redis connection failed: %v", err)
	}

	// Test basic operations
	testKey := "health_check_test"
	testValue := "test_value"
	
	// Set test value
	err = s.client.Set(ctx, testKey, testValue, time.Minute).Err()
	if err != nil {
		return fmt.Errorf("Redis set operation failed: %v", err)
	}

	// Get test value
	result, err := s.client.Get(ctx, testKey).Result()
	if err != nil {
		return fmt.Errorf("Redis get operation failed: %v", err)
	}

	if result != testValue {
		return fmt.Errorf("Redis get operation returned wrong value")
	}

	// Clean up test key
	s.client.Del(ctx, testKey)

	return nil
}

// Close closes the Redis connection
func (s *CacheService) Close() error {
	return s.client.Close()
} 