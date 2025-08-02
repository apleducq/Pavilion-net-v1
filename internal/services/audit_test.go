package services

import (
	"context"
	"testing"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuditService(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	assert.NotNil(t, service)
	assert.NotNil(t, service.config)
	assert.Equal(t, cfg, service.config)
}

func TestAuditService_LogVerification_Success(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Create a test request
	req := models.VerificationRequest{
		RPID:      "test-rp-001",
		UserID:    "user-123",
		ClaimType: "identity_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
			"phone": "+1234567890",
		},
	}

	// Create a test response
	response := &models.VerificationResponse{
		RequestID:       "test-request-123",
		Status:          "verified",
		ConfidenceScore: 0.95,
		Timestamp:       time.Now().Format(time.RFC3339),
		ExpiresAt:       time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		DPID:            "test-dp-001",
		ProcessingTime:  "150ms",
	}

	ctx := context.WithValue(context.Background(), RequestIDKey, "test-request-123")
	status := "SUCCESS"

	// This should not panic and should log the audit entry
	service.LogVerification(ctx, req, response, status)

	// Since we're just logging to console in the current implementation,
	// we can't easily verify the output, but we can ensure it doesn't panic
}

func TestAuditService_LogVerification_WithNilResponse(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Create a test request
	req := models.VerificationRequest{
		RPID:      "test-rp-001",
		UserID:    "user-123",
		ClaimType: "identity_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}

	ctx := context.WithValue(context.Background(), RequestIDKey, "test-request-123")
	status := "ERROR"

	// This should handle nil response gracefully
	service.LogVerification(ctx, req, nil, status)

	// Should not panic
}

func TestAuditService_LogVerification_WithoutRequestID(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Create a test request
	req := models.VerificationRequest{
		RPID:      "test-rp-001",
		UserID:    "user-123",
		ClaimType: "identity_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}

	// Context without request ID
	ctx := context.Background()
	status := "SUCCESS"

	// This should handle missing request ID gracefully
	service.LogVerification(ctx, req, nil, status)

	// Should not panic and should use "unknown" as request ID
}

func TestAuditService_GeneratePrivacyHash(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Create a test request
	req := models.VerificationRequest{
		RPID:      "test-rp-001",
		UserID:    "user-123",
		ClaimType: "identity_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
			"phone": "+1234567890",
		},
	}

	hash := service.generatePrivacyHash(req)

	assert.NotEmpty(t, hash)
	assert.Len(t, hash, 64)                  // SHA-256 hex string length
	assert.Regexp(t, `^[a-f0-9]{64}$`, hash) // Valid hex string

	// Same request should produce same hash
	hash2 := service.generatePrivacyHash(req)
	assert.Equal(t, hash, hash2)

	// Different request should produce different hash
	req2 := models.VerificationRequest{
		RPID:      "test-rp-002", // Different RP
		UserID:    "user-123",
		ClaimType: "identity_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}
	hash3 := service.generatePrivacyHash(req2)
	assert.NotEqual(t, hash, hash3)
}

func TestAuditService_GeneratePrivacyHash_DifferentIdentifiers(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Create requests with different identifier counts
	req1 := models.VerificationRequest{
		RPID:      "test-rp-001",
		UserID:    "user-123",
		ClaimType: "identity_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}

	req2 := models.VerificationRequest{
		RPID:      "test-rp-001",
		UserID:    "user-123",
		ClaimType: "identity_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
			"phone": "+1234567890",
		},
	}

	hash1 := service.generatePrivacyHash(req1)
	hash2 := service.generatePrivacyHash(req2)

	assert.NotEqual(t, hash1, hash2)
}

func TestAuditService_GenerateMerkleProof(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Create a test request
	req := models.VerificationRequest{
		RPID:      "test-rp-001",
		UserID:    "user-123",
		ClaimType: "identity_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}

	// Create a test response
	response := &models.VerificationResponse{
		RequestID:       "test-request-123",
		Status:          "verified",
		ConfidenceScore: 0.95,
		Timestamp:       time.Now().Format(time.RFC3339),
		ExpiresAt:       time.Now().Add(24 * time.Hour).Format(time.RFC3339),
	}

	proof := service.generateMerkleProof(req, response)

	assert.NotEmpty(t, proof)
	assert.Len(t, proof, 64)                  // SHA-256 hex string length
	assert.Regexp(t, `^[a-f0-9]{64}$`, proof) // Valid hex string

	// Same request/response should produce same proof
	proof2 := service.generateMerkleProof(req, response)
	assert.Equal(t, proof, proof2)

	// Different request should produce different proof
	req2 := models.VerificationRequest{
		RPID:      "test-rp-002",
		UserID:    "user-123",
		ClaimType: "identity_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}
	proof3 := service.generateMerkleProof(req2, response)
	assert.NotEqual(t, proof, proof3)
}

func TestAuditService_GenerateMerkleProof_WithNilResponse(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Create a test request
	req := models.VerificationRequest{
		RPID:      "test-rp-001",
		UserID:    "user-123",
		ClaimType: "identity_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}

	proof := service.generateMerkleProof(req, nil)

	assert.NotEmpty(t, proof)
	assert.Len(t, proof, 64)                  // SHA-256 hex string length
	assert.Regexp(t, `^[a-f0-9]{64}$`, proof) // Valid hex string
}

func TestAuditService_LogAuditEntry(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Create a test audit entry
	entry := &models.AuditEntry{
		Timestamp:      time.Now().Format(time.RFC3339),
		RequestID:      "test-request-123",
		RPID:           "test-rp-001",
		ClaimType:      "identity_verification",
		PrivacyHash:    "abc123def456",
		MerkleProof:    "def456ghi789",
		PolicyDecision: "ALLOW",
		Status:         "SUCCESS",
		DPID:           "test-dp-001",
		Metadata: map[string]interface{}{
			"user_id":           "user-123",
			"identifiers_count": 2,
		},
	}

	// This should not panic and should log the entry
	service.logAuditEntry(entry)

	// Since we're just logging to console, we can't easily verify the output,
	// but we can ensure it doesn't panic
}

func TestAuditService_LogAuditEntry_WithComplexMetadata(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Create a test audit entry with complex metadata
	entry := &models.AuditEntry{
		Timestamp:      time.Now().Format(time.RFC3339),
		RequestID:      "test-request-123",
		RPID:           "test-rp-001",
		ClaimType:      "identity_verification",
		PrivacyHash:    "abc123def456",
		MerkleProof:    "def456ghi789",
		PolicyDecision: "ALLOW",
		Status:         "SUCCESS",
		DPID:           "test-dp-001",
		Metadata: map[string]interface{}{
			"user_id":              "user-123",
			"identifiers_count":    2,
			"processing_time_ms":   150,
			"confidence_score":     0.95,
			"verification_methods": []string{"document", "biometric"},
			"nested_data": map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": "value",
				},
			},
		},
	}

	// This should handle complex metadata gracefully
	service.logAuditEntry(entry)

	// Should not panic
}

func TestGetRequestID_WithRequestID(t *testing.T) {
	ctx := context.WithValue(context.Background(), RequestIDKey, "test-request-123")

	requestID := getRequestID(ctx)

	assert.Equal(t, "test-request-123", requestID)
}

func TestGetRequestID_WithoutRequestID(t *testing.T) {
	ctx := context.Background()

	requestID := getRequestID(ctx)

	assert.Equal(t, "unknown", requestID)
}

func TestGetRequestID_WithWrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), RequestIDKey, 123) // Wrong type

	requestID := getRequestID(ctx)

	assert.Equal(t, "unknown", requestID)
}

func TestAuditService_HealthCheck(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	ctx := context.Background()
	err := service.HealthCheck(ctx)

	assert.NoError(t, err)
}

func TestAuditService_LogVerification_CompleteFlow(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Create a comprehensive test request
	req := models.VerificationRequest{
		RPID:      "test-rp-001",
		UserID:    "user-123",
		ClaimType: "identity_verification",
		Identifiers: map[string]string{
			"email":    "test@example.com",
			"phone":    "+1234567890",
			"passport": "A12345678",
		},
	}

	// Create a comprehensive test response
	response := &models.VerificationResponse{
		RequestID:       "test-request-123",
		Status:          "verified",
		ConfidenceScore: 0.95,
		Timestamp:       time.Now().Format(time.RFC3339),
		ExpiresAt:       time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		DPID:            "test-dp-001",
		ProcessingTime:  "150ms",
		Reason:          "User identity verified through multiple factors",
		Evidence:        []string{"document_verified", "biometric_match", "liveness_check"},
	}

	ctx := context.WithValue(context.Background(), RequestIDKey, "test-request-123")
	status := "SUCCESS"

	// This should create a complete audit entry with all fields
	service.LogVerification(ctx, req, response, status)

	// Should not panic and should include all the data in the audit log
}

func TestAuditService_LogVerification_ErrorStatus(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Create a test request
	req := models.VerificationRequest{
		RPID:      "test-rp-001",
		UserID:    "user-123",
		ClaimType: "identity_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}

	ctx := context.WithValue(context.Background(), RequestIDKey, "test-request-123")

	// Test different error statuses
	errorStatuses := []string{
		"ERROR",
		"TIMEOUT",
		"INVALID_REQUEST",
		"DP_UNAVAILABLE",
		"POLICY_DENIED",
	}

	for _, status := range errorStatuses {
		service.LogVerification(ctx, req, nil, status)
		// Should not panic for any error status
	}
}

func TestAuditService_GeneratePrivacyHash_EdgeCases(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Test with empty identifiers
	req1 := models.VerificationRequest{
		RPID:        "test-rp-001",
		UserID:      "user-123",
		ClaimType:   "identity_verification",
		Identifiers: map[string]string{},
	}

	hash1 := service.generatePrivacyHash(req1)
	assert.NotEmpty(t, hash1)

	// Test with empty RPID
	req2 := models.VerificationRequest{
		RPID:      "",
		UserID:    "user-123",
		ClaimType: "identity_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}

	hash2 := service.generatePrivacyHash(req2)
	assert.NotEmpty(t, hash2)

	// Test with empty UserID
	req3 := models.VerificationRequest{
		RPID:      "test-rp-001",
		UserID:    "",
		ClaimType: "identity_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}

	hash3 := service.generatePrivacyHash(req3)
	assert.NotEmpty(t, hash3)

	// All hashes should be different
	assert.NotEqual(t, hash1, hash2)
	assert.NotEqual(t, hash1, hash3)
	assert.NotEqual(t, hash2, hash3)
}

func TestAuditService_GenerateMerkleProof_EdgeCases(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Test with empty request
	req1 := models.VerificationRequest{
		RPID:        "",
		UserID:      "",
		ClaimType:   "",
		Identifiers: map[string]string{},
	}

	proof1 := service.generateMerkleProof(req1, nil)
	assert.NotEmpty(t, proof1)

	// Test with empty response
	req2 := models.VerificationRequest{
		RPID:      "test-rp-001",
		UserID:    "user-123",
		ClaimType: "identity_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}

	emptyResponse := &models.VerificationResponse{
		RequestID: "",
		Status:    "",
		Timestamp: "",
	}

	proof2 := service.generateMerkleProof(req2, emptyResponse)
	assert.NotEmpty(t, proof2)

	// Proofs should be different
	assert.NotEqual(t, proof1, proof2)
}

func TestAuditService_LogVerification_WithAuditReference(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Create a test request
	req := models.VerificationRequest{
		RPID:      "test-rp-001",
		UserID:    "user-123",
		ClaimType: "identity_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
			"phone": "+1234567890",
		},
	}

	// Create a test response
	response := &models.VerificationResponse{
		RequestID:       "test-request-123",
		Status:          "verified",
		ConfidenceScore: 0.95,
		Timestamp:       time.Now().Format(time.RFC3339),
		ExpiresAt:       time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		DPID:            "test-dp-001",
		ProcessingTime:  "150ms",
	}

	ctx := context.WithValue(context.Background(), RequestIDKey, "test-request-123")
	status := "SUCCESS"

	// Test that LogVerification now returns an audit reference
	auditRef := service.LogVerification(ctx, req, response, status)

	require.NotNil(t, auditRef)
	assert.NotEmpty(t, auditRef.AuditEntryID)
	assert.NotEmpty(t, auditRef.MerkleProof)
	assert.NotEmpty(t, auditRef.Timestamp)
	assert.NotEmpty(t, auditRef.Hash)
	assert.Contains(t, auditRef.AuditEntryID, "audit_")
}

func TestAuditService_LogVerification_NilResponse(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Create a test request
	req := models.VerificationRequest{
		RPID:      "test-rp-001",
		UserID:    "user-123",
		ClaimType: "identity_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}

	ctx := context.WithValue(context.Background(), RequestIDKey, "test-request-123")
	status := "ERROR"

	// Test with nil response
	auditRef := service.LogVerification(ctx, req, nil, status)

	require.NotNil(t, auditRef)
	assert.NotEmpty(t, auditRef.AuditEntryID)
	assert.NotEmpty(t, auditRef.MerkleProof)
	assert.NotEmpty(t, auditRef.Timestamp)
	assert.NotEmpty(t, auditRef.Hash)
}

func TestAuditService_LogVerification_NoRequestID(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Create a test request
	req := models.VerificationRequest{
		RPID:      "test-rp-001",
		UserID:    "user-123",
		ClaimType: "identity_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}

	ctx := context.Background() // No request ID
	status := "SUCCESS"

	// Test without request ID
	auditRef := service.LogVerification(ctx, req, nil, status)

	require.NotNil(t, auditRef)
	assert.NotEmpty(t, auditRef.AuditEntryID)
	assert.NotEmpty(t, auditRef.MerkleProof)
	assert.NotEmpty(t, auditRef.Timestamp)
	assert.NotEmpty(t, auditRef.Hash)
}

func TestAuditService_GenerateAuditEntryID(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Create a test request
	req := models.VerificationRequest{
		RPID:      "test-rp-001",
		UserID:    "user-123",
		ClaimType: "identity_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}

	// Test audit entry ID generation
	entryID1 := service.generateAuditEntryID(req, nil)
	entryID2 := service.generateAuditEntryID(req, nil)

	assert.NotEmpty(t, entryID1)
	assert.NotEmpty(t, entryID2)
	assert.Contains(t, entryID1, "audit_")
	assert.Contains(t, entryID2, "audit_")

	// IDs should be different due to timestamp
	assert.NotEqual(t, entryID1, entryID2)
}

func TestAuditService_GenerateAuditEntryID_WithResponse(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Create a test request
	req := models.VerificationRequest{
		RPID:      "test-rp-001",
		UserID:    "user-123",
		ClaimType: "identity_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}

	// Create a test response
	response := &models.VerificationResponse{
		RequestID:       "test-request-123",
		Status:          "verified",
		ConfidenceScore: 0.95,
		Timestamp:       time.Now().Format(time.RFC3339),
		DPID:            "test-dp-001",
	}

	// Test audit entry ID generation with response
	entryID := service.generateAuditEntryID(req, response)

	assert.NotEmpty(t, entryID)
	assert.Contains(t, entryID, "audit_")
}

func TestAuditService_CreateAuditReference(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Create a test audit entry
	entry := &models.AuditEntry{
		Timestamp:      time.Now().Format(time.RFC3339),
		RequestID:      "test-request-123",
		RPID:           "test-rp-001",
		ClaimType:      "identity_verification",
		PrivacyHash:    "test_privacy_hash",
		MerkleProof:    "test_merkle_proof",
		PolicyDecision: "ALLOW",
		Status:         "SUCCESS",
		Metadata: map[string]interface{}{
			"user_id": "user-123",
		},
	}

	auditEntryID := "audit_test123"

	// Test audit reference creation
	auditRef := service.createAuditReference(entry, auditEntryID)

	require.NotNil(t, auditRef)
	assert.Equal(t, auditEntryID, auditRef.AuditEntryID)
	assert.Equal(t, entry.MerkleProof, auditRef.MerkleProof)
	assert.Equal(t, entry.Timestamp, auditRef.Timestamp)
	assert.NotEmpty(t, auditRef.Hash)
}

func TestAuditService_ValidateAuditReference_Success(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Create a valid audit reference
	auditRef := &AuditReference{
		AuditEntryID: "audit_test123",
		MerkleProof:  "test_merkle_proof",
		Timestamp:    time.Now().Format(time.RFC3339),
		Hash:         "test_hash",
	}

	// Test validation
	err := service.ValidateAuditReference(auditRef)

	assert.NoError(t, err)
}

func TestAuditService_ValidateAuditReference_NilReference(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Test validation with nil reference
	err := service.ValidateAuditReference(nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "audit reference is nil")
}

func TestAuditService_ValidateAuditReference_EmptyFields(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Test validation with empty audit entry ID
	auditRef1 := &AuditReference{
		AuditEntryID: "",
		MerkleProof:  "test_merkle_proof",
		Timestamp:    time.Now().Format(time.RFC3339),
		Hash:         "test_hash",
	}

	err1 := service.ValidateAuditReference(auditRef1)
	assert.Error(t, err1)
	assert.Contains(t, err1.Error(), "audit entry ID is empty")

	// Test validation with empty merkle proof
	auditRef2 := &AuditReference{
		AuditEntryID: "audit_test123",
		MerkleProof:  "",
		Timestamp:    time.Now().Format(time.RFC3339),
		Hash:         "test_hash",
	}

	err2 := service.ValidateAuditReference(auditRef2)
	assert.Error(t, err2)
	assert.Contains(t, err2.Error(), "merkle proof is empty")

	// Test validation with empty timestamp
	auditRef3 := &AuditReference{
		AuditEntryID: "audit_test123",
		MerkleProof:  "test_merkle_proof",
		Timestamp:    "",
		Hash:         "test_hash",
	}

	err3 := service.ValidateAuditReference(auditRef3)
	assert.Error(t, err3)
	assert.Contains(t, err3.Error(), "timestamp is empty")

	// Test validation with empty hash
	auditRef4 := &AuditReference{
		AuditEntryID: "audit_test123",
		MerkleProof:  "test_merkle_proof",
		Timestamp:    time.Now().Format(time.RFC3339),
		Hash:         "",
	}

	err4 := service.ValidateAuditReference(auditRef4)
	assert.Error(t, err4)
	assert.Contains(t, err4.Error(), "hash is empty")
}

func TestAuditService_ValidateAuditReference_InvalidTimestamp(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Create audit reference with invalid timestamp
	auditRef := &AuditReference{
		AuditEntryID: "audit_test123",
		MerkleProof:  "test_merkle_proof",
		Timestamp:    "invalid-timestamp",
		Hash:         "test_hash",
	}

	// Test validation
	err := service.ValidateAuditReference(auditRef)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid timestamp format")
}

func TestAuditService_GetAuditReference_Success(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Test retrieving audit reference
	auditRef, err := service.GetAuditReference("audit_test123")

	require.NoError(t, err)
	assert.NotNil(t, auditRef)
	assert.Equal(t, "audit_test123", auditRef.AuditEntryID)
	assert.NotEmpty(t, auditRef.MerkleProof)
	assert.NotEmpty(t, auditRef.Timestamp)
	assert.NotEmpty(t, auditRef.Hash)
}

func TestAuditService_GetAuditReference_EmptyID(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Test retrieving audit reference with empty ID
	auditRef, err := service.GetAuditReference("")

	assert.Error(t, err)
	assert.Nil(t, auditRef)
	assert.Contains(t, err.Error(), "audit entry ID is empty")
}

func TestAuditService_GenerateMerkleProof_WithResponse(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Create a test request
	req := models.VerificationRequest{
		RPID:      "test-rp-001",
		UserID:    "user-123",
		ClaimType: "identity_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}

	// Create a test response
	response := &models.VerificationResponse{
		RequestID:       "test-request-123",
		Status:          "verified",
		ConfidenceScore: 0.95,
		Timestamp:       time.Now().Format(time.RFC3339),
		DPID:            "test-dp-001",
		Verified:        true,
	}

	// Test Merkle proof generation with response
	proof := service.generateMerkleProof(req, response)

	assert.NotEmpty(t, proof)
	// Should be a hex string
	assert.Len(t, proof, 64) // SHA-256 hash is 32 bytes = 64 hex chars
}

func TestAuditService_GenerateMerkleProof_WithoutResponse(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Create a test request
	req := models.VerificationRequest{
		RPID:      "test-rp-001",
		UserID:    "user-123",
		ClaimType: "identity_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}

	// Test Merkle proof generation without response
	proof := service.generateMerkleProof(req, nil)

	assert.NotEmpty(t, proof)
	// Should be a hex string
	assert.Len(t, proof, 64) // SHA-256 hash is 32 bytes = 64 hex chars
}

func TestAuditService_GenerateMerkleProof_DifferentInputs(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Create test requests
	req1 := models.VerificationRequest{
		RPID:      "test-rp-001",
		UserID:    "user-123",
		ClaimType: "identity_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}

	req2 := models.VerificationRequest{
		RPID:      "test-rp-002",
		UserID:    "user-456",
		ClaimType: "age_verification",
		Identifiers: map[string]string{
			"phone": "+1234567890",
		},
	}

	// Test that different inputs produce different proofs
	proof1 := service.generateMerkleProof(req1, nil)
	proof2 := service.generateMerkleProof(req2, nil)

	assert.NotEqual(t, proof1, proof2)
	assert.NotEmpty(t, proof1)
	assert.NotEmpty(t, proof2)
}

func TestAuditService_HealthCheck_Second(t *testing.T) {
	cfg := &config.Config{
		AuditDBURL: "postgres://localhost:5432/test_audit",
	}

	service := NewAuditService(cfg)

	// Test health check
	err := service.HealthCheck(context.Background())

	assert.NoError(t, err)
}
