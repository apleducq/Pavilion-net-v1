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

// AuditService handles audit logging with cryptographic integrity
type AuditService struct {
	config *config.Config
}

// NewAuditService creates a new audit service
func NewAuditService(cfg *config.Config) *AuditService {
	return &AuditService{
		config: cfg,
	}
}

// LogVerification logs a verification request/response for audit purposes
func (s *AuditService) LogVerification(ctx context.Context, req models.VerificationRequest, response *models.VerificationResponse, status string) {
	// Create audit entry
	entry := &models.AuditEntry{
		Timestamp:     time.Now().Format(time.RFC3339),
		RequestID:     getRequestID(ctx),
		RPID:          req.RPID,
		ClaimType:     req.ClaimType,
		PrivacyHash:   s.generatePrivacyHash(req),
		MerkleProof:   s.generateMerkleProof(req, response),
		PolicyDecision: "ALLOW", // TODO: Get from policy service
		Status:        status,
		Metadata: map[string]interface{}{
			"user_id": req.UserID,
			"identifiers_count": len(req.Identifiers),
		},
	}

	// Add DP ID if response exists
	if response != nil {
		entry.DPID = "mock-dp-001" // TODO: Get from response
	}

	// TODO: Send to audit database
	// For now, just log to console
	s.logAuditEntry(entry)
}

// generatePrivacyHash creates a privacy-preserving hash of the request
func (s *AuditService) generatePrivacyHash(req models.VerificationRequest) string {
	// Hash the request without exposing raw PII
	data := fmt.Sprintf("%s:%s:%s:%d", req.RPID, req.UserID, req.ClaimType, len(req.Identifiers))
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// generateMerkleProof creates a simple Merkle proof for the audit entry
func (s *AuditService) generateMerkleProof(req models.VerificationRequest, response *models.VerificationResponse) string {
	// TODO: Implement proper Merkle tree
	// For now, create a simple hash-based proof
	data := fmt.Sprintf("%s:%s:%s", req.RPID, req.ClaimType, time.Now().Format(time.RFC3339))
	hash := sha256.Sum256([]byte(data))
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
	if requestID, ok := ctx.Value("request_id").(string); ok {
		return requestID
	}
	return "unknown"
}

// HealthCheck checks if the audit service is healthy
func (s *AuditService) HealthCheck(ctx context.Context) error {
	// TODO: Implement actual health check
	// For now, always return healthy
	return nil
} 