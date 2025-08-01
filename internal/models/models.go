package models

import (
	"fmt"
	"time"
)

// VerificationRequest represents a verification request from an RP
type VerificationRequest struct {
	RPID      string                 `json:"rp_id" validate:"required"`
	UserID    string                 `json:"user_id" validate:"required"`
	ClaimType string                 `json:"claim_type" validate:"required"`
	Identifiers map[string]string    `json:"identifiers" validate:"required"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// Validate validates the verification request
func (r *VerificationRequest) Validate() error {
	if r.RPID == "" {
		return fmt.Errorf("rp_id is required")
	}
	if r.UserID == "" {
		return fmt.Errorf("user_id is required")
	}
	if r.ClaimType == "" {
		return fmt.Errorf("claim_type is required")
	}
	if len(r.Identifiers) == 0 {
		return fmt.Errorf("at least one identifier is required")
	}
	return nil
}

// VerificationResponse represents a verification response to an RP
type VerificationResponse struct {
	VerificationID string  `json:"verification_id"`
	Status         string  `json:"status"` // verified, not_found, error
	ConfidenceScore float64 `json:"confidence_score"`
	Attestation    string  `json:"attestation,omitempty"` // JWS token
	AuditReference string  `json:"audit_reference,omitempty"`
	Timestamp      string  `json:"timestamp"`
	ExpiresAt      string  `json:"expires_at"`
	RequestID      string  `json:"request_id"`
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
	ConfidenceScore float64 `json:"confidence_score"`
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

// HealthStatus represents the health status of a service
type HealthStatus struct {
	Status    string                 `json:"status"`
	Timestamp string                 `json:"timestamp"`
	Version   string                 `json:"version"`
	Details   map[string]interface{} `json:"details,omitempty"`
} 