package services

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/models"
)

// PolicyService handles policy enforcement using OPA
type PolicyService struct {
	config *config.Config
	client *http.Client
}

// NewPolicyService creates a new policy service
func NewPolicyService(cfg *config.Config) *PolicyService {
	return &PolicyService{
		config: cfg,
		client: &http.Client{
			Timeout: cfg.OPATimeout,
		},
	}
}

// EnforcePolicy checks if the request is allowed based on policies
func (s *PolicyService) EnforcePolicy(ctx context.Context, req models.VerificationRequest) error {
	// Create policy query
	query := map[string]interface{}{
		"input": map[string]interface{}{
			"rp_id":      req.RPID,
			"claim_type": req.ClaimType,
			"user_id":    req.UserID,
		},
	}

	// Send query to OPA
	decision, err := s.queryOPA(ctx, query)
	if err != nil {
		return fmt.Errorf("policy query failed: %w", err)
	}

	// Check if request is allowed
	if !decision.Allowed {
		return fmt.Errorf("policy violation: %s", decision.Reason)
	}

	return nil
}

// queryOPA sends a query to the OPA service
func (s *PolicyService) queryOPA(ctx context.Context, query map[string]interface{}) (*models.PolicyDecision, error) {
	// TODO: Implement actual OPA integration
	// For now, return a mock decision that allows all requests
	
	return &models.PolicyDecision{
		Allowed:   true,
		Reason:    "Mock policy decision",
		PolicyID:  "mock-policy-001",
		Timestamp: time.Now().Format(time.RFC3339),
	}, nil
}

// HealthCheck checks if the policy service is healthy
func (s *PolicyService) HealthCheck(ctx context.Context) error {
	// TODO: Implement actual health check
	// For now, always return healthy
	return nil
} 