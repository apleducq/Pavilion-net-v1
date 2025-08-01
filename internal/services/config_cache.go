package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/pavilion-trust/core-broker/internal/config"
)

// ConfigCacheService handles caching of configuration data
type ConfigCacheService struct {
	config *config.Config
	client *redis.Client
	mu     sync.RWMutex
	// Configuration cache metrics
	dpConfigHits     int64
	policyRuleHits   int64
	policyDecisionHits int64
	dpConfigMisses   int64
	policyRuleMisses int64
	policyDecisionMisses int64
	errors           int64
}

// DPConfig represents cached DP configuration data
type DPConfig struct {
	DPID           string                 `json:"dp_id"`
	Name           string                 `json:"name"`
	Endpoint       string                 `json:"endpoint"`
	SupportedClaims []string              `json:"supported_claims"`
	RateLimit      int                    `json:"rate_limit"`
	Timeout        time.Duration          `json:"timeout"`
	Metadata       map[string]interface{} `json:"metadata"`
	LastUpdated    string                 `json:"last_updated"`
}

// PolicyRule represents cached policy rules
type PolicyRule struct {
	RuleID         string                 `json:"rule_id"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Conditions     map[string]interface{} `json:"conditions"`
	Actions        []string               `json:"actions"`
	Priority       int                    `json:"priority"`
	Enabled        bool                   `json:"enabled"`
	LastUpdated    string                 `json:"last_updated"`
}

// PolicyDecision represents cached policy decisions
type PolicyDecision struct {
	DecisionID     string                 `json:"decision_id"`
	RequestID      string                 `json:"request_id"`
	RPID           string                 `json:"rp_id"`
	ClaimType      string                 `json:"claim_type"`
	Decision       string                 `json:"decision"`
	Reason         string                 `json:"reason"`
	AppliedRules   []string               `json:"applied_rules"`
	Timestamp      string                 `json:"timestamp"`
	ExpiresAt      string                 `json:"expires_at"`
}

// NewConfigCacheService creates a new configuration cache service
func NewConfigCacheService(cfg *config.Config) *ConfigCacheService {
	// Initialize Redis client
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
		PoolSize: 10,
	})

	return &ConfigCacheService{
		config: cfg,
		client: client,
	}
}

// CacheDPConfig caches DP configuration data
func (s *ConfigCacheService) CacheDPConfig(dpConfig *DPConfig) error {
	ctx := context.Background()
	key := fmt.Sprintf("dp_config:%s", dpConfig.DPID)

	// Update timestamp
	dpConfig.LastUpdated = time.Now().Format(time.RFC3339)

	// Serialize config
	data, err := json.Marshal(dpConfig)
	if err != nil {
		s.errors++
		return fmt.Errorf("failed to serialize DP config: %w", err)
	}

	// Cache with 24-hour TTL
	ttl := 24 * time.Hour
	err = s.client.Set(ctx, key, data, ttl).Err()
	if err != nil {
		s.errors++
		return fmt.Errorf("failed to cache DP config: %w", err)
	}

	fmt.Printf("CONFIG CACHE: Cached DP config for %s (TTL: 24 hours)\n", dpConfig.DPID)
	return nil
}

// GetDPConfig retrieves cached DP configuration
func (s *ConfigCacheService) GetDPConfig(dpID string) (*DPConfig, error) {
	ctx := context.Background()
	key := fmt.Sprintf("dp_config:%s", dpID)

	// Get from cache
	result, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			s.dpConfigMisses++
			return nil, nil // Not found, not an error
		}
		s.errors++
		return nil, fmt.Errorf("failed to get DP config from cache: %w", err)
	}

	// Deserialize
	var config DPConfig
	if err := json.Unmarshal([]byte(result), &config); err != nil {
		s.errors++
		return nil, fmt.Errorf("failed to deserialize DP config: %w", err)
	}

	s.dpConfigHits++
	return &config, nil
}

// CachePolicyRule caches policy rules
func (s *ConfigCacheService) CachePolicyRule(rule *PolicyRule) error {
	ctx := context.Background()
	key := fmt.Sprintf("policy_rule:%s", rule.RuleID)

	// Update timestamp
	rule.LastUpdated = time.Now().Format(time.RFC3339)

	// Serialize rule
	data, err := json.Marshal(rule)
	if err != nil {
		s.errors++
		return fmt.Errorf("failed to serialize policy rule: %w", err)
	}

	// Cache with 1-hour TTL (policy rules change less frequently)
	ttl := 1 * time.Hour
	err = s.client.Set(ctx, key, data, ttl).Err()
	if err != nil {
		s.errors++
		return fmt.Errorf("failed to cache policy rule: %w", err)
	}

	fmt.Printf("CONFIG CACHE: Cached policy rule %s (TTL: 1 hour)\n", rule.RuleID)
	return nil
}

// GetPolicyRule retrieves cached policy rule
func (s *ConfigCacheService) GetPolicyRule(ruleID string) (*PolicyRule, error) {
	ctx := context.Background()
	key := fmt.Sprintf("policy_rule:%s", ruleID)

	// Get from cache
	result, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			s.policyRuleMisses++
			return nil, nil // Not found, not an error
		}
		s.errors++
		return nil, fmt.Errorf("failed to get policy rule from cache: %w", err)
	}

	// Deserialize
	var rule PolicyRule
	if err := json.Unmarshal([]byte(result), &rule); err != nil {
		s.errors++
		return nil, fmt.Errorf("failed to deserialize policy rule: %w", err)
	}

	s.policyRuleHits++
	return &rule, nil
}

// CachePolicyDecision caches policy decisions
func (s *ConfigCacheService) CachePolicyDecision(decision *PolicyDecision) error {
	ctx := context.Background()
	key := fmt.Sprintf("policy_decision:%s", decision.DecisionID)

	// Set expiration to 1 hour from now
	decision.ExpiresAt = time.Now().Add(1 * time.Hour).Format(time.RFC3339)

	// Serialize decision
	data, err := json.Marshal(decision)
	if err != nil {
		s.errors++
		return fmt.Errorf("failed to serialize policy decision: %w", err)
	}

	// Cache with 1-hour TTL
	ttl := 1 * time.Hour
	err = s.client.Set(ctx, key, data, ttl).Err()
	if err != nil {
		s.errors++
		return fmt.Errorf("failed to cache policy decision: %w", err)
	}

	fmt.Printf("CONFIG CACHE: Cached policy decision %s (TTL: 1 hour)\n", decision.DecisionID)
	return nil
}

// GetPolicyDecision retrieves cached policy decision
func (s *ConfigCacheService) GetPolicyDecision(decisionID string) (*PolicyDecision, error) {
	ctx := context.Background()
	key := fmt.Sprintf("policy_decision:%s", decisionID)

	// Get from cache
	result, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			s.policyDecisionMisses++
			return nil, nil // Not found, not an error
		}
		s.errors++
		return nil, fmt.Errorf("failed to get policy decision from cache: %w", err)
	}

	// Deserialize
	var decision PolicyDecision
	if err := json.Unmarshal([]byte(result), &decision); err != nil {
		s.errors++
		return nil, fmt.Errorf("failed to deserialize policy decision: %w", err)
	}

	// Check if expired
	if decision.ExpiresAt != "" {
		expiresAt, err := time.Parse(time.RFC3339, decision.ExpiresAt)
		if err == nil && time.Now().After(expiresAt) {
			// Remove expired entry
			s.client.Del(ctx, key)
			s.policyDecisionMisses++
			return nil, nil
		}
	}

	s.policyDecisionHits++
	return &decision, nil
}

// WarmCache performs cache warming for frequently accessed configurations
func (s *ConfigCacheService) WarmCache() error {
	fmt.Println("CONFIG CACHE: Starting cache warming...")

	// Warm DP configurations (mock data for now)
	dpConfigs := []*DPConfig{
		{
			DPID:           "dp-001",
			Name:           "Student Records DP",
			Endpoint:       "https://student-records.example.com/api",
			SupportedClaims: []string{"student_verification", "enrollment_status"},
			RateLimit:      1000,
			Timeout:        30 * time.Second,
			Metadata: map[string]interface{}{
				"region": "us-east-1",
				"version": "1.0.0",
			},
		},
		{
			DPID:           "dp-002",
			Name:           "Employment Records DP",
			Endpoint:       "https://employment-records.example.com/api",
			SupportedClaims: []string{"employee_verification", "employment_status"},
			RateLimit:      500,
			Timeout:        45 * time.Second,
			Metadata: map[string]interface{}{
				"region": "us-west-2",
				"version": "1.1.0",
			},
		},
	}

	// Cache DP configurations
	for _, config := range dpConfigs {
		if err := s.CacheDPConfig(config); err != nil {
			fmt.Printf("CONFIG CACHE WARNING: Failed to warm DP config %s: %v\n", config.DPID, err)
		}
	}

	// Warm policy rules (mock data for now)
	policyRules := []*PolicyRule{
		{
			RuleID:      "rule-001",
			Name:        "Student Verification Rule",
			Description: "Allow student verification requests",
			Conditions: map[string]interface{}{
				"claim_type": "student_verification",
				"rp_trust_level": "high",
			},
			Actions:  []string{"ALLOW"},
			Priority: 1,
			Enabled:  true,
		},
		{
			RuleID:      "rule-002",
			Name:        "Employee Verification Rule",
			Description: "Allow employee verification requests",
			Conditions: map[string]interface{}{
				"claim_type": "employee_verification",
				"rp_trust_level": "medium",
			},
			Actions:  []string{"ALLOW"},
			Priority: 2,
			Enabled:  true,
		},
	}

	// Cache policy rules
	for _, rule := range policyRules {
		if err := s.CachePolicyRule(rule); err != nil {
			fmt.Printf("CONFIG CACHE WARNING: Failed to warm policy rule %s: %v\n", rule.RuleID, err)
		}
	}

	fmt.Println("CONFIG CACHE: Cache warming completed")
	return nil
}

// GetCachePerformance returns cache performance metrics
func (s *ConfigCacheService) GetCachePerformance() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Calculate hit rates
	dpConfigTotal := s.dpConfigHits + s.dpConfigMisses
	policyRuleTotal := s.policyRuleHits + s.policyRuleMisses
	policyDecisionTotal := s.policyDecisionHits + s.policyDecisionMisses

	dpConfigHitRate := 0.0
	if dpConfigTotal > 0 {
		dpConfigHitRate = float64(s.dpConfigHits) / float64(dpConfigTotal)
	}

	policyRuleHitRate := 0.0
	if policyRuleTotal > 0 {
		policyRuleHitRate = float64(s.policyRuleHits) / float64(policyRuleTotal)
	}

	policyDecisionHitRate := 0.0
	if policyDecisionTotal > 0 {
		policyDecisionHitRate = float64(s.policyDecisionHits) / float64(policyDecisionTotal)
	}

	return map[string]interface{}{
		"dp_config": map[string]interface{}{
			"hits":     s.dpConfigHits,
			"misses":   s.dpConfigMisses,
			"hit_rate": dpConfigHitRate,
		},
		"policy_rules": map[string]interface{}{
			"hits":     s.policyRuleHits,
			"misses":   s.policyRuleMisses,
			"hit_rate": policyRuleHitRate,
		},
		"policy_decisions": map[string]interface{}{
			"hits":     s.policyDecisionHits,
			"misses":   s.policyDecisionMisses,
			"hit_rate": policyDecisionHitRate,
		},
		"errors": s.errors,
	}
}

// InvalidateDPConfig invalidates cached DP configuration
func (s *ConfigCacheService) InvalidateDPConfig(dpID string) error {
	ctx := context.Background()
	key := fmt.Sprintf("dp_config:%s", dpID)

	err := s.client.Del(ctx, key).Err()
	if err != nil {
		s.errors++
		return fmt.Errorf("failed to invalidate DP config: %w", err)
	}

	fmt.Printf("CONFIG CACHE: Invalidated DP config for %s\n", dpID)
	return nil
}

// InvalidatePolicyRule invalidates cached policy rule
func (s *ConfigCacheService) InvalidatePolicyRule(ruleID string) error {
	ctx := context.Background()
	key := fmt.Sprintf("policy_rule:%s", ruleID)

	err := s.client.Del(ctx, key).Err()
	if err != nil {
		s.errors++
		return fmt.Errorf("failed to invalidate policy rule: %w", err)
	}

	fmt.Printf("CONFIG CACHE: Invalidated policy rule %s\n", ruleID)
	return nil
}

// InvalidateAllPolicyDecisions invalidates all cached policy decisions
func (s *ConfigCacheService) InvalidateAllPolicyDecisions() error {
	ctx := context.Background()
	pattern := "policy_decision:*"

	// Scan for keys matching pattern
	iter := s.client.Scan(ctx, 0, pattern, 0).Iterator()
	var keys []string

	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan for policy decisions: %w", err)
	}

	// Delete all matching keys
	if len(keys) > 0 {
		err := s.client.Del(ctx, keys...).Err()
		if err != nil {
			s.errors++
			return fmt.Errorf("failed to invalidate policy decisions: %w", err)
		}

		fmt.Printf("CONFIG CACHE: Invalidated %d policy decision entries\n", len(keys))
	}

	return nil
}

// HealthCheck checks if the configuration cache service is healthy
func (s *ConfigCacheService) HealthCheck(ctx context.Context) error {
	// Test Redis connection
	_, err := s.client.Ping(ctx).Result()
	if err != nil {
		return fmt.Errorf("Redis connection failed: %v", err)
	}

	// Test basic operations
	testKey := "config_cache_health_check"
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
func (s *ConfigCacheService) Close() error {
	return s.client.Close()
} 