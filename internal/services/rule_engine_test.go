package services

import (
	"context"
	"testing"
	"time"

	"github.com/pavilion-trust/core-broker/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestNewRuleEngine(t *testing.T) {
	engine := NewRuleEngine()
	assert.NotNil(t, engine)
	assert.NotNil(t, engine.cache)
}

func TestRuleEngine_EvaluatePolicy_Simple(t *testing.T) {
	engine := NewRuleEngine()

	// Create a simple policy
	conditions := models.PolicyConditions{
		Operator: "AND",
		Rules: []models.PolicyRule{
			{
				Type:           "credential_required",
				CredentialType: "StudentCredential",
			},
		},
	}

	privacy := models.PrivacySettings{
		PPRLEnabled:         true,
		SelectiveDisclosure: true,
		AuditLevel:          "minimal",
		RetentionDays:       90,
	}

	policy := models.NewPolicy("Test Policy", "Test Description", "test-user", conditions, privacy)

	// Create test credentials
	credentials := []models.Credential{
		{
			ID:             "cred-123",
			Type:           "StudentCredential",
			Issuer:         "test-issuer",
			Subject:        "test-subject",
			IssuanceDate:   time.Now().Format(time.RFC3339),
			ExpirationDate: time.Now().Add(24 * time.Hour).Format(time.RFC3339),
			Claims: map[string]interface{}{
				"student_id": "12345",
				"university": "Test University",
			},
			Proof: models.CredentialProof{
				Type:               "Ed25519Signature2020",
				Created:            time.Now().Format(time.RFC3339),
				VerificationMethod: "test-verification-method",
				ProofPurpose:       "assertionMethod",
			},
			Status: "valid",
		},
	}

	// Evaluate policy
	response, err := engine.EvaluatePolicy(context.Background(), policy, credentials)

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.True(t, response.Allowed)
	assert.Equal(t, policy.ID, response.PolicyID)
	assert.Greater(t, response.Confidence, 0.0)
}

func TestRuleEngine_EvaluatePolicy_NoCredentials(t *testing.T) {
	engine := NewRuleEngine()

	// Create a policy that requires credentials
	conditions := models.PolicyConditions{
		Operator: "AND",
		Rules: []models.PolicyRule{
			{
				Type:           "credential_required",
				CredentialType: "StudentCredential",
			},
		},
	}

	privacy := models.PrivacySettings{
		PPRLEnabled:         true,
		SelectiveDisclosure: true,
		AuditLevel:          "minimal",
		RetentionDays:       90,
	}

	policy := models.NewPolicy("Test Policy", "Test Description", "test-user", conditions, privacy)

	// Evaluate policy with no credentials
	response, err := engine.EvaluatePolicy(context.Background(), policy, []models.Credential{})

	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.False(t, response.Allowed)
	assert.Contains(t, response.Reason, "required")
}
