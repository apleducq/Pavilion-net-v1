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
			// Key not found
			return nil
		}
		// Log error but don't fail the request
		fmt.Printf("CACHE ERROR: Failed to get from cache: %v\n", err)
		return nil
	}

	// Deserialize response
	var response models.VerificationResponse
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		fmt.Printf("CACHE ERROR: Failed to deserialize cached response: %v\n", err)
		return nil
	}

	// Check if expired
	if response.ExpiresAt != "" {
		expiresAt, err := time.Parse(time.RFC3339, response.ExpiresAt)
		if err == nil && time.Now().After(expiresAt) {
			// Remove expired entry
			s.client.Del(ctx, key)
			return nil
		}
	}

	return &response
}

// CacheVerificationResult stores a verification result in cache
func (s *CacheService) CacheVerificationResult(req models.VerificationRequest, response *models.VerificationResponse) {
	ctx := context.Background()
	key := s.generateCacheKey(req)

	// Serialize response
	data, err := json.Marshal(response)
	if err != nil {
		fmt.Printf("CACHE ERROR: Failed to serialize response: %v\n", err)
		return
	}

	// Set in Redis with TTL
	ttl := time.Duration(s.config.Redis.TTL) * time.Second
	err = s.client.Set(ctx, key, data, ttl).Err()
	if err != nil {
		fmt.Printf("CACHE ERROR: Failed to cache response: %v\n", err)
		return
	}

	fmt.Printf("CACHE: Cached verification result for RP %s, User %s, Claim %s (TTL: %s)\n", 
		req.RPID, req.UserID, req.ClaimType, ttl)
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
	
	return stats, nil
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