package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// VerificationRequest represents a verification request from an RP
type VerificationRequest struct {
	RPID        string                 `json:"rp_id" validate:"required,min=1,max=100"`
	UserID      string                 `json:"user_id" validate:"required,min=1,max=100"`
	ClaimType   string                 `json:"claim_type" validate:"required,oneof=student_verification employee_verification age_verification address_verification"`
	Identifiers map[string]string     `json:"identifiers" validate:"required,min=1,dive,keys,required,endkeys,required"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Validate validates the verification request
func (r *VerificationRequest) Validate() error {
	validate := validator.New()
	if err := validate.Struct(r); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	
	// Additional business logic validation
	if len(r.Identifiers) == 0 {
		return fmt.Errorf("at least one identifier is required")
	}
	
	// Validate identifier keys
	validKeys := map[string]bool{
		"email": true, "name": true, "phone": true, "address": true,
		"ssn": true, "passport": true, "license": true,
	}
	
	for key := range r.Identifiers {
		if !validKeys[key] {
			return fmt.Errorf("invalid identifier key: %s", key)
		}
	}
	
	return nil
}

// ToJSON converts the request to JSON
func (r *VerificationRequest) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// FromJSON creates a VerificationRequest from JSON
func FromJSON(data []byte) (*VerificationRequest, error) {
	var req VerificationRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	return &req, nil
}

// VerificationResponse represents a verification response
type VerificationResponse struct {
	VerificationID  string  `json:"verification_id" validate:"required"`
	Status          string  `json:"status" validate:"required,oneof=verified not_found error"` // verified, not_found, error
	Verified        bool    `json:"verified"`
	ConfidenceScore float64 `json:"confidence_score" validate:"min=0,max=1"`
	Reason          string  `json:"reason,omitempty"`
	Evidence        []string `json:"evidence,omitempty"`
	DPID            string  `json:"dp_id"`
	Attestation     string  `json:"attestation,omitempty"` // JWS token
	AuditReference  string  `json:"audit_reference,omitempty"`
	Timestamp       string  `json:"timestamp" validate:"required"`
	ExpiresAt       string  `json:"expires_at" validate:"required"`
	RequestID       string  `json:"request_id" validate:"required"`
	ProcessingTime  string  `json:"processing_time,omitempty"`
	RequestHash     string  `json:"request_hash,omitempty"`
	ResponseHash    string  `json:"response_hash,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	ValidationErrors []string              `json:"validation_errors,omitempty"`
	Error           *Error  `json:"error,omitempty"`
}

// NewVerificationResponse creates a new verification response
func NewVerificationResponse(status string, confidenceScore float64, requestID string) *VerificationResponse {
	now := time.Now()
	expiresAt := now.Add(90 * 24 * time.Hour) // 90 days
	
	return &VerificationResponse{
		VerificationID:  uuid.New().String(),
		Status:          status,
		ConfidenceScore: confidenceScore,
		Timestamp:       now.Format(time.RFC3339),
		ExpiresAt:       expiresAt.Format(time.RFC3339),
		RequestID:       requestID,
	}
}

// ToJSON converts the response to JSON
func (r *VerificationResponse) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// Validate validates the verification response
func (r *VerificationResponse) Validate() error {
	validate := validator.New()
	return validate.Struct(r)
}

// PrivacyRequest represents a privacy-transformed request for DP communication
type PrivacyRequest struct {
	RPID      string                 `json:"rp_id"`
	UserHash  string                 `json:"user_hash"`
	ClaimType string                 `json:"claim_type"`
	HashedIdentifiers map[string]string `json:"hashed_identifiers"`
	BloomFilters     map[string]string `json:"bloom_filters,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// DPResponse represents a response from a Data Provider
type DPResponse struct {
	Status         string  `json:"status"`
	Verified       bool    `json:"verified"`
	ConfidenceScore float64 `json:"confidence_score"`
	Reason         string  `json:"reason,omitempty"`
	Evidence       []string `json:"evidence,omitempty"`
	DPID          string  `json:"dp_id"`
	Timestamp     string  `json:"timestamp"`
	Error         string  `json:"error,omitempty"`
}

// PolicyDecision represents a policy enforcement decision
type PolicyDecision struct {
	Allowed    bool   `json:"allowed"`
	Reason     string `json:"reason"`
	PolicyID   string `json:"policy_id"`
	Timestamp  string `json:"timestamp"`
}

// AuditEntry represents an audit log entry
type AuditEntry struct {
	Timestamp     string                 `json:"timestamp"`
	RequestID     string                 `json:"request_id"`
	RPID          string                 `json:"rp_id"`
	DPID          string                 `json:"dp_id,omitempty"`
	ClaimType     string                 `json:"claim_type"`
	PrivacyHash   string                 `json:"privacy_hash"`
	MerkleProof   string                 `json:"merkle_proof"`
	PolicyDecision string                `json:"policy_decision"`
	Status        string                 `json:"status"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// CacheEntry represents a cached verification result
type CacheEntry struct {
	Request   *VerificationRequest  `json:"request"`
	Response  *VerificationResponse `json:"response"`
	Timestamp time.Time             `json:"timestamp"`
	ExpiresAt time.Time             `json:"expires_at"`
}

// Error represents a structured error response
type Error struct {
	Code      string `json:"code" validate:"required"`
	Message   string `json:"message" validate:"required"`
	Timestamp string `json:"timestamp" validate:"required"`
	RequestID string `json:"request_id,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// NewError creates a new error
func NewError(code, message, requestID string) *Error {
	return &Error{
		Code:      code,
		Message:   message,
		Timestamp: time.Now().Format(time.RFC3339),
		RequestID: requestID,
	}
}

// ToJSON converts the error to JSON
func (e *Error) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// ErrorResponse represents a complete error response
type ErrorResponse struct {
	Error *Error `json:"error" validate:"required"`
}

// NewErrorResponse creates a new error response
func NewErrorResponse(code, message, requestID string) *ErrorResponse {
	return &ErrorResponse{
		Error: NewError(code, message, requestID),
	}
}

// ToJSON converts the error response to JSON
func (r *ErrorResponse) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// HealthStatus represents the health status of a service
type HealthStatus struct {
	Status    string                 `json:"status"`
	Timestamp string                 `json:"timestamp"`
	Version   string                 `json:"version"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
}

// ValidationErrors represents multiple validation errors
type ValidationErrors struct {
	Errors []ValidationError `json:"errors"`
}

// AddError adds a validation error
func (v *ValidationErrors) AddError(field, message, value string) {
	v.Errors = append(v.Errors, ValidationError{
		Field:   field,
		Message: message,
		Value:   value,
	})
}

// HasErrors returns true if there are validation errors
func (v *ValidationErrors) HasErrors() bool {
	return len(v.Errors) > 0
}

// ToJSON converts validation errors to JSON
func (v *ValidationErrors) ToJSON() ([]byte, error) {
	return json.Marshal(v)
} 