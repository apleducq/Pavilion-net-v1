package services

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/models"
)

func TestNewAuthorizationService(t *testing.T) {
	cfg := &config.Config{
		OPAURL:     "http://opa:8181",
		OPATimeout: 5 * time.Second,
	}

	policyService := NewPolicyService(cfg)
	authService := NewAuthorizationService(cfg, policyService)

	if authService == nil {
		t.Fatal("Expected authorization service to be created")
	}

	if authService.config != cfg {
		t.Error("Expected config to be set")
	}

	if authService.policyService != policyService {
		t.Error("Expected policy service to be set")
	}
}

func TestAuthorizeRequest_Success(t *testing.T) {
	cfg := &config.Config{
		OPAURL:     "http://invalid-opa-url:8181", // Will fail OPA but pass other checks
		OPATimeout: 1 * time.Second,
	}

	policyService := NewPolicyService(cfg)
	authService := NewAuthorizationService(cfg, policyService)

	req := models.VerificationRequest{
		RPID:      "test-rp",
		UserID:    "test-user",
		ClaimType: "student_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}

	decision, err := authService.AuthorizeRequest(context.Background(), req)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if decision == nil {
		t.Fatal("Expected authorization decision")
	}

	// Should fail due to OPA policy enforcement
	if decision.Allowed {
		t.Error("Expected request to be denied due to OPA failure")
	}

	if decision.RuleID != "opa-policy-enforcement" {
		t.Errorf("Expected rule ID 'opa-policy-enforcement', got %s", decision.RuleID)
	}

	if decision.RPID != req.RPID {
		t.Errorf("Expected RPID %s, got %s", req.RPID, decision.RPID)
	}

	if decision.ClaimType != req.ClaimType {
		t.Errorf("Expected claim type %s, got %s", req.ClaimType, decision.ClaimType)
	}

	if decision.UserID != req.UserID {
		t.Errorf("Expected user ID %s, got %s", req.UserID, decision.UserID)
	}
}

func TestAuthorizeRequest_InvalidClaimType(t *testing.T) {
	cfg := &config.Config{
		OPAURL:     "http://invalid-opa-url:8181",
		OPATimeout: 1 * time.Second,
	}

	policyService := NewPolicyService(cfg)
	authService := NewAuthorizationService(cfg, policyService)

	req := models.VerificationRequest{
		RPID:      "test-rp",
		UserID:    "test-user",
		ClaimType: "invalid_claim_type",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}

	decision, err := authService.AuthorizeRequest(context.Background(), req)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if decision == nil {
		t.Fatal("Expected authorization decision")
	}

	// Should fail due to DP access validation
	if decision.Allowed {
		t.Error("Expected request to be denied due to invalid claim type")
	}

	if decision.RuleID != "dp-access-check" {
		t.Errorf("Expected rule ID 'dp-access-check', got %s", decision.RuleID)
	}
}

func TestGetRPPermissionRules(t *testing.T) {
	cfg := &config.Config{}
	policyService := NewPolicyService(cfg)
	authService := NewAuthorizationService(cfg, policyService)

	rules := authService.getRPPermissionRules()

	if len(rules) != 4 {
		t.Errorf("Expected 4 RP permission rules, got %d", len(rules))
	}

	// Check that all claim types are covered
	claimTypes := make(map[string]bool)
	for _, rule := range rules {
		claimTypes[rule.ClaimType] = true
	}

	expectedClaimTypes := []string{
		"student_verification",
		"employee_verification", 
		"age_verification",
		"address_verification",
	}

	for _, expected := range expectedClaimTypes {
		if !claimTypes[expected] {
			t.Errorf("Expected claim type %s to be covered", expected)
		}
	}
}

func TestGetDPAccessRules(t *testing.T) {
	cfg := &config.Config{}
	policyService := NewPolicyService(cfg)
	authService := NewAuthorizationService(cfg, policyService)

	rules := authService.getDPAccessRules()

	if len(rules) != 4 {
		t.Errorf("Expected 4 DP access rules, got %d", len(rules))
	}

	// Check that all DPs are covered
	dps := make(map[string]bool)
	for _, rule := range rules {
		dps[rule.DPID] = true
	}

	expectedDPs := []string{
		"university-dp",
		"hr-dp",
		"government-dp", 
		"postal-dp",
	}

	for _, expected := range expectedDPs {
		if !dps[expected] {
			t.Errorf("Expected DP %s to be covered", expected)
		}
	}
}

func TestCheckRPPermissions(t *testing.T) {
	cfg := &config.Config{}
	policyService := NewPolicyService(cfg)
	authService := NewAuthorizationService(cfg, policyService)

	req := models.VerificationRequest{
		RPID:      "test-rp",
		UserID:    "test-user",
		ClaimType: "student_verification",
	}

	decision := &AuthorizationDecision{
		RPID:      req.RPID,
		ClaimType: req.ClaimType,
		UserID:    req.UserID,
		Timestamp: time.Now().Format(time.RFC3339),
		Details:   make(map[string]interface{}),
	}

	// Test successful RP permission check
	err := authService.checkRPPermissions(req, decision)
	if err != nil {
		t.Errorf("Expected no error for valid RP permissions, got %v", err)
	}

	// Test with invalid claim type
	req.ClaimType = "invalid_claim_type"
	err = authService.checkRPPermissions(req, decision)
	if err != nil {
		t.Errorf("Expected no error for invalid claim type (default allow), got %v", err)
	}
}

func TestCheckDPAccess(t *testing.T) {
	cfg := &config.Config{}
	policyService := NewPolicyService(cfg)
	authService := NewAuthorizationService(cfg, policyService)

	req := models.VerificationRequest{
		RPID:      "test-rp",
		UserID:    "test-user",
		ClaimType: "student_verification",
	}

	decision := &AuthorizationDecision{
		RPID:      req.RPID,
		ClaimType: req.ClaimType,
		UserID:    req.UserID,
		Timestamp: time.Now().Format(time.RFC3339),
		Details:   make(map[string]interface{}),
	}

	// Test successful DP access check
	err := authService.checkDPAccess(req, decision)
	if err != nil {
		t.Errorf("Expected no error for valid DP access, got %v", err)
	}

	if decision.DPID != "university-dp" {
		t.Errorf("Expected DP ID 'university-dp', got %s", decision.DPID)
	}

	// Test with invalid claim type
	req.ClaimType = "invalid_claim_type"
	err = authService.checkDPAccess(req, decision)
	if err == nil {
		t.Error("Expected error for invalid claim type")
	}
}

func TestGetAuthorizationStats(t *testing.T) {
	cfg := &config.Config{}
	policyService := NewPolicyService(cfg)
	authService := NewAuthorizationService(cfg, policyService)

	stats := authService.GetAuthorizationStats()

	if stats["rp_rules_count"] != 4 {
		t.Errorf("Expected 4 RP rules, got %v", stats["rp_rules_count"])
	}

	if stats["dp_rules_count"] != 4 {
		t.Errorf("Expected 4 DP rules, got %v", stats["dp_rules_count"])
	}

	if stats["service_status"] != "active" {
		t.Errorf("Expected service status 'active', got %v", stats["service_status"])
	}
}

func TestHealthCheck(t *testing.T) {
	cfg := &config.Config{
		OPAURL:     "http://invalid-opa-url:8181",
		OPATimeout: 1 * time.Second,
	}

	policyService := NewPolicyService(cfg)
	authService := NewAuthorizationService(cfg, policyService)

	// Test health check (should fail due to OPA service being unavailable)
	err := authService.HealthCheck(context.Background())
	if err == nil {
		t.Error("Expected health check to fail due to OPA service being unavailable")
	}

	if !strings.Contains(err.Error(), "authorization service health check") {
		t.Errorf("Expected authorization service health check error, got %v", err)
	}
}

func TestLogAuthorizationDecision(t *testing.T) {
	cfg := &config.Config{}
	policyService := NewPolicyService(cfg)
	authService := NewAuthorizationService(cfg, policyService)

	decision := &AuthorizationDecision{
		Allowed:   true,
		Reason:    "Test decision",
		RuleID:    "test-rule",
		RPID:      "test-rp",
		ClaimType: "student_verification",
		UserID:    "test-user",
		Timestamp: time.Now().Format(time.RFC3339),
		Details:   make(map[string]interface{}),
	}

	// Test logging
	authService.logAuthorizationDecision(decision)

	if decision.Details["logged"] != true {
		t.Error("Expected decision to be marked as logged")
	}

	if decision.Details["log_timestamp"] == "" {
		t.Error("Expected log timestamp to be set")
	}
} 