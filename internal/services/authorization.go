package services

import (
	"context"
	"fmt"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/models"
)

// AuthorizationService handles authorization logic and rules
type AuthorizationService struct {
	config        *config.Config
	policyService *PolicyService
}

// AuthorizationRule defines a specific authorization rule
type AuthorizationRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	RPID        string                 `json:"rp_id,omitempty"`
	ClaimType   string                 `json:"claim_type,omitempty"`
	DPID        string                 `json:"dp_id,omitempty"`
	Conditions  map[string]interface{} `json:"conditions"`
	Action      string                 `json:"action"` // "allow" or "deny"
	Priority    int                    `json:"priority"`
}

// AuthorizationDecision represents an authorization decision
type AuthorizationDecision struct {
	Allowed     bool                   `json:"allowed"`
	Reason      string                 `json:"reason"`
	RuleID      string                 `json:"rule_id"`
	RPID        string                 `json:"rp_id"`
	DPID        string                 `json:"dp_id,omitempty"`
	ClaimType   string                 `json:"claim_type"`
	UserID      string                 `json:"user_id"`
	Timestamp   string                 `json:"timestamp"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// NewAuthorizationService creates a new authorization service
func NewAuthorizationService(cfg *config.Config, policyService *PolicyService) *AuthorizationService {
	return &AuthorizationService{
		config:        cfg,
		policyService: policyService,
	}
}

// AuthorizeRequest performs authorization checks on a verification request
func (s *AuthorizationService) AuthorizeRequest(ctx context.Context, req models.VerificationRequest) (*AuthorizationDecision, error) {
	// Create authorization decision
	decision := &AuthorizationDecision{
		RPID:      req.RPID,
		ClaimType: req.ClaimType,
		UserID:    req.UserID,
		Timestamp: time.Now().Format(time.RFC3339),
		Details:   make(map[string]interface{}),
	}

	// Step 1: Check RP permissions
	if err := s.checkRPPermissions(req, decision); err != nil {
		decision.Allowed = false
		decision.Reason = fmt.Sprintf("RP permission check failed: %s", err.Error())
		decision.RuleID = "rp-permission-check"
		return decision, nil
	}

	// Step 2: Check DP access validation
	if err := s.checkDPAccess(req, decision); err != nil {
		decision.Allowed = false
		decision.Reason = fmt.Sprintf("DP access validation failed: %s", err.Error())
		decision.RuleID = "dp-access-check"
		return decision, nil
	}

	// Step 3: Apply OPA policy enforcement
	if err := s.policyService.EnforcePolicy(ctx, req); err != nil {
		decision.Allowed = false
		decision.Reason = fmt.Sprintf("Policy enforcement failed: %s", err.Error())
		decision.RuleID = "opa-policy-enforcement"
		return decision, nil
	}

	// All checks passed
	decision.Allowed = true
	decision.Reason = "Request authorized"
	decision.RuleID = "authorization-complete"

	// Log the authorization decision
	s.logAuthorizationDecision(decision)

	return decision, nil
}

// checkRPPermissions validates RP permissions for the request
func (s *AuthorizationService) checkRPPermissions(req models.VerificationRequest, decision *AuthorizationDecision) error {
	// Define RP permission rules
	rpRules := s.getRPPermissionRules()

	// Check if RP is allowed for this claim type
	for _, rule := range rpRules {
		if rule.RPID == req.RPID && rule.ClaimType == req.ClaimType {
			if rule.Action == "deny" {
				return fmt.Errorf("RP %s is not authorized for claim type %s", req.RPID, req.ClaimType)
			}
			// Rule allows the request
			decision.Details["rp_rule"] = rule.ID
			return nil
		}
	}

	// Default: allow if no specific rule exists
	return nil
}

// checkDPAccess validates DP access for the request
func (s *AuthorizationService) checkDPAccess(req models.VerificationRequest, decision *AuthorizationDecision) error {
	// Define DP access rules
	dpRules := s.getDPAccessRules()

	// For now, we'll use a simple mapping based on claim type
	// In a real implementation, this would check against actual DP configurations
	dpMapping := map[string]string{
		"student_verification":   "university-dp",
		"employee_verification":  "hr-dp",
		"age_verification":       "government-dp",
		"address_verification":   "postal-dp",
	}

	dpID := dpMapping[req.ClaimType]
	if dpID == "" {
		return fmt.Errorf("no DP mapping found for claim type %s", req.ClaimType)
	}

	// Check if RP has access to this DP
	for _, rule := range dpRules {
		if rule.RPID == req.RPID && rule.DPID == dpID {
			if rule.Action == "deny" {
				return fmt.Errorf("RP %s is not authorized to access DP %s", req.RPID, dpID)
			}
			// Rule allows the access
			decision.DPID = dpID
			decision.Details["dp_rule"] = rule.ID
			return nil
		}
	}

	// Default: allow if no specific rule exists
	decision.DPID = dpID
	return nil
}

// getRPPermissionRules returns the RP permission rules
func (s *AuthorizationService) getRPPermissionRules() []AuthorizationRule {
	return []AuthorizationRule{
		{
			ID:          "rp-student-verification",
			Name:        "Student Verification RP Rule",
			Description: "Allows RPs to request student verification",
			ClaimType:   "student_verification",
			Action:      "allow",
			Priority:    1,
		},
		{
			ID:          "rp-employee-verification",
			Name:        "Employee Verification RP Rule",
			Description: "Allows RPs to request employee verification",
			ClaimType:   "employee_verification",
			Action:      "allow",
			Priority:    1,
		},
		{
			ID:          "rp-age-verification",
			Name:        "Age Verification RP Rule",
			Description: "Allows RPs to request age verification",
			ClaimType:   "age_verification",
			Action:      "allow",
			Priority:    1,
		},
		{
			ID:          "rp-address-verification",
			Name:        "Address Verification RP Rule",
			Description: "Allows RPs to request address verification",
			ClaimType:   "address_verification",
			Action:      "allow",
			Priority:    1,
		},
	}
}

// getDPAccessRules returns the DP access rules
func (s *AuthorizationService) getDPAccessRules() []AuthorizationRule {
	return []AuthorizationRule{
		{
			ID:          "dp-university-access",
			Name:        "University DP Access",
			Description: "Allows RPs to access university DP for student verification",
			DPID:        "university-dp",
			ClaimType:   "student_verification",
			Action:      "allow",
			Priority:    1,
		},
		{
			ID:          "dp-hr-access",
			Name:        "HR DP Access",
			Description: "Allows RPs to access HR DP for employee verification",
			DPID:        "hr-dp",
			ClaimType:   "employee_verification",
			Action:      "allow",
			Priority:    1,
		},
		{
			ID:          "dp-government-access",
			Name:        "Government DP Access",
			Description: "Allows RPs to access government DP for age verification",
			DPID:        "government-dp",
			ClaimType:   "age_verification",
			Action:      "allow",
			Priority:    1,
		},
		{
			ID:          "dp-postal-access",
			Name:        "Postal DP Access",
			Description: "Allows RPs to access postal DP for address verification",
			DPID:        "postal-dp",
			ClaimType:   "address_verification",
			Action:      "allow",
			Priority:    1,
		},
	}
}

// logAuthorizationDecision logs the authorization decision
func (s *AuthorizationService) logAuthorizationDecision(decision *AuthorizationDecision) {
	// In a real implementation, this would log to an audit service
	// For now, we'll just add it to the decision details
	decision.Details["logged"] = true
	decision.Details["log_timestamp"] = time.Now().Format(time.RFC3339)
}

// GetAuthorizationStats returns authorization statistics for monitoring
func (s *AuthorizationService) GetAuthorizationStats() map[string]interface{} {
	return map[string]interface{}{
		"rp_rules_count": len(s.getRPPermissionRules()),
		"dp_rules_count": len(s.getDPAccessRules()),
		"service_status": "active",
	}
}

// HealthCheck checks if the authorization service is healthy
func (s *AuthorizationService) HealthCheck(ctx context.Context) error {
	// Check if policy service is available
	if err := s.policyService.HealthCheck(ctx); err != nil {
		return fmt.Errorf("authorization service health check failed: %w", err)
	}

	return nil
} 