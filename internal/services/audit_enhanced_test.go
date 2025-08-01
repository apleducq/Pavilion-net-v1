package services

import (
	"context"
	"testing"
	"time"

	"github.com/pavilion-trust/core-broker/internal/config"
	"github.com/pavilion-trust/core-broker/internal/models"
)

func TestGetPolicyDecision(t *testing.T) {
	service := NewAuditService(&config.Config{})
	ctx := context.Background()

	// Test student verification
	req := models.VerificationRequest{
		ClaimType: "student_verification",
	}
	decision := service.getPolicyDecision(ctx, req)
	if decision != "ALLOW" {
		t.Errorf("Expected ALLOW for student verification, got %s", decision)
	}

	// Test employee verification
	req.ClaimType = "employee_verification"
	decision = service.getPolicyDecision(ctx, req)
	if decision != "ALLOW" {
		t.Errorf("Expected ALLOW for employee verification, got %s", decision)
	}

	// Test age verification
	req.ClaimType = "age_verification"
	decision = service.getPolicyDecision(ctx, req)
	if decision != "ALLOW" {
		t.Errorf("Expected ALLOW for age verification, got %s", decision)
	}

	// Test address verification
	req.ClaimType = "address_verification"
	decision = service.getPolicyDecision(ctx, req)
	if decision != "ALLOW" {
		t.Errorf("Expected ALLOW for address verification, got %s", decision)
	}

	// Test unknown claim type
	req.ClaimType = "unknown_verification"
	decision = service.getPolicyDecision(ctx, req)
	if decision != "DENY" {
		t.Errorf("Expected DENY for unknown verification, got %s", decision)
	}
}

func TestCreateAuditMetadata(t *testing.T) {
	service := NewAuditService(&config.Config{})

	req := models.VerificationRequest{
		RPID:      "rp-001",
		UserID:    "user-123",
		ClaimType: "student_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
			"name":  "John Doe",
		},
		Metadata: map[string]interface{}{
			"source": "web",
			"ip":     "192.168.1.1",
		},
	}

	response := &models.VerificationResponse{
		VerificationID:  "verif-456",
		Status:          "verified",
		Verified:        true,
		ConfidenceScore: 0.95,
		DPID:            "dp-001",
		ProcessingTime:  "150ms",
	}

	auditEntryID := "audit-789"

	metadata := service.createAuditMetadata(req, response, auditEntryID)

	// Test basic fields
	if metadata["user_id"] != "user-123" {
		t.Error("Expected user_id to be set")
	}

	if metadata["identifiers_count"] != 2 {
		t.Error("Expected identifiers_count to be set")
	}

	if metadata["audit_entry_id"] != "audit-789" {
		t.Error("Expected audit_entry_id to be set")
	}

	if metadata["claim_type"] != "student_verification" {
		t.Error("Expected claim_type to be set")
	}

	if metadata["rp_id"] != "rp-001" {
		t.Error("Expected rp_id to be set")
	}

	// Test identifier types
	identifierTypes, ok := metadata["identifier_types"].([]string)
	if !ok {
		t.Fatal("Expected identifier_types to be a slice")
	}

	if len(identifierTypes) != 2 {
		t.Errorf("Expected 2 identifier types, got %d", len(identifierTypes))
	}

	// Test response metadata
	if metadata["verification_id"] != "verif-456" {
		t.Error("Expected verification_id to be set")
	}

	if metadata["verified"] != true {
		t.Error("Expected verified to be set")
	}

	if metadata["confidence_score"] != 0.95 {
		t.Error("Expected confidence_score to be set")
	}

	if metadata["dp_id"] != "dp-001" {
		t.Error("Expected dp_id to be set")
	}

	if metadata["status"] != "verified" {
		t.Error("Expected status to be set")
	}

	if metadata["processing_time"] != "150ms" {
		t.Error("Expected processing_time to be set")
	}

	// Test request metadata
	if metadata["request_source"] != "web" {
		t.Error("Expected request_source to be set")
	}

	if metadata["request_ip"] != "192.168.1.1" {
		t.Error("Expected request_ip to be set")
	}

	// Test with nil response
	metadata = service.createAuditMetadata(req, nil, auditEntryID)
	if metadata["verification_id"] != nil {
		t.Error("Expected verification_id to be nil for nil response")
	}
}

func TestGetNextSequenceNumber(t *testing.T) {
	service := NewAuditService(&config.Config{})

	// Test sequence number generation
	seq1 := service.getNextSequenceNumber()
	seq2 := service.getNextSequenceNumber()

	if seq1 == 0 {
		t.Error("Expected non-zero sequence number")
	}

	if seq2 == 0 {
		t.Error("Expected non-zero sequence number")
	}

	// Sequence numbers should be different (based on timestamp)
	if seq1 == seq2 {
		t.Error("Expected different sequence numbers")
	}
}

func TestLogPolicyDecision(t *testing.T) {
	service := NewAuditService(&config.Config{})
	ctx := context.WithValue(context.Background(), "request_id", "req-123")

	req := models.VerificationRequest{
		RPID:      "rp-001",
		UserID:    "user-123",
		ClaimType: "student_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}

	// This should not panic and should log the entry
	service.LogPolicyDecision(ctx, req, "ALLOW", "User is eligible for student verification")
}

func TestLogPrivacyHash(t *testing.T) {
	service := NewAuditService(&config.Config{})
	ctx := context.WithValue(context.Background(), "request_id", "req-123")

	req := models.VerificationRequest{
		RPID:      "rp-001",
		UserID:    "user-123",
		ClaimType: "student_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}

	privacyHash := "privacy_hash_123"

	// This should not panic and should log the entry
	service.LogPrivacyHash(ctx, req, privacyHash)
}

func TestEnhancedLogVerification(t *testing.T) {
	service := NewAuditService(&config.Config{})
	ctx := context.WithValue(context.Background(), "request_id", "req-123")

	req := models.VerificationRequest{
		RPID:      "rp-001",
		UserID:    "user-123",
		ClaimType: "student_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
			"name":  "John Doe",
		},
		Metadata: map[string]interface{}{
			"source": "mobile",
		},
	}

	response := &models.VerificationResponse{
		VerificationID:  "verif-456",
		Status:          "verified",
		Verified:        true,
		ConfidenceScore: 0.95,
		DPID:            "dp-001",
		ProcessingTime:  "150ms",
	}

	// Test enhanced logging
	auditRef := service.LogVerification(ctx, req, response, "SUCCESS")

	if auditRef == nil {
		t.Fatal("Expected audit reference to be returned")
	}

	if auditRef.AuditEntryID == "" {
		t.Error("Expected audit entry ID to be set")
	}

	if auditRef.MerkleProof == "" {
		t.Error("Expected Merkle proof to be set")
	}

	if auditRef.Timestamp == "" {
		t.Error("Expected timestamp to be set")
	}

	if auditRef.Hash == "" {
		t.Error("Expected hash to be set")
	}

	// Test with nil response
	auditRef2 := service.LogVerification(ctx, req, nil, "ERROR")

	if auditRef2 == nil {
		t.Fatal("Expected audit reference to be returned even with nil response")
	}

	if auditRef2.AuditEntryID == "" {
		t.Error("Expected audit entry ID to be set")
	}
}

func TestEnhancedHealthCheck(t *testing.T) {
	service := NewAuditService(&config.Config{})
	ctx := context.Background()

	err := service.HealthCheck(ctx)
	if err != nil {
		t.Errorf("Expected health check to pass: %v", err)
	}
}

func TestAuditEntryStructure(t *testing.T) {
	service := NewAuditService(&config.Config{})
	ctx := context.WithValue(context.Background(), "request_id", "req-123")

	req := models.VerificationRequest{
		RPID:      "rp-001",
		UserID:    "user-123",
		ClaimType: "student_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}

	response := &models.VerificationResponse{
		VerificationID:  "verif-456",
		Status:          "verified",
		Verified:        true,
		ConfidenceScore: 0.95,
		DPID:            "dp-001",
	}

	// Test that the enhanced logging creates proper audit entry structure
	auditRef := service.LogVerification(ctx, req, response, "SUCCESS")

	if auditRef == nil {
		t.Fatal("Expected audit reference")
	}

	// Verify the audit reference structure
	if auditRef.AuditEntryID == "" {
		t.Error("Expected audit entry ID")
	}

	if auditRef.MerkleProof == "" {
		t.Error("Expected Merkle proof")
	}

	if auditRef.Timestamp == "" {
		t.Error("Expected timestamp")
	}

	if auditRef.Hash == "" {
		t.Error("Expected hash")
	}

	// Test validation
	err := service.ValidateAuditReference(auditRef)
	if err != nil {
		t.Errorf("Expected valid audit reference: %v", err)
	}
}

func TestPolicyDecisionLogging(t *testing.T) {
	service := NewAuditService(&config.Config{})
	ctx := context.WithValue(context.Background(), "request_id", "req-123")

	req := models.VerificationRequest{
		RPID:      "rp-001",
		UserID:    "user-123",
		ClaimType: "student_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
		},
	}

	// Test policy decision logging
	service.LogPolicyDecision(ctx, req, "ALLOW", "User meets eligibility criteria")

	// Test privacy hash logging
	service.LogPrivacyHash(ctx, req, "privacy_hash_123")
}

func TestSequenceNumberGeneration(t *testing.T) {
	service := NewAuditService(&config.Config{})

	// Test that sequence numbers are generated
	seq1 := service.getNextSequenceNumber()
	time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	seq2 := service.getNextSequenceNumber()

	if seq1 == 0 {
		t.Error("Expected non-zero sequence number")
	}

	if seq2 == 0 {
		t.Error("Expected non-zero sequence number")
	}

	if seq1 == seq2 {
		t.Error("Expected different sequence numbers")
	}
}

func TestMetadataCompleteness(t *testing.T) {
	service := NewAuditService(&config.Config{})

	req := models.VerificationRequest{
		RPID:      "rp-001",
		UserID:    "user-123",
		ClaimType: "student_verification",
		Identifiers: map[string]string{
			"email": "test@example.com",
			"phone": "123-456-7890",
		},
		Metadata: map[string]interface{}{
			"source": "web",
			"ip":     "192.168.1.1",
		},
	}

	response := &models.VerificationResponse{
		VerificationID:  "verif-456",
		Status:          "verified",
		Verified:        true,
		ConfidenceScore: 0.95,
		DPID:            "dp-001",
		ProcessingTime:  "150ms",
	}

	metadata := service.createAuditMetadata(req, response, "audit-789")

	// Test all expected fields are present
	expectedFields := []string{
		"user_id", "identifiers_count", "audit_entry_id", "claim_type", "rp_id", "timestamp",
		"identifier_types", "verification_id", "verified", "confidence_score", "dp_id", "status", "processing_time",
		"request_source", "request_ip",
	}

	for _, field := range expectedFields {
		if metadata[field] == nil {
			t.Errorf("Expected field %s to be present", field)
		}
	}

	// Test identifier types
	identifierTypes, ok := metadata["identifier_types"].([]string)
	if !ok {
		t.Fatal("Expected identifier_types to be a slice")
	}

	if len(identifierTypes) != 2 {
		t.Errorf("Expected 2 identifier types, got %d", len(identifierTypes))
	}

	// Check for specific identifier types
	hasEmail := false
	hasPhone := false
	for _, idType := range identifierTypes {
		if idType == "email" {
			hasEmail = true
		}
		if idType == "phone" {
			hasPhone = true
		}
	}

	if !hasEmail {
		t.Error("Expected email identifier type")
	}

	if !hasPhone {
		t.Error("Expected phone identifier type")
	}
} 