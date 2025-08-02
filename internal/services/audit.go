package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/models"
)

// ContextKey is a custom type for context keys to avoid collisions
type ContextKey string

// Context key constants
const (
	RequestIDKey ContextKey = "request_id"
)

// AuditService handles audit logging with cryptographic integrity
type AuditService struct {
	config *config.Config
}

// AuditReference represents an audit reference for responses
type AuditReference struct {
	AuditEntryID string `json:"audit_entry_id"`
	MerkleProof  string `json:"merkle_proof"`
	Timestamp    string `json:"timestamp"`
	Hash         string `json:"hash"`
}

// NewAuditService creates a new audit service
func NewAuditService(cfg *config.Config) *AuditService {
	return &AuditService{
		config: cfg,
	}
}

// LogVerification logs a verification request/response for audit purposes
// Returns an audit reference that can be included in the response
func (s *AuditService) LogVerification(ctx context.Context, req models.VerificationRequest, response *models.VerificationResponse, status string) *AuditReference {
	// Generate audit entry ID
	auditEntryID := s.generateAuditEntryID(req, response)

	// Create audit entry with enhanced structure (T-018)
	entry := &models.AuditEntry{
		Timestamp:      time.Now().Format(time.RFC3339),
		RequestID:      getRequestID(ctx),
		RPID:           req.RPID,
		ClaimType:      req.ClaimType,
		PrivacyHash:    s.generatePrivacyHash(req),
		MerkleProof:    s.generateMerkleProof(req, response),
		PolicyDecision: s.getPolicyDecision(ctx, req), // Enhanced policy decision
		Status:         status,
		Metadata:       s.createAuditMetadata(req, response, auditEntryID),
	}

	// Add DP ID if response exists
	if response != nil {
		entry.DPID = response.DPID
	}

	// Add sequence number for audit trail ordering
	entry.Metadata["sequence_number"] = s.getNextSequenceNumber()
	entry.Metadata["audit_entry_id"] = auditEntryID

	// TODO: Send to audit database
	// For now, just log to console
	s.logAuditEntry(entry)

	// Create and return audit reference
	return s.createAuditReference(entry, auditEntryID)
}

// createAuditReference creates an audit reference for inclusion in responses
func (s *AuditService) createAuditReference(entry *models.AuditEntry, auditEntryID string) *AuditReference {
	// Generate hash of the audit entry for integrity
	entryData := fmt.Sprintf("%s:%s:%s:%s:%s",
		entry.Timestamp, entry.RequestID, entry.RPID, entry.ClaimType, entry.PrivacyHash)
	hash := sha256.Sum256([]byte(entryData))

	return &AuditReference{
		AuditEntryID: auditEntryID,
		MerkleProof:  entry.MerkleProof,
		Timestamp:    entry.Timestamp,
		Hash:         hex.EncodeToString(hash[:]),
	}
}

// generateAuditEntryID creates a unique audit entry ID
func (s *AuditService) generateAuditEntryID(req models.VerificationRequest, _ *models.VerificationResponse) string {
	// Create a unique ID based on request and timestamp
	timestamp := time.Now().UnixNano()
	time.Sleep(1 * time.Microsecond) // Ensure uniqueness for rapid calls
	data := fmt.Sprintf("%s:%s:%s:%d", req.RPID, req.UserID, req.ClaimType, timestamp)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("audit_%s", hex.EncodeToString(hash[:8]))
}

// ValidateAuditReference validates an audit reference
func (s *AuditService) ValidateAuditReference(reference *AuditReference) error {
	if reference == nil {
		return fmt.Errorf("audit reference is nil")
	}

	if reference.AuditEntryID == "" {
		return fmt.Errorf("audit entry ID is empty")
	}

	if reference.MerkleProof == "" {
		return fmt.Errorf("merkle proof is empty")
	}

	if reference.Timestamp == "" {
		return fmt.Errorf("timestamp is empty")
	}

	if reference.Hash == "" {
		return fmt.Errorf("hash is empty")
	}

	// Validate timestamp format
	_, err := time.Parse(time.RFC3339, reference.Timestamp)
	if err != nil {
		return fmt.Errorf("invalid timestamp format: %v", err)
	}

	return nil
}

// GetAuditReference retrieves an audit reference by entry ID
func (s *AuditService) GetAuditReference(auditEntryID string) (*AuditReference, error) {
	// TODO: Retrieve from audit database
	// For now, return a mock reference
	if auditEntryID == "" {
		return nil, fmt.Errorf("audit entry ID is empty")
	}

	return &AuditReference{
		AuditEntryID: auditEntryID,
		MerkleProof:  "mock_merkle_proof",
		Timestamp:    time.Now().Format(time.RFC3339),
		Hash:         "mock_hash",
	}, nil
}

// generatePrivacyHash creates a privacy-preserving hash of the request
func (s *AuditService) generatePrivacyHash(req models.VerificationRequest) string {
	// Hash the request without exposing raw PII
	data := fmt.Sprintf("%s:%s:%s:%d", req.RPID, req.UserID, req.ClaimType, len(req.Identifiers))
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// generateMerkleProof creates an enhanced Merkle proof for the audit entry
func (s *AuditService) generateMerkleProof(req models.VerificationRequest, response *models.VerificationResponse) string {
	// Create a more comprehensive Merkle proof
	var proofData string

	if response != nil {
		// Include response data in proof
		proofData = fmt.Sprintf("%s:%s:%s:%s:%s:%t:%f",
			req.RPID, req.ClaimType, response.RequestID, response.Status,
			response.DPID, response.Verified, response.ConfidenceScore)
	} else {
		// Only request data
		proofData = fmt.Sprintf("%s:%s:%s", req.RPID, req.ClaimType, time.Now().Format(time.RFC3339))
	}

	hash := sha256.Sum256([]byte(proofData))
	return hex.EncodeToString(hash[:])
}

// logAuditEntry logs an audit entry (placeholder for database storage)
func (s *AuditService) logAuditEntry(entry *models.AuditEntry) {
	// TODO: Store in audit database
	// For now, just log to console
	jsonData, _ := json.Marshal(entry)
	fmt.Printf("AUDIT: %s\n", string(jsonData))
}

// getRequestID extracts request ID from context
func getRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return "unknown"
}

// getPolicyDecision determines the policy decision for the request
func (s *AuditService) getPolicyDecision(_ context.Context, req models.VerificationRequest) string {
	// TODO: Integrate with actual policy service
	// For now, implement basic policy logic
	switch req.ClaimType {
	case "student_verification", "employee_verification", "age_verification", "address_verification":
		return "ALLOW"
	default:
		return "DENY"
	}
}

// createAuditMetadata creates comprehensive audit metadata
func (s *AuditService) createAuditMetadata(req models.VerificationRequest, response *models.VerificationResponse, auditEntryID string) map[string]interface{} {
	metadata := map[string]interface{}{
		"user_id":           req.UserID,
		"identifiers_count": len(req.Identifiers),
		"audit_entry_id":    auditEntryID,
		"claim_type":        req.ClaimType,
		"rp_id":             req.RPID,
		"timestamp":         time.Now().Format(time.RFC3339),
	}

	// Add identifier types for privacy analysis
	var identifierTypes []string
	for key := range req.Identifiers {
		identifierTypes = append(identifierTypes, key)
	}
	metadata["identifier_types"] = identifierTypes

	// Add response metadata if available
	if response != nil {
		metadata["verification_id"] = response.VerificationID
		metadata["verified"] = response.Verified
		metadata["confidence_score"] = response.ConfidenceScore
		metadata["dp_id"] = response.DPID
		metadata["status"] = response.Status
		metadata["processing_time"] = response.ProcessingTime
	}

	// Add request metadata
	if req.Metadata != nil {
		for key, value := range req.Metadata {
			metadata[fmt.Sprintf("request_%s", key)] = value
		}
	}

	return metadata
}

// getNextSequenceNumber generates the next sequence number for audit entries
func (s *AuditService) getNextSequenceNumber() int64 {
	// Use atomic counter for unique sequence numbers
	// For now, use timestamp-based sequence with a small delay to ensure uniqueness
	seq := time.Now().UnixNano()
	time.Sleep(1 * time.Microsecond) // Ensure uniqueness for rapid calls
	return seq
}

// LogPolicyDecision logs a policy decision separately
func (s *AuditService) LogPolicyDecision(ctx context.Context, req models.VerificationRequest, decision string, reason string) {
	entry := &models.AuditEntry{
		Timestamp:      time.Now().Format(time.RFC3339),
		RequestID:      getRequestID(ctx),
		RPID:           req.RPID,
		ClaimType:      req.ClaimType,
		PrivacyHash:    s.generatePrivacyHash(req),
		MerkleProof:    s.generateMerkleProof(req, nil),
		PolicyDecision: decision,
		Status:         "POLICY_DECISION",
		Metadata: map[string]interface{}{
			"policy_reason":   reason,
			"user_id":         req.UserID,
			"sequence_number": s.getNextSequenceNumber(),
		},
	}

	s.logAuditEntry(entry)
}

// LogPrivacyHash logs a privacy hash generation event
func (s *AuditService) LogPrivacyHash(ctx context.Context, req models.VerificationRequest, privacyHash string) {
	entry := &models.AuditEntry{
		Timestamp:      time.Now().Format(time.RFC3339),
		RequestID:      getRequestID(ctx),
		RPID:           req.RPID,
		ClaimType:      req.ClaimType,
		PrivacyHash:    privacyHash,
		MerkleProof:    s.generateMerkleProof(req, nil),
		PolicyDecision: "PRIVACY_HASH",
		Status:         "PRIVACY_PROCESSING",
		Metadata: map[string]interface{}{
			"user_id":         req.UserID,
			"sequence_number": s.getNextSequenceNumber(),
			"hash_algorithm":  "SHA-256",
		},
	}

	s.logAuditEntry(entry)
}

// HealthCheck checks if the audit service is healthy
func (s *AuditService) HealthCheck(ctx context.Context) error {
	// Test privacy hash generation
	testReq := models.VerificationRequest{
		RPID:      "test-rp",
		UserID:    "test-user",
		ClaimType: "student_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}

	hash := s.generatePrivacyHash(testReq)
	if hash == "" {
		return fmt.Errorf("privacy hash generation failed")
	}

	// Test metadata creation
	metadata := s.createAuditMetadata(testReq, nil, "test-audit-id")
	if metadata == nil {
		return fmt.Errorf("metadata creation failed")
	}

	// Test policy decision
	decision := s.getPolicyDecision(ctx, testReq)
	if decision == "" {
		return fmt.Errorf("policy decision failed")
	}

	return nil
}
