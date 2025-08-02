package services

import (
	"context"
	"testing"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
)

func TestConfigCacheService_DPConfigCaching(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
	}

	service := NewConfigCacheService(cfg)

	dpConfig := &DPConfig{
		DPID:            "dp-001",
		Name:            "Student Records DP",
		Endpoint:        "https://student-records.example.com/api",
		SupportedClaims: []string{"student_verification", "enrollment_status"},
		RateLimit:       1000,
		Timeout:         30 * time.Second,
		Metadata: map[string]interface{}{
			"region":  "us-east-1",
			"version": "1.0.0",
		},
	}

	t.Run("CacheDPConfig", func(t *testing.T) {
		err := service.CacheDPConfig(dpConfig)
		if err != nil {
			t.Errorf("Failed to cache DP config: %v", err)
		}
	})

	t.Run("GetDPConfig", func(t *testing.T) {
		cached, err := service.GetDPConfig("dp-001")
		if err != nil {
			t.Errorf("Failed to get DP config: %v", err)
		}

		if cached == nil {
			t.Error("Expected cached DP config, got nil")
		} else if cached.DPID != dpConfig.DPID {
			t.Errorf("Expected DP ID %s, got %s", dpConfig.DPID, cached.DPID)
		} else if cached.Name != dpConfig.Name {
			t.Errorf("Expected name %s, got %s", dpConfig.Name, cached.Name)
		}
	})

	t.Run("GetDPConfig_NotFound", func(t *testing.T) {
		cached, err := service.GetDPConfig("nonexistent")
		if err != nil {
			t.Errorf("Expected no error for non-existent config: %v", err)
		}

		if cached != nil {
			t.Error("Expected nil for non-existent config")
		}
	})
}

func TestConfigCacheService_PolicyRuleCaching(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
	}

	service := NewConfigCacheService(cfg)

	policyRule := &PolicyRule{
		RuleID:      "rule-001",
		Name:        "Student Verification Rule",
		Description: "Allow student verification requests",
		Conditions: map[string]interface{}{
			"claim_type":     "student_verification",
			"rp_trust_level": "high",
		},
		Actions:  []string{"ALLOW"},
		Priority: 1,
		Enabled:  true,
	}

	t.Run("CachePolicyRule", func(t *testing.T) {
		err := service.CachePolicyRule(policyRule)
		if err != nil {
			t.Errorf("Failed to cache policy rule: %v", err)
		}
	})

	t.Run("GetPolicyRule", func(t *testing.T) {
		cached, err := service.GetPolicyRule("rule-001")
		if err != nil {
			t.Errorf("Failed to get policy rule: %v", err)
		}

		if cached == nil {
			t.Error("Expected cached policy rule, got nil")
		} else if cached.RuleID != policyRule.RuleID {
			t.Errorf("Expected rule ID %s, got %s", policyRule.RuleID, cached.RuleID)
		} else if cached.Name != policyRule.Name {
			t.Errorf("Expected name %s, got %s", policyRule.Name, cached.Name)
		}
	})
}

func TestConfigCacheService_PolicyDecisionCaching(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
	}

	service := NewConfigCacheService(cfg)

	decision := &PolicyDecision{
		DecisionID:   "decision-001",
		RequestID:    "req-123",
		RPID:         "rp-001",
		ClaimType:    "student_verification",
		Decision:     "ALLOW",
		Reason:       "Valid student verification request",
		AppliedRules: []string{"rule-001"},
		Timestamp:    time.Now().Format(time.RFC3339),
	}

	t.Run("CachePolicyDecision", func(t *testing.T) {
		err := service.CachePolicyDecision(decision)
		if err != nil {
			t.Errorf("Failed to cache policy decision: %v", err)
		}
	})

	t.Run("GetPolicyDecision", func(t *testing.T) {
		cached, err := service.GetPolicyDecision("decision-001")
		if err != nil {
			t.Errorf("Failed to get policy decision: %v", err)
		}

		if cached == nil {
			t.Error("Expected cached policy decision, got nil")
		} else if cached.DecisionID != decision.DecisionID {
			t.Errorf("Expected decision ID %s, got %s", decision.DecisionID, cached.DecisionID)
		} else if cached.Decision != decision.Decision {
			t.Errorf("Expected decision %s, got %s", decision.Decision, cached.Decision)
		}
	})
}

func TestConfigCacheService_CacheWarming(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
	}

	service := NewConfigCacheService(cfg)

	t.Run("WarmCache", func(t *testing.T) {
		err := service.WarmCache()
		if err != nil {
			t.Errorf("Failed to warm cache: %v", err)
		}

		// Verify DP configs were cached
		dpConfig, err := service.GetDPConfig("dp-001")
		if err != nil {
			t.Errorf("Failed to get warmed DP config: %v", err)
		}

		if dpConfig == nil {
			t.Error("Expected warmed DP config, got nil")
		}

		// Verify policy rules were cached
		policyRule, err := service.GetPolicyRule("rule-001")
		if err != nil {
			t.Errorf("Failed to get warmed policy rule: %v", err)
		}

		if policyRule == nil {
			t.Error("Expected warmed policy rule, got nil")
		}
	})
}

func TestConfigCacheService_PerformanceMetrics(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
	}

	service := NewConfigCacheService(cfg)

	// Generate some cache activity
	dpConfig := &DPConfig{
		DPID: "dp-test",
		Name: "Test DP",
	}

	policyRule := &PolicyRule{
		RuleID: "rule-test",
		Name:   "Test Rule",
	}

	decision := &PolicyDecision{
		DecisionID: "decision-test",
		RequestID:  "req-test",
		RPID:       "rp-test",
		ClaimType:  "test_verification",
		Decision:   "ALLOW",
	}

	t.Run("CachePerformance", func(t *testing.T) {
		// Cache some data
		service.CacheDPConfig(dpConfig)
		service.CachePolicyRule(policyRule)
		service.CachePolicyDecision(decision)

		// Retrieve data to generate hits
		service.GetDPConfig("dp-test")
		service.GetPolicyRule("rule-test")
		service.GetPolicyDecision("decision-test")

		// Get performance metrics
		performance := service.GetCachePerformance()

		if performance["dp_config"] == nil {
			t.Error("Expected DP config metrics")
		}

		if performance["policy_rules"] == nil {
			t.Error("Expected policy rules metrics")
		}

		if performance["policy_decisions"] == nil {
			t.Error("Expected policy decisions metrics")
		}

		if performance["errors"] == nil {
			t.Error("Expected errors metric")
		}
	})
}

func TestConfigCacheService_CacheInvalidation(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
	}

	service := NewConfigCacheService(cfg)

	dpConfig := &DPConfig{
		DPID: "dp-invalidate",
		Name: "Test DP",
	}

	policyRule := &PolicyRule{
		RuleID: "rule-invalidate",
		Name:   "Test Rule",
	}

	t.Run("InvalidateDPConfig", func(t *testing.T) {
		// Cache DP config
		service.CacheDPConfig(dpConfig)

		// Verify it's cached
		cached, _ := service.GetDPConfig("dp-invalidate")
		if cached == nil {
			t.Error("Expected cached DP config before invalidation")
		}

		// Invalidate
		err := service.InvalidateDPConfig("dp-invalidate")
		if err != nil {
			t.Errorf("Failed to invalidate DP config: %v", err)
		}

		// Verify it's gone
		cached, _ = service.GetDPConfig("dp-invalidate")
		if cached != nil {
			t.Error("Expected nil after DP config invalidation")
		}
	})

	t.Run("InvalidatePolicyRule", func(t *testing.T) {
		// Cache policy rule
		service.CachePolicyRule(policyRule)

		// Verify it's cached
		cached, _ := service.GetPolicyRule("rule-invalidate")
		if cached == nil {
			t.Error("Expected cached policy rule before invalidation")
		}

		// Invalidate
		err := service.InvalidatePolicyRule("rule-invalidate")
		if err != nil {
			t.Errorf("Failed to invalidate policy rule: %v", err)
		}

		// Verify it's gone
		cached, _ = service.GetPolicyRule("rule-invalidate")
		if cached != nil {
			t.Error("Expected nil after policy rule invalidation")
		}
	})

	t.Run("InvalidateAllPolicyDecisions", func(t *testing.T) {
		// Cache some policy decisions
		decision1 := &PolicyDecision{
			DecisionID: "decision-1",
			RequestID:  "req-1",
			RPID:       "rp-1",
			ClaimType:  "test",
			Decision:   "ALLOW",
		}

		decision2 := &PolicyDecision{
			DecisionID: "decision-2",
			RequestID:  "req-2",
			RPID:       "rp-2",
			ClaimType:  "test",
			Decision:   "DENY",
		}

		service.CachePolicyDecision(decision1)
		service.CachePolicyDecision(decision2)

		// Verify they're cached
		cached1, _ := service.GetPolicyDecision("decision-1")
		cached2, _ := service.GetPolicyDecision("decision-2")

		if cached1 == nil || cached2 == nil {
			t.Error("Expected cached policy decisions before invalidation")
		}

		// Invalidate all
		err := service.InvalidateAllPolicyDecisions()
		if err != nil {
			t.Errorf("Failed to invalidate all policy decisions: %v", err)
		}

		// Verify they're gone
		cached1, _ = service.GetPolicyDecision("decision-1")
		cached2, _ = service.GetPolicyDecision("decision-2")

		if cached1 != nil || cached2 != nil {
			t.Error("Expected nil after policy decisions invalidation")
		}
	})
}

func TestConfigCacheService_HealthCheck(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
	}

	service := NewConfigCacheService(cfg)

	t.Run("HealthCheck", func(t *testing.T) {
		ctx := context.Background()
		err := service.HealthCheck(ctx)
		if err != nil {
			t.Errorf("Health check failed: %v", err)
		}
	})
}

func TestConfigCacheService_Close(t *testing.T) {
	cfg := &config.Config{
		Redis: config.RedisConfig{
			Host: "localhost",
			Port: 6379,
		},
	}

	service := NewConfigCacheService(cfg)

	t.Run("Close", func(t *testing.T) {
		err := service.Close()
		if err != nil {
			t.Errorf("Failed to close service: %v", err)
		}
	})
}
