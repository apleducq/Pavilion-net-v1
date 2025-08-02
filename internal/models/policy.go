package models

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// Policy represents a verification policy
type Policy struct {
	ID          string                 `json:"id" validate:"required"`
	Version     string                 `json:"version" validate:"required"`
	Name        string                 `json:"name" validate:"required,min=1,max=100"`
	Description string                 `json:"description" validate:"max=500"`
	Conditions  PolicyConditions       `json:"conditions" validate:"required"`
	Privacy     PrivacySettings        `json:"privacy" validate:"required"`
	CreatedAt   string                `json:"created_at" validate:"required"`
	UpdatedAt   string                `json:"updated_at" validate:"required"`
	CreatedBy   string                `json:"created_by" validate:"required"`
	Status      string                 `json:"status" validate:"required,oneof=active inactive draft"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// PolicyConditions represents the conditions for a policy
type PolicyConditions struct {
	Operator string        `json:"operator" validate:"required,oneof=AND OR NOT"`
	Rules    []PolicyRule  `json:"rules" validate:"required,min=1,dive"`
}

// PolicyRule represents a single policy rule
type PolicyRule struct {
	Type           string                 `json:"type" validate:"required,oneof=credential_required claim_equals claim_greater_than claim_less_than claim_in_range issuer_trusted not_expired"`
	CredentialType string                 `json:"credential_type,omitempty"`
	Issuer         string                 `json:"issuer,omitempty"`
	Claim          string                 `json:"claim,omitempty"`
	Value          interface{}            `json:"value,omitempty"`
	MinValue       interface{}            `json:"min_value,omitempty"`
	MaxValue       interface{}            `json:"max_value,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// PrivacySettings represents privacy configuration for a policy
type PrivacySettings struct {
	PPRLEnabled        bool   `json:"pprl_enabled"`
	SelectiveDisclosure bool   `json:"selective_disclosure"`
	AuditLevel         string `json:"audit_level" validate:"required,oneof=minimal standard detailed"`
	RetentionDays      int    `json:"retention_days" validate:"min=1,max=3650"`
}

// PolicyTemplate represents a policy template
type PolicyTemplate struct {
	ID          string                 `json:"id" validate:"required"`
	Name        string                 `json:"name" validate:"required"`
	Description string                 `json:"description" validate:"max=500"`
	Category    string                 `json:"category" validate:"required,oneof=age_verification student_verification employment_verification address_verification"`
	Template    Policy                 `json:"template" validate:"required"`
	CreatedAt   string                `json:"created_at" validate:"required"`
	UpdatedAt   string                `json:"updated_at" validate:"required"`
	CreatedBy   string                `json:"created_by" validate:"required"`
	Status      string                 `json:"status" validate:"required,oneof=active inactive draft"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// PolicyEvaluationRequest represents a policy evaluation request
type PolicyEvaluationRequest struct {
	PolicyID    string                 `json:"policy_id" validate:"required"`
	Credentials []Credential           `json:"credentials" validate:"required,min=1,dive"`
	Context     map[string]interface{} `json:"context,omitempty"`
	RequestID   string                 `json:"request_id" validate:"required"`
	Timestamp   string                 `json:"timestamp" validate:"required"`
}

// Credential represents a verifiable credential
type Credential struct {
	ID           string                 `json:"id" validate:"required"`
	Type         string                 `json:"type" validate:"required"`
	Issuer       string                 `json:"issuer" validate:"required"`
	Subject      string                 `json:"subject" validate:"required"`
	IssuanceDate string                 `json:"issuance_date" validate:"required"`
	ExpirationDate string               `json:"expiration_date,omitempty"`
	Claims       map[string]interface{} `json:"claims" validate:"required"`
	Proof        CredentialProof        `json:"proof" validate:"required"`
	Status       string                 `json:"status" validate:"required,oneof=valid revoked expired"`
}

// CredentialProof represents a credential proof
type CredentialProof struct {
	Type               string `json:"type" validate:"required"`
	Created            string `json:"created" validate:"required"`
	VerificationMethod string `json:"verification_method" validate:"required"`
	ProofPurpose       string `json:"proof_purpose" validate:"required"`
	JWS                string `json:"jws,omitempty"`
}

// PolicyEvaluationResponse represents a policy evaluation response
type PolicyEvaluationResponse struct {
	RequestID    string                 `json:"request_id" validate:"required"`
	PolicyID     string                 `json:"policy_id" validate:"required"`
	Allowed      bool                   `json:"allowed"`
	Reason       string                 `json:"reason"`
	Confidence   float64                `json:"confidence" validate:"min=0,max=1"`
	EvaluatedAt  string                 `json:"evaluated_at" validate:"required"`
	ProcessingTime string               `json:"processing_time,omitempty"`
	Evidence     []string               `json:"evidence,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// PolicyStorage interface defines methods for policy storage
type PolicyStorage interface {
	CreatePolicy(ctx context.Context, policy *Policy) error
	GetPolicy(ctx context.Context, id string) (*Policy, error)
	UpdatePolicy(ctx context.Context, policy *Policy) error
	DeletePolicy(ctx context.Context, id string) error
	ListPolicies(ctx context.Context, filters map[string]interface{}) ([]*Policy, error)
	CreateTemplate(ctx context.Context, template *PolicyTemplate) error
	GetTemplate(ctx context.Context, id string) (*PolicyTemplate, error)
	ListTemplates(ctx context.Context, filters map[string]interface{}) ([]*PolicyTemplate, error)
}

// NewPolicy creates a new policy
func NewPolicy(name, description, createdBy string, conditions PolicyConditions, privacy PrivacySettings) *Policy {
	now := time.Now().Format(time.RFC3339)
	return &Policy{
		ID:          uuid.New().String(),
		Version:     "1.0",
		Name:        name,
		Description: description,
		Conditions:  conditions,
		Privacy:     privacy,
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   createdBy,
		Status:      "draft",
	}
}

// Validate validates the policy
func (p *Policy) Validate() error {
	validate := validator.New()
	if err := validate.Struct(p); err != nil {
		return fmt.Errorf("policy validation failed: %w", err)
	}
	return nil
}

// ToJSON converts the policy to JSON
func (p *Policy) ToJSON() ([]byte, error) {
	return json.Marshal(p)
}

// FromJSON creates a Policy from JSON
func PolicyFromJSON(data []byte) (*Policy, error) {
	var policy Policy
	if err := json.Unmarshal(data, &policy); err != nil {
		return nil, fmt.Errorf("failed to parse policy JSON: %w", err)
	}
	return &policy, nil
}

// NewPolicyTemplate creates a new policy template
func NewPolicyTemplate(name, description, category, createdBy string, template Policy) *PolicyTemplate {
	now := time.Now().Format(time.RFC3339)
	return &PolicyTemplate{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Category:    category,
		Template:    template,
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedBy:   createdBy,
		Status:      "active",
	}
}

// Validate validates the policy template
func (pt *PolicyTemplate) Validate() error {
	validate := validator.New()
	if err := validate.Struct(pt); err != nil {
		return fmt.Errorf("policy template validation failed: %w", err)
	}
	return nil
}

// ToJSON converts the policy template to JSON
func (pt *PolicyTemplate) ToJSON() ([]byte, error) {
	return json.Marshal(pt)
}

// FromJSON creates a PolicyTemplate from JSON
func PolicyTemplateFromJSON(data []byte) (*PolicyTemplate, error) {
	var template PolicyTemplate
	if err := json.Unmarshal(data, &template); err != nil {
		return nil, fmt.Errorf("failed to parse policy template JSON: %w", err)
	}
	return &template, nil
}

// NewPolicyEvaluationRequest creates a new policy evaluation request
func NewPolicyEvaluationRequest(policyID string, credentials []Credential, requestID string) *PolicyEvaluationRequest {
	return &PolicyEvaluationRequest{
		PolicyID:    policyID,
		Credentials: credentials,
		RequestID:   requestID,
		Timestamp:   time.Now().Format(time.RFC3339),
	}
}

// Validate validates the policy evaluation request
func (per *PolicyEvaluationRequest) Validate() error {
	validate := validator.New()
	if err := validate.Struct(per); err != nil {
		return fmt.Errorf("policy evaluation request validation failed: %w", err)
	}
	return nil
}

// ToJSON converts the policy evaluation request to JSON
func (per *PolicyEvaluationRequest) ToJSON() ([]byte, error) {
	return json.Marshal(per)
}

// FromJSON creates a PolicyEvaluationRequest from JSON
func PolicyEvaluationRequestFromJSON(data []byte) (*PolicyEvaluationRequest, error) {
	var request PolicyEvaluationRequest
	if err := json.Unmarshal(data, &request); err != nil {
		return nil, fmt.Errorf("failed to parse policy evaluation request JSON: %w", err)
	}
	return &request, nil
}

// NewPolicyEvaluationResponse creates a new policy evaluation response
func NewPolicyEvaluationResponse(requestID, policyID string, allowed bool, reason string, confidence float64) *PolicyEvaluationResponse {
	return &PolicyEvaluationResponse{
		RequestID:   requestID,
		PolicyID:    policyID,
		Allowed:     allowed,
		Reason:      reason,
		Confidence:  confidence,
		EvaluatedAt: time.Now().Format(time.RFC3339),
	}
}

// Validate validates the policy evaluation response
func (per *PolicyEvaluationResponse) Validate() error {
	validate := validator.New()
	if err := validate.Struct(per); err != nil {
		return fmt.Errorf("policy evaluation response validation failed: %w", err)
	}
	return nil
}

// ToJSON converts the policy evaluation response to JSON
func (per *PolicyEvaluationResponse) ToJSON() ([]byte, error) {
	return json.Marshal(per)
}

// FromJSON creates a PolicyEvaluationResponse from JSON
func PolicyEvaluationResponseFromJSON(data []byte) (*PolicyEvaluationResponse, error) {
	var response PolicyEvaluationResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse policy evaluation response JSON: %w", err)
	}
	return &response, nil
} 