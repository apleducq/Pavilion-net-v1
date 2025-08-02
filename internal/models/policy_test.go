package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewPolicy(t *testing.T) {
	conditions := PolicyConditions{
		Operator: "AND",
		Rules: []PolicyRule{
			{
				Type:           "credential_required",
				CredentialType: "StudentCredential",
			},
		},
	}

	privacy := PrivacySettings{
		PPRLEnabled:        true,
		SelectiveDisclosure: true,
		AuditLevel:         "minimal",
		RetentionDays:      90,
	}

	policy := NewPolicy("Test Policy", "Test Description", "test-user", conditions, privacy)

	assert.NotEmpty(t, policy.ID)
	assert.Equal(t, "1.0", policy.Version)
	assert.Equal(t, "Test Policy", policy.Name)
	assert.Equal(t, "Test Description", policy.Description)
	assert.Equal(t, "test-user", policy.CreatedBy)
	assert.Equal(t, "draft", policy.Status)
	assert.NotEmpty(t, policy.CreatedAt)
	assert.NotEmpty(t, policy.UpdatedAt)
}

func TestPolicy_Validate(t *testing.T) {
	tests := []struct {
		name    string
		policy  *Policy
		wantErr bool
	}{
		{
			name: "valid policy",
			policy: &Policy{
				ID:          "test-id",
				Version:     "1.0",
				Name:        "Test Policy",
				Description: "Test Description",
				Conditions: PolicyConditions{
					Operator: "AND",
					Rules: []PolicyRule{
						{
							Type:           "credential_required",
							CredentialType: "StudentCredential",
						},
					},
				},
				Privacy: PrivacySettings{
					PPRLEnabled:        true,
					SelectiveDisclosure: true,
					AuditLevel:         "minimal",
					RetentionDays:      90,
				},
				CreatedAt: time.Now().Format(time.RFC3339),
				UpdatedAt: time.Now().Format(time.RFC3339),
				CreatedBy: "test-user",
				Status:    "active",
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			policy: &Policy{
				Version:     "1.0",
				Name:        "Test Policy",
				Description: "Test Description",
				Conditions: PolicyConditions{
					Operator: "AND",
					Rules:    []PolicyRule{},
				},
				Privacy: PrivacySettings{
					PPRLEnabled:        true,
					SelectiveDisclosure: true,
					AuditLevel:         "minimal",
					RetentionDays:      90,
				},
				CreatedAt: time.Now().Format(time.RFC3339),
				UpdatedAt: time.Now().Format(time.RFC3339),
				CreatedBy: "test-user",
				Status:    "active",
			},
			wantErr: true,
		},
		{
			name: "invalid status",
			policy: &Policy{
				ID:          "test-id",
				Version:     "1.0",
				Name:        "Test Policy",
				Description: "Test Description",
				Conditions: PolicyConditions{
					Operator: "AND",
					Rules:    []PolicyRule{},
				},
				Privacy: PrivacySettings{
					PPRLEnabled:        true,
					SelectiveDisclosure: true,
					AuditLevel:         "minimal",
					RetentionDays:      90,
				},
				CreatedAt: time.Now().Format(time.RFC3339),
				UpdatedAt: time.Now().Format(time.RFC3339),
				CreatedBy: "test-user",
				Status:    "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.policy.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPolicy_ToJSON(t *testing.T) {
	policy := &Policy{
		ID:          "test-id",
		Version:     "1.0",
		Name:        "Test Policy",
		Description: "Test Description",
		Conditions: PolicyConditions{
			Operator: "AND",
			Rules: []PolicyRule{
				{
					Type:           "credential_required",
					CredentialType: "StudentCredential",
				},
			},
		},
		Privacy: PrivacySettings{
			PPRLEnabled:        true,
			SelectiveDisclosure: true,
			AuditLevel:         "minimal",
			RetentionDays:      90,
		},
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
		CreatedBy: "test-user",
		Status:    "active",
	}

	jsonData, err := policy.ToJSON()
	assert.NoError(t, err)
	assert.NotEmpty(t, jsonData)

	// Verify we can unmarshal it back
	var unmarshaled Policy
	err = json.Unmarshal(jsonData, &unmarshaled)
	assert.NoError(t, err)
	assert.Equal(t, policy.ID, unmarshaled.ID)
	assert.Equal(t, policy.Name, unmarshaled.Name)
}

func TestPolicyFromJSON(t *testing.T) {
	policy := &Policy{
		ID:          "test-id",
		Version:     "1.0",
		Name:        "Test Policy",
		Description: "Test Description",
		Conditions: PolicyConditions{
			Operator: "AND",
			Rules: []PolicyRule{
				{
					Type:           "credential_required",
					CredentialType: "StudentCredential",
				},
			},
		},
		Privacy: PrivacySettings{
			PPRLEnabled:        true,
			SelectiveDisclosure: true,
			AuditLevel:         "minimal",
			RetentionDays:      90,
		},
		CreatedAt: time.Now().Format(time.RFC3339),
		UpdatedAt: time.Now().Format(time.RFC3339),
		CreatedBy: "test-user",
		Status:    "active",
	}

	jsonData, err := policy.ToJSON()
	assert.NoError(t, err)

	unmarshaled, err := PolicyFromJSON(jsonData)
	assert.NoError(t, err)
	assert.Equal(t, policy.ID, unmarshaled.ID)
	assert.Equal(t, policy.Name, unmarshaled.Name)
	assert.Equal(t, policy.Description, unmarshaled.Description)
}

func TestNewPolicyEvaluationRequest(t *testing.T) {
	credentials := []Credential{
		{
			ID:           "cred-1",
			Type:         "StudentCredential",
			Issuer:       "university.edu",
			Subject:      "student-123",
			IssuanceDate: time.Now().Format(time.RFC3339),
			Claims: map[string]interface{}{
				"status": "enrolled",
				"age":    20,
			},
			Proof: CredentialProof{
				Type:               "JWT",
				Created:            time.Now().Format(time.RFC3339),
				VerificationMethod: "https://university.edu/keys/1",
				ProofPurpose:       "assertionMethod",
			},
			Status: "valid",
		},
	}

	request := NewPolicyEvaluationRequest("policy-123", credentials, "request-456")

	assert.Equal(t, "policy-123", request.PolicyID)
	assert.Equal(t, "request-456", request.RequestID)
	assert.Len(t, request.Credentials, 1)
	assert.NotEmpty(t, request.Timestamp)
}

func TestPolicyEvaluationRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request *PolicyEvaluationRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: &PolicyEvaluationRequest{
				PolicyID: "policy-123",
				Credentials: []Credential{
					{
						ID:           "cred-1",
						Type:         "StudentCredential",
						Issuer:       "university.edu",
						Subject:      "student-123",
						IssuanceDate: time.Now().Format(time.RFC3339),
						Claims: map[string]interface{}{
							"status": "enrolled",
						},
						Proof: CredentialProof{
							Type:               "JWT",
							Created:            time.Now().Format(time.RFC3339),
							VerificationMethod: "https://university.edu/keys/1",
							ProofPurpose:       "assertionMethod",
						},
						Status: "valid",
					},
				},
				RequestID: "request-456",
				Timestamp: time.Now().Format(time.RFC3339),
			},
			wantErr: false,
		},
		{
			name: "missing policy ID",
			request: &PolicyEvaluationRequest{
				Credentials: []Credential{
					{
						ID:           "cred-1",
						Type:         "StudentCredential",
						Issuer:       "university.edu",
						Subject:      "student-123",
						IssuanceDate: time.Now().Format(time.RFC3339),
						Claims: map[string]interface{}{
							"status": "enrolled",
						},
						Proof: CredentialProof{
							Type:               "JWT",
							Created:            time.Now().Format(time.RFC3339),
							VerificationMethod: "https://university.edu/keys/1",
							ProofPurpose:       "assertionMethod",
						},
						Status: "valid",
					},
				},
				RequestID: "request-456",
				Timestamp: time.Now().Format(time.RFC3339),
			},
			wantErr: true,
		},
		{
			name: "empty credentials",
			request: &PolicyEvaluationRequest{
				PolicyID:   "policy-123",
				Credentials: []Credential{},
				RequestID:   "request-456",
				Timestamp:   time.Now().Format(time.RFC3339),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewPolicyEvaluationResponse(t *testing.T) {
	response := NewPolicyEvaluationResponse("request-456", "policy-123", true, "Policy evaluation successful", 0.95)

	assert.Equal(t, "request-456", response.RequestID)
	assert.Equal(t, "policy-123", response.PolicyID)
	assert.True(t, response.Allowed)
	assert.Equal(t, "Policy evaluation successful", response.Reason)
	assert.Equal(t, 0.95, response.Confidence)
	assert.NotEmpty(t, response.EvaluatedAt)
}

func TestPolicyEvaluationResponse_Validate(t *testing.T) {
	tests := []struct {
		name     string
		response *PolicyEvaluationResponse
		wantErr  bool
	}{
		{
			name: "valid response",
			response: &PolicyEvaluationResponse{
				RequestID:   "request-456",
				PolicyID:    "policy-123",
				Allowed:     true,
				Reason:      "Policy evaluation successful",
				Confidence:  0.95,
				EvaluatedAt: time.Now().Format(time.RFC3339),
			},
			wantErr: false,
		},
		{
			name: "missing request ID",
			response: &PolicyEvaluationResponse{
				PolicyID:    "policy-123",
				Allowed:     true,
				Reason:      "Policy evaluation successful",
				Confidence:  0.95,
				EvaluatedAt: time.Now().Format(time.RFC3339),
			},
			wantErr: true,
		},
		{
			name: "invalid confidence",
			response: &PolicyEvaluationResponse{
				RequestID:   "request-456",
				PolicyID:    "policy-123",
				Allowed:     true,
				Reason:      "Policy evaluation successful",
				Confidence:  1.5, // Invalid: > 1.0
				EvaluatedAt: time.Now().Format(time.RFC3339),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.response.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
} 