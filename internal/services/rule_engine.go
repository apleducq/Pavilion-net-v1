package services

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pavilion-trust/core-broker/internal/models"
)

// RuleEngine handles policy rule evaluation
type RuleEngine struct {
	cache *RuleCache
}

// RuleCache provides caching for rule evaluation results
type RuleCache struct {
	results map[string]*RuleResult
	mu      sync.RWMutex
	ttl     time.Duration
}

// RuleResult represents the result of a rule evaluation
type RuleResult struct {
	Allowed   bool      `json:"allowed"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
	ExpiresAt time.Time `json:"expires_at"`
}

// NewRuleCache creates a new rule cache
func NewRuleCache(ttl time.Duration) *RuleCache {
	return &RuleCache{
		results: make(map[string]*RuleResult),
		ttl:     ttl,
	}
}

// Get retrieves a cached rule result
func (c *RuleCache) Get(key string) (*RuleResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	result, exists := c.results[key]
	if !exists {
		return nil, false
	}
	
	// Check if result has expired
	if time.Now().After(result.ExpiresAt) {
		delete(c.results, key)
		return nil, false
	}
	
	return result, true
}

// Set stores a rule result in cache
func (c *RuleCache) Set(key string, result *RuleResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.results[key] = result
}

// NewRuleEngine creates a new rule engine
func NewRuleEngine() *RuleEngine {
	return &RuleEngine{
		cache: NewRuleCache(5 * time.Minute), // Cache rule results for 5 minutes
	}
}

// EvaluatePolicy evaluates a policy against provided credentials
func (re *RuleEngine) EvaluatePolicy(ctx context.Context, policy *models.Policy, credentials []models.Credential) (*models.PolicyEvaluationResponse, error) {
	startTime := time.Now()

	// Validate policy
	if err := policy.Validate(); err != nil {
		return nil, fmt.Errorf("policy validation failed: %w", err)
	}

	// Evaluate conditions
	result, err := re.evaluateConditions(ctx, policy.Conditions, credentials)
	if err != nil {
		return nil, fmt.Errorf("condition evaluation failed: %w", err)
	}

	// Calculate confidence score based on evaluation results
	confidence := re.calculateConfidence(result)

	// Create evaluation response
	response := models.NewPolicyEvaluationResponse(
		"", // RequestID will be set by caller
		policy.ID,
		result.Allowed,
		result.Reason,
		confidence,
	)

	// Add processing time
	response.ProcessingTime = time.Since(startTime).String()

	return response, nil
}

// evaluateConditions evaluates policy conditions
func (re *RuleEngine) evaluateConditions(ctx context.Context, conditions models.PolicyConditions, credentials []models.Credential) (*RuleResult, error) {
	// Generate cache key
	cacheKey := re.generateCacheKey(conditions, credentials)
	
	// Check cache first
	if cached, exists := re.cache.Get(cacheKey); exists {
		return cached, nil
	}

	// Evaluate rules based on operator
	var results []*RuleResult
	for _, rule := range conditions.Rules {
		result, err := re.evaluateRule(ctx, rule, credentials)
		if err != nil {
			return nil, fmt.Errorf("rule evaluation failed: %w", err)
		}
		results = append(results, result)
	}

	// Combine results based on operator
	var finalResult *RuleResult
	switch strings.ToUpper(conditions.Operator) {
	case "AND":
		finalResult = re.combineAND(results)
	case "OR":
		finalResult = re.combineOR(results)
	case "NOT":
		if len(results) != 1 {
			return nil, fmt.Errorf("NOT operator requires exactly one rule")
		}
		finalResult = re.combineNOT(results[0])
	default:
		return nil, fmt.Errorf("unsupported operator: %s", conditions.Operator)
	}

	// Cache the result
	re.cache.Set(cacheKey, finalResult)

	return finalResult, nil
}

// evaluateRule evaluates a single policy rule
func (re *RuleEngine) evaluateRule(ctx context.Context, rule models.PolicyRule, credentials []models.Credential) (*RuleResult, error) {
	switch rule.Type {
	case "credential_required":
		return re.evaluateCredentialRequired(rule, credentials)
	case "claim_equals":
		return re.evaluateClaimEquals(rule, credentials)
	case "claim_greater_than":
		return re.evaluateClaimGreaterThan(rule, credentials)
	case "claim_less_than":
		return re.evaluateClaimLessThan(rule, credentials)
	case "claim_in_range":
		return re.evaluateClaimInRange(rule, credentials)
	case "issuer_trusted":
		return re.evaluateIssuerTrusted(rule, credentials)
	case "not_expired":
		return re.evaluateNotExpired(rule, credentials)
	default:
		return nil, fmt.Errorf("unsupported rule type: %s", rule.Type)
	}
}

// evaluateCredentialRequired checks if a specific credential type is present
func (re *RuleEngine) evaluateCredentialRequired(rule models.PolicyRule, credentials []models.Credential) (*RuleResult, error) {
	for _, cred := range credentials {
		if cred.Type == rule.CredentialType {
			return &RuleResult{
				Allowed:   true,
				Reason:    fmt.Sprintf("Required credential type '%s' found", rule.CredentialType),
				Timestamp: time.Now(),
				ExpiresAt: time.Now().Add(re.cache.ttl),
			}, nil
		}
	}

	return &RuleResult{
		Allowed:   false,
		Reason:    fmt.Sprintf("Required credential type '%s' not found", rule.CredentialType),
		Timestamp: time.Now(),
		ExpiresAt: time.Now().Add(re.cache.ttl),
	}, nil
}

// evaluateClaimEquals checks if a claim equals a specific value
func (re *RuleEngine) evaluateClaimEquals(rule models.PolicyRule, credentials []models.Credential) (*RuleResult, error) {
	for _, cred := range credentials {
		if claimValue, exists := cred.Claims[rule.Claim]; exists {
			if reflect.DeepEqual(claimValue, rule.Value) {
				return &RuleResult{
					Allowed:   true,
					Reason:    fmt.Sprintf("Claim '%s' equals expected value", rule.Claim),
					Timestamp: time.Now(),
					ExpiresAt: time.Now().Add(re.cache.ttl),
				}, nil
			}
		}
	}

	return &RuleResult{
		Allowed:   false,
		Reason:    fmt.Sprintf("Claim '%s' does not equal expected value", rule.Claim),
		Timestamp: time.Now(),
		ExpiresAt: time.Now().Add(re.cache.ttl),
	}, nil
}

// evaluateClaimGreaterThan checks if a claim is greater than a specific value
func (re *RuleEngine) evaluateClaimGreaterThan(rule models.PolicyRule, credentials []models.Credential) (*RuleResult, error) {
	for _, cred := range credentials {
		if claimValue, exists := cred.Claims[rule.Claim]; exists {
			// Convert to comparable values
			claimNum, err := re.convertToNumber(claimValue)
			if err != nil {
				continue
			}
			
			ruleNum, err := re.convertToNumber(rule.Value)
			if err != nil {
				return nil, fmt.Errorf("invalid rule value: %w", err)
			}

			if claimNum > ruleNum {
				return &RuleResult{
					Allowed:   true,
					Reason:    fmt.Sprintf("Claim '%s' is greater than %v", rule.Claim, rule.Value),
					Timestamp: time.Now(),
					ExpiresAt: time.Now().Add(re.cache.ttl),
				}, nil
			}
		}
	}

	return &RuleResult{
		Allowed:   false,
		Reason:    fmt.Sprintf("Claim '%s' is not greater than %v", rule.Claim, rule.Value),
		Timestamp: time.Now(),
		ExpiresAt: time.Now().Add(re.cache.ttl),
	}, nil
}

// evaluateClaimLessThan checks if a claim is less than a specific value
func (re *RuleEngine) evaluateClaimLessThan(rule models.PolicyRule, credentials []models.Credential) (*RuleResult, error) {
	for _, cred := range credentials {
		if claimValue, exists := cred.Claims[rule.Claim]; exists {
			// Convert to comparable values
			claimNum, err := re.convertToNumber(claimValue)
			if err != nil {
				continue
			}
			
			ruleNum, err := re.convertToNumber(rule.Value)
			if err != nil {
				return nil, fmt.Errorf("invalid rule value: %w", err)
			}

			if claimNum < ruleNum {
				return &RuleResult{
					Allowed:   true,
					Reason:    fmt.Sprintf("Claim '%s' is less than %v", rule.Claim, rule.Value),
					Timestamp: time.Now(),
					ExpiresAt: time.Now().Add(re.cache.ttl),
				}, nil
			}
		}
	}

	return &RuleResult{
		Allowed:   false,
		Reason:    fmt.Sprintf("Claim '%s' is not less than %v", rule.Claim, rule.Value),
		Timestamp: time.Now(),
		ExpiresAt: time.Now().Add(re.cache.ttl),
	}, nil
}

// evaluateClaimInRange checks if a claim is within a specific range
func (re *RuleEngine) evaluateClaimInRange(rule models.PolicyRule, credentials []models.Credential) (*RuleResult, error) {
	for _, cred := range credentials {
		if claimValue, exists := cred.Claims[rule.Claim]; exists {
			// Convert to comparable values
			claimNum, err := re.convertToNumber(claimValue)
			if err != nil {
				continue
			}
			
			minNum, err := re.convertToNumber(rule.MinValue)
			if err != nil {
				return nil, fmt.Errorf("invalid min value: %w", err)
			}
			
			maxNum, err := re.convertToNumber(rule.MaxValue)
			if err != nil {
				return nil, fmt.Errorf("invalid max value: %w", err)
			}

			if claimNum >= minNum && claimNum <= maxNum {
				return &RuleResult{
					Allowed:   true,
					Reason:    fmt.Sprintf("Claim '%s' is within range [%v, %v]", rule.Claim, rule.MinValue, rule.MaxValue),
					Timestamp: time.Now(),
					ExpiresAt: time.Now().Add(re.cache.ttl),
				}, nil
			}
		}
	}

	return &RuleResult{
		Allowed:   false,
		Reason:    fmt.Sprintf("Claim '%s' is not within range [%v, %v]", rule.Claim, rule.MinValue, rule.MaxValue),
		Timestamp: time.Now(),
		ExpiresAt: time.Now().Add(re.cache.ttl),
	}, nil
}

// evaluateIssuerTrusted checks if the credential issuer is trusted
func (re *RuleEngine) evaluateIssuerTrusted(rule models.PolicyRule, credentials []models.Credential) (*RuleResult, error) {
	for _, cred := range credentials {
		if cred.Issuer == rule.Issuer {
			return &RuleResult{
				Allowed:   true,
				Reason:    fmt.Sprintf("Credential issuer '%s' is trusted", rule.Issuer),
				Timestamp: time.Now(),
				ExpiresAt: time.Now().Add(re.cache.ttl),
			}, nil
		}
	}

	return &RuleResult{
		Allowed:   false,
		Reason:    fmt.Sprintf("No credential from trusted issuer '%s' found", rule.Issuer),
		Timestamp: time.Now(),
		ExpiresAt: time.Now().Add(re.cache.ttl),
	}, nil
}

// evaluateNotExpired checks if credentials are not expired
func (re *RuleEngine) evaluateNotExpired(rule models.PolicyRule, credentials []models.Credential) (*RuleResult, error) {
	now := time.Now()
	
	for _, cred := range credentials {
		if cred.ExpirationDate != "" {
			expiration, err := time.Parse(time.RFC3339, cred.ExpirationDate)
			if err != nil {
				continue // Skip invalid dates
			}
			
			if now.Before(expiration) {
				return &RuleResult{
					Allowed:   true,
					Reason:    "Credential is not expired",
					Timestamp: time.Now(),
					ExpiresAt: time.Now().Add(re.cache.ttl),
				}, nil
			}
		}
	}

	return &RuleResult{
		Allowed:   false,
		Reason:    "All credentials are expired",
		Timestamp: time.Now(),
		ExpiresAt: time.Now().Add(re.cache.ttl),
	}, nil
}

// combineAND combines results with AND logic
func (re *RuleEngine) combineAND(results []*RuleResult) *RuleResult {
	for _, result := range results {
		if !result.Allowed {
			return &RuleResult{
				Allowed:   false,
				Reason:    fmt.Sprintf("AND condition failed: %s", result.Reason),
				Timestamp: time.Now(),
				ExpiresAt: time.Now().Add(re.cache.ttl),
			}
		}
	}

	return &RuleResult{
		Allowed:   true,
		Reason:    "All AND conditions satisfied",
		Timestamp: time.Now(),
		ExpiresAt: time.Now().Add(re.cache.ttl),
	}
}

// combineOR combines results with OR logic
func (re *RuleEngine) combineOR(results []*RuleResult) *RuleResult {
	for _, result := range results {
		if result.Allowed {
			return &RuleResult{
				Allowed:   true,
				Reason:    fmt.Sprintf("OR condition satisfied: %s", result.Reason),
				Timestamp: time.Now(),
				ExpiresAt: time.Now().Add(re.cache.ttl),
			}
		}
	}

	return &RuleResult{
		Allowed:   false,
		Reason:    "No OR conditions satisfied",
		Timestamp: time.Now(),
		ExpiresAt: time.Now().Add(re.cache.ttl),
	}
}

// combineNOT negates a result
func (re *RuleEngine) combineNOT(result *RuleResult) *RuleResult {
	return &RuleResult{
		Allowed:   !result.Allowed,
		Reason:    fmt.Sprintf("NOT condition: %s", result.Reason),
		Timestamp: time.Now(),
		ExpiresAt: time.Now().Add(re.cache.ttl),
	}
}

// convertToNumber converts a value to a number for comparison
func (re *RuleEngine) convertToNumber(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("cannot convert %v to number", value)
	}
}

// generateCacheKey creates a cache key for rule evaluation
func (re *RuleEngine) generateCacheKey(conditions models.PolicyConditions, credentials []models.Credential) string {
	// Simple hash-based key generation
	// In a real implementation, you might want to use a proper hash function
	key := fmt.Sprintf("%s-%d-%d", conditions.Operator, len(conditions.Rules), len(credentials))
	return key
}

// calculateConfidence calculates a confidence score based on evaluation results
func (re *RuleEngine) calculateConfidence(result *RuleResult) float64 {
	if result.Allowed {
		return 0.95 // High confidence for allowed results
	}
	return 0.85 // Slightly lower confidence for denied results
} 